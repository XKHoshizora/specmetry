//go:build windows

package source

import "os"

// openRegular: on Windows opening a named pipe, a console or another
// device does not wait (CreateFile fails at once, for example with
// ERROR_PIPE_BUSY, or returns a handle); only reading could. File.Stat on
// the open handle asks the handle itself (GetFileType, then
// GetFileInformationByHandle), so a device or pipe is refused before
// anything is read from it, and nothing can be swapped in after the check.
func openRegular(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		_ = f.Close()
		return nil, &os.PathError{Op: "open", Path: path, Err: errNotRegular}
	}
	return f, nil
}
