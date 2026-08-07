package recorder

import (
	"context"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
)

// HuaweiStateClient 是 huawei.Client.GetConferenceState 方法的接口抽象。
//
// 抽接口的目的：
//   - testability：Phase 24 ActivityWatcher 必须可注入 fake 实现以驱动
//     确定性状态机断言；直接依赖 *huawei.Client 不可 fake（也无 New 钩子）
//   - 解耦具体类型：Phase 25 scheduler 不应绑定到 *huawei.Client；watcher
//     持有 interface 即可在调度层切换 Huawei client 实现或 mock。
//
// 隐式满足保证：*huawei.Client 已实现 GetConferenceState(ctx) (*ConferenceState, error)，
// 编译期由 Go 接口满足规则保证本接口可由 *huawei.Client 直接满足，无需额外 adapter 代码。
// 见 internal/huawei/client.go:861-879 方法签名。
type HuaweiStateClient interface {
	GetConferenceState(ctx context.Context) (*huawei.ConferenceState, error)
}
