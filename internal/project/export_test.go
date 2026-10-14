package project

import "testing"

// SetLink replaces the hard link that publishes the marker for the rest of
// the test, to simulate a file system on which it fails.
func SetLink(t testing.TB, link func(oldname, newname string) error) {
	t.Helper()
	old := linkFile
	linkFile = link
	t.Cleanup(func() { linkFile = old })
}
