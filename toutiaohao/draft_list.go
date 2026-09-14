package toutiaohao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/example/toutiaohao-mcp-server/cookies"
)

// DraftSummary 草稿箱条目。真实端点是 agw/creator_center/draft_list?type=2
// （2026-09 实测；上游代码使用的 agw/article/list/?status=draft 已与平台不一致，
// 该端点永远返回空列表）。
type DraftSummary struct {
	ArticleID string
	Title     string
	Raw       map[string]interface{}
}

// ListCreatorDrafts 拉取当前账号草稿箱（创作者中心口径）。
func ListCreatorDrafts(ctx context.Context, cookieStore cookies.Cookier) ([]DraftSummary, error) {
	url := "https://mp.toutiao.com/mp/agw/creator_center/draft_list?type=2&count=20&app_id=1231"
	body, err := doAuthenticatedGet(ctx, url, cookieStore)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code    int                      `json:"code"`
		List    []map[string]interface{} `json:"draft_list"`
		Message string                   `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("draft_list api code=%d message=%s", resp.Code, resp.Message)
	}
	out := make([]DraftSummary, 0, len(resp.List))
	for _, item := range resp.List {
		ds := DraftSummary{Raw: item}
		for _, k := range []string{"article_id", "id", "group_id", "item_id"} {
			if v, ok := item[k]; ok && v != nil {
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s != "" && s != "0" {
					ds.ArticleID = s
					break
				}
			}
		}
		if v, ok := item["title"].(string); ok {
			ds.Title = v
		}
		out = append(out, ds)
	}
	return out, nil
}
