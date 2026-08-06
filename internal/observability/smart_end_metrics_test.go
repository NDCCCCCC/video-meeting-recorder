package observability

import (
	"testing"
)

// TestSmartEndMetrics_RecordExtend 校验 RecordSmartExtend 累加 3 次后
// SmartExtendTotal == 3。计数器是进程级全局,测试入口用 ResetForTest 清零,
// 出口用 t.Cleanup 再清零以避免污染兄弟测试。
func TestSmartEndMetrics_RecordExtend(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	for i := 0; i < 3; i++ {
		RecordSmartExtend()
	}
	if got := SmartExtendTotal(); got != 3 {
		t.Fatalf("SmartExtendTotal after 3 RecordSmartExtend = %d, want 3", got)
	}
}

// TestSmartEndMetrics_RecordEarlyEnd 校验 RecordSmartEarlyEnd 累加 5 次后
// SmartEarlyEndTotal == 5。
func TestSmartEndMetrics_RecordEarlyEnd(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	for i := 0; i < 5; i++ {
		RecordSmartEarlyEnd()
	}
	if got := SmartEarlyEndTotal(); got != 5 {
		t.Fatalf("SmartEarlyEndTotal after 5 RecordSmartEarlyEnd = %d, want 5", got)
	}
}

// TestSmartEndMetrics_RecordWatcherDegraded 校验 RecordWatcherDegraded
// 累加 2 次后 WatcherDegradedTotal == 2。
func TestSmartEndMetrics_RecordWatcherDegraded(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	for i := 0; i < 2; i++ {
		RecordWatcherDegraded()
	}
	if got := WatcherDegradedTotal(); got != 2 {
		t.Fatalf("WatcherDegradedTotal after 2 RecordWatcherDegraded = %d, want 2", got)
	}
}

// TestSmartEndMetrics_ResetForTest 校验所有 3 个 Record* 函数累加后,一次
// ResetForTest 把所有 getter 拉回 0。验证 reset 是全量清零而非仅清零部分。
func TestSmartEndMetrics_ResetForTest(t *testing.T) {
	t.Cleanup(ResetForTest)

	RecordSmartExtend()
	RecordSmartExtend()
	RecordSmartEarlyEnd()
	RecordWatcherDegraded()

	if got := SmartExtendTotal(); got != 2 {
		t.Fatalf("pre-reset SmartExtendTotal = %d, want 2", got)
	}
	if got := SmartEarlyEndTotal(); got != 1 {
		t.Fatalf("pre-reset SmartEarlyEndTotal = %d, want 1", got)
	}
	if got := WatcherDegradedTotal(); got != 1 {
		t.Fatalf("pre-reset WatcherDegradedTotal = %d, want 1", got)
	}

	ResetForTest()

	if got := SmartExtendTotal(); got != 0 {
		t.Fatalf("post-reset SmartExtendTotal = %d, want 0", got)
	}
	if got := SmartEarlyEndTotal(); got != 0 {
		t.Fatalf("post-reset SmartEarlyEndTotal = %d, want 0", got)
	}
	if got := WatcherDegradedTotal(); got != 0 {
		t.Fatalf("post-reset WatcherDegradedTotal = %d, want 0", got)
	}
}