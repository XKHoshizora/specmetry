// Package buildinfo reports the version of the running binary.
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

// Version is overridden for release builds with
//
//	-ldflags "-X github.com/XKHoshizora/specmetry/internal/buildinfo.Version=0.1.0"
//
// It is a semantic version without the leading "v".
var Version = "0.0.0-dev"

// Info describes the running binary.
type Info struct {
	Version   string
	Commit    string
	Modified  bool
	GoVersion string
	Platform  string
}

// Get returns build information, using VCS metadata embedded by the Go
// toolchain when available.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    "unknown",
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				info.Commit = s.Value
			case "vcs.modified":
				info.Modified = s.Value == "true"
			}
		}
	}
	return info
}
