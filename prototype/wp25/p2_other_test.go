//go:build !linux

package wp25

// P-2 is Linux's (docs/wp2.5-prototype.md §14.4).
func p2Helper(string, []string) (int, bool) { return 0, false }
