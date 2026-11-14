package chain

import "os"

// FS is the file system the writer uses. Only the writer task calls it,
// always outside every lock.
type FS interface {
	// Mkdir creates the record directory; it must not exist yet.
	Mkdir(dir string) error
	// Create creates a new file exclusively (it must not exist yet) for
	// writing.
	Create(path string) (File, error)
}

// File is an open record file or manifest. A record item is on disk
// ("落盘", plan §6.4) when the Write that carried it returned with no error;
// there is no Sync. Close is called at most once per file, and a Close error
// means the release is unknown; it is never retried.
type File interface {
	Write(p []byte) (int, error)
	Close() error
}

// OSFS is the real file system.
type OSFS struct{}

// Mkdir is os.Mkdir with mode 0o755; it fails if dir exists.
func (OSFS) Mkdir(dir string) error { return os.Mkdir(dir, 0o755) }

// Create opens path with O_WRONLY|O_CREATE|O_EXCL and mode 0o644.
func (OSFS) Create(path string) (File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
