package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSchedulerChannelMerge_FanIn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chans := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	merged := mergeWatchers(ctx, []<-chan struct{}{chans[0], chans[1], chans[2]}, nil)
	close(chans[1])
	select {
	case <-merged:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("merged channel did not receive watcher signal")
	}
}

func TestSchedulerChannelMerge_Race(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const count = 8
	chans := make([]chan struct{}, count)
	inputs := make([]<-chan struct{}, count)
	for i := range chans {
		chans[i] = make(chan struct{})
		inputs[i] = chans[i]
	}
	merged := mergeWatchers(ctx, inputs, nil)
	var wg sync.WaitGroup
	wg.Add(count)
	for _, ch := range chans {
		go func(c chan struct{}) {
			defer wg.Done()
			close(c)
		}(ch)
	}
	wg.Wait()
	select {
	case <-merged:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("merged channel did not receive concurrent watcher signal")
	}
	select {
	case <-merged:
		t.Fatal("merged channel delivered duplicate signal")
	default:
	}
}

var _ = sync.Once{}
