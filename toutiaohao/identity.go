package toutiaohao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/toutiaohao-mcp-server/cookies"
)

// AccountIdentity 当前登录创作者身份。UID 是稳定主键；昵称仅作辅助核对，
// 平台可能允许改名，不得作为身份判据。
type AccountIdentity struct {
	UID  string `json:"creator_uid"`
	Name string `json:"creator_name"`
}

// GetAccountIdentity 通过 creator_center/user_info 读取当前登录账号身份。
// 该端点是创作后台自身加载主页使用的账号接口，随登录态变化，是最直接的
// “当前会话是谁”数据源。
func GetAccountIdentity(ctx context.Context, cookieStore cookies.Cookier) (*AccountIdentity, error) {
	url := "https://mp.toutiao.com/mp/agw/creator_center/user_info?app_id=1231"
	body, err := doAuthenticatedGet(ctx, url, cookieStore)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code       int    `json:"code"`
		UserIDStr  string `json:"user_id_str"`
		UserID     int64  `json:"user_id"`
		ScreenName string `json:"screen_name"`
		Name       string `json:"name"`
		UserName   string `json:"user_name"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	uid := strings.TrimSpace(resp.UserIDStr)
	if uid == "" && resp.UserID != 0 {
		uid = fmt.Sprintf("%d", resp.UserID)
	}
	if uid == "" {
		return nil, fmt.Errorf("user_info response has no user id")
	}
	name := resp.ScreenName
	if name == "" {
		name = resp.Name
	}
	if name == "" {
		name = resp.UserName
	}
	return &AccountIdentity{UID: uid, Name: name}, nil
}
