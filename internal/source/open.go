package source

import (
	"errors"
	"os"
)

// errNotRegular means a path names something other than a regular file:
// a FIFO, a device, a socket or a directory.
var errNotRegular = errors.New("not a regular file")

// OpenRegular opens path for reading only if it is a regular file
// (codex-review-020 B5). A FIFO without a writer, or a device, would make a
// plain open or the first read block for ever, before any type check or
// cancellation could run. So the file is opened in a way that cannot block
// on such files, and its type is then read from the open descriptor (or
// handle) itself, not from the path, so that nothing can be swapped in
// between. Anything but a regular file is closed again and refused; links
// are followed as before.
func OpenRegular(path string) (*os.File, error) { return openRegular(path) }
