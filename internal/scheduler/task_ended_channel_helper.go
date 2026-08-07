package scheduler

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// mergeWatchers returns a channel representing any of the input channels closing.
// The buffer-1 output and default-discard send prevent simultaneous watcher signals
// from blocking fan-in goroutines. The output is deliberately never closed because
// the caller consumes one termination signal rather than channel lifecycle state.
// With no inputs, the fan-in remains cancellable by waiting on ctx.Done(). This
// follows RESEARCH.md Pattern 3 / Pitfall 1 and exits its outer goroutine on cancel.
//
// once 守门：原实现只靠 select+default discard,理论上第一次 send 与 reader 消费之间
// 还能塞入第二次 send,TestSchedulerChannelMerge_Race 在 race detector 下
// -count=10 约 1/10 概率 fail。改用 sync.Once 保证 out 仅被写一次。
func mergeWatchers(ctx context.Context, chans []<-chan struct{}, logger *zap.Logger) <-chan struct{} {
	out := make(chan struct{}, 1)
	var once sync.Once
	go func() {
		for _, ch := range chans {
			if ch == nil {
				continue
			}
			go func(c <-chan struct{}) {
				select {
				case <-c:
					once.Do(func() {
						select {
						case out <- struct{}{}:
						default:
						}
					})
				case <-ctx.Done():
				}
			}(ch)
		}
		<-ctx.Done()
	}()
	return out
}
