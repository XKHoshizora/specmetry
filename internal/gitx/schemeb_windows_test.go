package gitx

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsAtomicPrivateDirectory(t *testing.T) {
	s, _ := scratchIn(t)
	f, err := createDirectory(s, "collision")
	if err != nil {
		t.Fatal(err)
	}
	key, dir, err := keyOf(f)
	if err != nil || !dir {
		t.Fatalf("identity: %v %v", dir, err)
	}
	if second, err := createDirectory(s, "collision"); err == nil {
		_ = second.Close()
		t.Fatal("FILE_CREATE accepted an existing directory")
	}
	path := filepath.Join(s.path, "collision")
	if err := os.Rename(path, path+".moved"); err == nil {
		t.Fatal("creation guard allowed rename")
	}
	current, _, err := lstatKey(path)
	if err != nil || current != key {
		t.Fatalf("creation identity changed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	// Simulate replacement AFTER obtaining the deleting handle. The handle
	// must delete its original object; it cannot delete the new namesake.
	h, err := openEntry(path, windows.DELETE|windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".moved"); err != nil {
		_ = windows.CloseHandle(h)
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		_ = windows.CloseHandle(h)
		t.Fatal(err)
	}
	if err := disposeEntry(h, key, true); err != nil {
		_ = windows.CloseHandle(h)
		t.Fatal(err)
	}
	if err := windows.CloseHandle(h); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".moved"); !os.IsNotExist(err) {
		t.Fatalf("opened original retained: %v", err)
	}
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		t.Fatalf("replacement was removed: %v", err)
	}
}
