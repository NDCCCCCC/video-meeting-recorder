package main

import (
	"context"
	"fmt"

	huaweiapi "github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/recorder"
)

// huaweiManagerStateAdapter 把 *huaweiapi.Manager 桥接到 recorder.HuaweiStateClient
// 接口，供 Phase 25 SCHED-01 在 cmd/server/app.go:1149 通过 coordinator.SetHuaweiCli
// 注入。recorder.HuaweiStateClient 接口设计预期 *huaweiapi.Client（已实现
// GetConferenceState），但 app 层只有 Manager；本 adapter 在单设备部署场景下取
// 已注册的第一个 HuaweiClient 转发 GetConferenceState。
//
// 多设备场景：caller 应改用基于具体 configID 的 adapter（构造期固定 configID →
// 直接调 Manager.GetClient(ctx, configID).GetConferenceState），避免跨终端
// 状态错配。Phase 25 当前为单设备假设，不引入额外抽象。
type huaweiManagerStateAdapter struct {
	mgr *huaweiapi.Manager
}

// GetConferenceState 实现 recorder.HuaweiStateClient 接口。
// 失败语义：
//   - 无 client 注册：返回 error，Phase 24 ActivityWatcher 走
//     huaweiConsecFailures 累加路径 → 触发 HuaweiFailureThreshold 降级。
//   - 拿到 client 后 *huaweiapi.Client.GetConferenceState 失败：error 透传，
//     同样的降级路径。
func (a *huaweiManagerStateAdapter) GetConferenceState(ctx context.Context) (*huaweiapi.ConferenceState, error) {
	client, ok := a.mgr.GetFirstRegisteredClient()
	if !ok {
		return nil, fmt.Errorf("huawei manager has no registered clients (H signal unavailable)")
	}
	return client.GetConferenceState(ctx)
}

// Compile-time check: huaweiManagerStateAdapter 必须实现 recorder.HuaweiStateClient。
var _ recorder.HuaweiStateClient = (*huaweiManagerStateAdapter)(nil)