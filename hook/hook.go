package hook

import (
	"context"
)

type DpHook struct{}

func (h *DpHook) OnAfterLoadConfig(ctx context.Context) error {
	// 加载配置之后
	return nil
}

func (h *DpHook) OnShutdown(ctx context.Context) error {
	// 服务停止之前
	return nil
}
