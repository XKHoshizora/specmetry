package gitx

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileKey is a file's identity: its volume and file ID, read once from a
// handle (os.SameFile would look a FileInfo's ID up by path when compared,
// and so find whatever is at the path by then).
type fileKey string

// fileIDInfo is FILE_ID_INFO: the 128-bit file ID ReFS needs.
type fileIDInfo struct {
	VolumeSerialNumber uint64
	FileID             [16]byte
}

// handleKey returns the identity and attributes of the file open as h.
func handleKey(h windows.Handle) (fileKey, uint32, error) {
	var bi windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &bi); err != nil {
		return "", 0, err
	}
	var id fileIDInfo
	buf := (*byte)(unsafe.Pointer(&id)) //nolint:gosec // FILE_ID_INFO's documented layout
	if err := windows.GetFileInformationByHandleEx(h, windows.FileIdInfo, buf, uint32(unsafe.Sizeof(id))); err == nil {
		return fileKey(fmt.Sprintf("%x:%x", id.VolumeSerialNumber, id.FileID)), bi.FileAttributes, nil
	}
	return fileKey(fmt.Sprintf("%x:%x:%x", bi.VolumeSerialNumber, bi.FileIndexHigh, bi.FileIndexLow)), bi.FileAttributes, nil
}

func isDir(attrs uint32) bool { return attrs&windows.FILE_ATTRIBUTE_DIRECTORY != 0 }

func isReparse(attrs uint32) bool { return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 }

// keyOf returns the identity of the open file f, and whether it is a
// directory.
func keyOf(f *os.File) (fileKey, bool, error) {
	k, attrs, err := handleKey(windows.Handle(f.Fd()))
	return k, isDir(attrs), err
}

// openEntry opens what is at path itself, a link or junction not followed.
func openEntry(path string, access uint32) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return windows.CreateFile(p, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
}

// lstatKey returns the identity of what is at path, not following a link
// or junction there, and whether it is a directory (a link or junction, or
// any other reparse point, is not).
func lstatKey(path string) (fileKey, bool, error) {
	h, err := openEntry(path, windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	k, attrs, err := handleKey(h)
	return k, isDir(attrs) && !isReparse(attrs), err
}

// removeEntry removes the entry name of the directory at parentPath if it
// is still the file key: a directory when dir is set (removed only if
// empty), otherwise a file. What is at the name is opened, its identity
// read from that handle, and the deletion set on the same handle, so that
// only the file key can be removed, whatever is put at the name meanwhile;
// it goes once the last handle on it is closed.
func removeEntry(_ *os.File, parentPath, name string, key fileKey, dir bool) error {
	h, err := openEntry(filepath.Join(parentPath, name), windows.DELETE|windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	k, attrs, err := handleKey(h)
	if err != nil {
		return err
	}
	if k != key || isReparse(attrs) || isDir(attrs) != dir {
		return errReplaced
	}
	deleteFile := byte(1) // FILE_DISPOSITION_INFO
	return windows.SetFileInformationByHandle(h, windows.FileDispositionInfo, &deleteFile, 1)
}
