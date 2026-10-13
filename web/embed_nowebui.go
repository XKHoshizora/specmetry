//go:build nowebui

package web

import (
	"errors"
	"io/fs"
)

// Embedded reports whether the dashboard is compiled into this binary.
func Embedded() bool { return false }

// Assets reports that this binary was built without the dashboard.
func Assets() (fs.FS, error) {
	return nil, errors.New("dashboard not embedded (built with -tags nowebui)")
}
