//go:build linux || darwin

package wp25

import "time"

// stopwatch starts a measurement; the returned function gives the time
// since.
func stopwatch() func() time.Duration {
	t0 := time.Now()
	return func() time.Duration { return time.Since(t0) }
}
