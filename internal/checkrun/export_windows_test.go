package checkrun

import (
	"testing"

	"golang.org/x/sys/windows"
)

// OnHandle runs f with finalPath's handle of the path p just before
// finalPath closes it.
func OnHandle(t testing.TB, f func(p string, h windows.Handle)) {
	testHookHandle = f
	t.Cleanup(func() { testHookHandle = nil })
}
