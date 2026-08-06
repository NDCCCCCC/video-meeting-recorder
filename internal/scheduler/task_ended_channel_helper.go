package scheduler

import (
	"context"

	"go.uber.org/zap"
)

// mergeWatchers returns a channel representing any of the input channels closing.
// The buffer-1 output and default-discard send prevent simultaneous watcher signals
// from blocking fan-in goroutines. The output is deliberately never closed because
// the caller consumes one termination signal rather than channel lifecycle state.
// With no inputs, the fan-in remains cancellable by waiting on ctx.Done(). This
// follows RESEARCH.md Pattern 3 / Pitfall 1 and exits its outer goroutine on cancel.
func mergeWatchers(ctx context.Context, chans []<-chan struct{}, logger *zap.Logger) <-chan struct{} {
	out := make(chan struct{}, 1)
	go func() {
		for _, ch := range chans {
			if ch == nil {
				continue
			}
			go func(c <-chan struct{}) {
				select {
				case <-c:
					select {
					case out <- struct{}{}:
					default:
					}
				case <-ctx.Done():
				}
			}(ch)
		}
		<-ctx.Done()
	}()
	return out
}
