//go:build !windows

package gitx

// securePrivateDir has nothing to do: os.MkdirTemp makes the directory
// 0700, and what is made in it is 0600 or 0700.
func securePrivateDir(string) error { return nil }
