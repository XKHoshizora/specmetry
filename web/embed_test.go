//go:build !nowebui

package web

import (
	"io/fs"
	"testing"
)

func TestAssetsContainIndex(t *testing.T) {
	assets, err := Assets()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(assets, "index.html"); err != nil {
		t.Fatalf("index.html not embedded: %v", err)
	}
}
