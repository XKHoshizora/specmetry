// Package modsnap implements the module content snapshot I-4 (group H),
// the controlled read I-5 (groups C and P) with its ReadDir view, and the
// plain recorded read of files outside the go.sum chain (the repository
// checkout and the standard library). Method record §2.3 to §2.6; plan
// §2.5.4.
//
// Every real open, read and close is done by this package, once per call,
// and every real Close result is kept: dirhash.Hash1 and go/build's
// MatchFile, which do not check Close, only ever see bytes already read.
package modsnap

import (
	"io"
	"io/fs"
	"os"
)

// FS is the read side of the file system. The real one is OS; tests
// inject faults with their own.
type FS interface {
	Lstat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Open(path string) (File, error)
}

// File is an opened file.
type File interface {
	Stat() (fs.FileInfo, error)
	io.Reader
	Close() error
}

// OS is the real file system.
type OS struct{}

// Lstat is os.Lstat.
func (OS) Lstat(path string) (fs.FileInfo, error) { return os.Lstat(path) }

// ReadDir is os.ReadDir.
func (OS) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }

// Open is os.Open.
func (OS) Open(path string) (File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// readOnce opens path once, checks that the handle is a regular file,
// reads to EOF and closes once. It returns the bytes, whether the handle
// was regular, the read error and the real Close result separately; the
// close is attempted whenever the open succeeded.
func readOnce(fsys FS, path string) (b []byte, regular bool, openErr, statErr, readErr, closeErr error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, false, err, nil, nil, nil
	}
	fi, serr := f.Stat()
	switch {
	case serr != nil:
		statErr = serr
	case fi.Mode().IsRegular():
		regular = true
		b, readErr = io.ReadAll(f)
	}
	closeErr = f.Close()
	return b, regular, nil, statErr, readErr, closeErr
}
