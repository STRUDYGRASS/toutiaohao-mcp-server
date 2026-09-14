package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/example/toutiaohao-mcp-server/toutiaohao"
	log "github.com/sirupsen/logrus"
)

// AccountMismatchError 当前登录账号与绑定账号不一致。所有写操作必须在上游
// 浏览器动作执行之前拒绝，不允许自动切换账号。
type AccountMismatchError struct {
	ExpectedUID  string
	ExpectedName string
	ActualUID    string
	ActualName   string
	Action       string
}

func (e *AccountMismatchError) Error() string {
	return fmt.Sprintf("ACCOUNT_MISMATCH: action=%s expected creator_uid=%s (%s) but logged in as %s (%s); refusing to write, never auto-switch accounts",
		e.Action, e.ExpectedUID, e.ExpectedName, e.ActualUID, e.ActualName)
}

// identityBinding 首次绑定时落盘的期望身份。之后每次写操作前与实际登录身份比对。
type identityBinding struct {
	CreatorUID  string    `json:"creator_uid"`
	CreatorName string    `json:"creator_name"`
	BoundAt     time.Time `json:"bound_at"`
}

var (
	identityBindingMu sync.Mutex
)

// identityBindingPath 绑定文件默认与 cookies.json 同目录；可用环境变量改写（测试用）。
func identityBindingPath() string {
	if p := os.Getenv("TOUTIAO_IDENTITY_BINDING_PATH"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(cookieBindingRef()), "identity_binding.json")
}

// cookieBindingRef 返回 cookies.json 的绝对路径（绑定文件与它同目录）。
func cookieBindingRef() string {
	p := os.Getenv("TOUTIAOHAO_COOKIES_PATH")
	if p == "" {
		p = "cookies.json"
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// expectedCreatorUID 期望身份来源：环境变量 TOUTIAO_EXPECTED_CREATOR_UID 优先，
// 否则读绑定文件。两者都没有 → 返回空（写操作将 fail-closed 拒绝）。
func expectedCreatorUID() (string, string, error) {
	if env := strings.TrimSpace(os.Getenv("TOUTIAO_EXPECTED_CREATOR_UID")); env != "" {
		return env, "(from env)", nil
	}
	data, err := os.ReadFile(identityBindingPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", err
	}
	var b identityBinding
	if err := json.Unmarshal(data, &b); err != nil {
		return "", "", fmt.Errorf("invalid identity binding file %s: %w", identityBindingPath(), err)
	}
	return strings.TrimSpace(b.CreatorUID), b.CreatorName, nil
}

// GetAccountIdentity 读取当前登录账号身份（uid 稳定主键）。
func (s *ToutiaoService) GetAccountIdentity(ctx context.Context) (*toutiaohao.AccountIdentity, error) {
	return toutiaohao.GetAccountIdentity(ctx, s.cookieStore)
}

// BindAccountIdentity 显式绑定当前登录账号为期望写操作对象。绑定即落盘，
// 之后所有写操作都要求实际登录账号与之一致。
func (s *ToutiaoService) BindAccountIdentity(ctx context.Context) (*toutiaohao.AccountIdentity, error) {
	idn, err := s.GetAccountIdentity(ctx)
	if err != nil {
		return nil, err
	}
	identityBindingMu.Lock()
	defer identityBindingMu.Unlock()
	b := identityBinding{CreatorUID: idn.UID, CreatorName: idn.Name, BoundAt: time.Now()}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(identityBindingPath(), data, 0600); err != nil {
		return nil, err
	}
	log.Infof("Identity bound: creator_uid=%s name=%q -> %s", idn.UID, idn.Name, identityBindingPath())
	return idn, nil
}

// guardWriteOperation 所有写操作（publish/update/delete/draft/reply）在启动浏览器
// 之前必须调用。无期望身份 → fail-closed；身份不一致 → AccountMismatchError。
func (s *ToutiaoService) guardWriteOperation(ctx context.Context, action string) (*toutiaohao.AccountIdentity, error) {
	expectedUID, expectedName, err := expectedCreatorUID()
	if err != nil {
		return nil, err
	}
	if expectedUID == "" {
		return nil, fmt.Errorf("ACCOUNT_UNBOUND: action=%s refused: no identity binding found (set TOUTIAO_EXPECTED_CREATOR_UID or POST /api/v1/account/bind first); writes are fail-closed", action)
	}
	actual, err := s.GetAccountIdentity(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot verify account identity for action=%s: %w", action, err)
	}
	if actual.UID != expectedUID {
		return actual, &AccountMismatchError{
			ExpectedUID: expectedUID, ExpectedName: expectedName,
			ActualUID: actual.UID, ActualName: actual.Name,
			Action: action,
		}
	}
	log.Infof("Identity guard OK: action=%s creator_uid=%s name=%q", action, actual.UID, actual.Name)
	return actual, nil
}

// isAccountMismatch 判断 err 是否为身份不一致（供 API 层映射状态码/响应）。
func isAccountMismatch(err error) bool {
	var mm *AccountMismatchError
	return errors.As(err, &mm)
}
