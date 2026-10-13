package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// SnapshotTree records every entry under root: its type, permission bits,
// size, modification time and content hash. Two equal snapshots mean
// nothing under root was created, removed, rewritten or touched.
func SnapshotTree(t testing.TB, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		desc := fmt.Sprintf("%v %d %d", info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(b)
			desc += " " + hex.EncodeToString(sum[:])
		}
		out[filepath.ToSlash(rel)] = desc
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// DiffSnapshots describes the differences between two SnapshotTree results,
// or returns "" if they are equal.
func DiffSnapshots(before, after map[string]string) string {
	keys := map[string]bool{}
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	var diffs []string
	for _, k := range names {
		b, inB := before[k]
		a, inA := after[k]
		switch {
		case !inB:
			diffs = append(diffs, "created "+k)
		case !inA:
			diffs = append(diffs, "removed "+k)
		case a != b:
			diffs = append(diffs, fmt.Sprintf("changed %s: %s -> %s", k, b, a))
		}
	}
	return strings.Join(diffs, "\n")
}

// AssertTreeUnchanged fails the test if root differs from before.
func AssertTreeUnchanged(t testing.TB, root string, before map[string]string) {
	t.Helper()
	if d := DiffSnapshots(before, SnapshotTree(t, root)); d != "" {
		t.Fatalf("%s was modified:\n%s", root, d)
	}
}

// SettledSnapshot is SnapshotTree taken once two snapshots in a row agree.
// On Windows, NTFS may publish the modification time of another process's
// earlier write only later; settling first keeps that from being mistaken
// for a change made by the code under test.
func SettledSnapshot(t testing.TB, root string) map[string]string {
	t.Helper()
	prev := SnapshotTree(t, root)
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		cur := SnapshotTree(t, root)
		if DiffSnapshots(prev, cur) == "" {
			return cur
		}
		prev = cur
	}
	t.Fatalf("%s kept changing", root)
	return nil
}
