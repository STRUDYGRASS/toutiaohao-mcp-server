package toutiaohao

import (
	"os"
	"testing"
)

// requireIntegrationGate 是所有 *Manual 集成测试的统一门禁。
// 这些测试会启动真实浏览器、复用真实登录态甚至产生真实发布，
// 不能作为 `go test ./...` 的默认行为（会挂起等待扫码/人工介入）。
// 只有显式设置 TOUTIAO_INTEGRATION=1 时才执行；未设置时一律 Skip，
// 保证默认单测可以无人值守跑完。
func requireIntegrationGate(t *testing.T) {
	t.Helper()
	if os.Getenv("TOUTIAO_INTEGRATION") != "1" {
		t.Skip("skip manual integration test: set TOUTIAO_INTEGRATION=1 to run it")
	}
}
