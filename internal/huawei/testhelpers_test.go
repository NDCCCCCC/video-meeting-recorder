package huawei

import (
	"runtime"

	"go.uber.org/zap"
)

// zapNopForTest 返回一个禁用日志的 zap logger（测试用）。
func zapNopForTest() *zap.Logger {
	return zap.NewNop()
}

// runtimeNumGoroutine 返回当前活跃 goroutine 数量（测试用）。
func runtimeNumGoroutine() int {
	return runtime.NumGoroutine()
}
