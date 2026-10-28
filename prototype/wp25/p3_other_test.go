//go:build !darwin

package wp25

// P-3 is macOS's (docs/wp2.5-prototype.md §15); elsewhere WP25META has no
// extra fields.
func metaExtra() map[string]string { return nil }
