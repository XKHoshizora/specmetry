//go:build !nowebui

// Package web holds the built dashboard (web/dist) that is embedded into
// the specmetry binary. Build the frontend first (`npm run build` in web/),
// or build Go with `-tags nowebui` to leave it out.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Embedded reports whether the dashboard is compiled into this binary.
func Embedded() bool { return true }

// Assets returns the dashboard files rooted at dist/.
func Assets() (fs.FS, error) { return fs.Sub(dist, "dist") }
