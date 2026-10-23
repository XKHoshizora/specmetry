//go:build windows

package wp25

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Go's monotonic clock on Windows advances with the clock interrupt, so a
// latency below one tick reads as 0; the stopwatch uses the performance
// counter instead.
var (
	kernel32     = windows.NewLazySystemDLL("kernel32.dll")
	procQPC      = kernel32.NewProc("QueryPerformanceCounter")
	procQPF      = kernel32.NewProc("QueryPerformanceFrequency")
	qpcFrequency = func() int64 {
		var f int64
		_, _, _ = procQPF.Call(uintptr(unsafe.Pointer(&f)))
		return f
	}()
)

func qpc() int64 {
	var c int64
	_, _, _ = procQPC.Call(uintptr(unsafe.Pointer(&c)))
	return c
}

// stopwatch starts a measurement; the returned function gives the time
// since.
func stopwatch() func() time.Duration {
	t0 := qpc()
	return func() time.Duration {
		return time.Duration((qpc() - t0) * int64(time.Second) / qpcFrequency)
	}
}
