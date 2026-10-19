package gitx

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1" //nolint:gosec // Git SHA-1 object identity, not an authentication primitive.
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

const msgScanEmpty = "cannot verify that this git honours empty graft and shallow overrides; no commits were scanned; no pathname fallback is allowed"

// scanGit's cache is reset with every executable identity/size/mtime change.
// A failed/cancelled/incomplete probe never records a successful capability.
func scanGit(ctx context.Context, s *Scratch) (string, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	git, err := objectGitLocked(ctx)
	if err != nil {
		return "", err
	}
	if !state.scanEmpty {
		before := state.bin
		if err := probeScanEmpty(ctx, s, git); err != nil {
			return "", err
		}
		after, err := findGit()
		if err != nil || after != before {
			return "", refused(msgScanEmpty)
		}
		state.scanEmpty = true
	}
	return git, nil
}

// probeScanEmpty creates only synthetic Git objects in a manifested v2 lease.
// It invokes no writer Git commands, has no remotes/alternates/filters, and all
// subprocesses inherit the controlled deny-all transport environment. A real
// root+child+merge graph must be cut by the positive control and fully read by
// BOTH empty overrides; missing and corrupt parents must fail, even after output.
func probeScanEmpty(ctx context.Context, s *Scratch, git string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	d, err := newPrivateDir(s)
	if err != nil {
		return err
	}
	defer func() {
		if e := d.remove(); e != nil {
			err = errors.Join(err, e)
		}
	}()
	bad := func() error { return refused(msgScanEmpty) }
	for _, path := range []string{"objects/info", "objects/pack", "refs/heads", "info"} {
		if d.mkdirAll(path) != nil {
			return bad()
		}
	}
	for name, value := range map[string]string{"config": "[core]\nrepositoryformatversion=0\nbare=true\n", "HEAD": "ref: refs/heads/probe\n"} {
		if d.write(name, []byte(value)) != nil {
			return bad()
		}
	}
	object := func(kind, text string) (string, error) {
		raw := []byte(fmt.Sprintf("%s %d%c%s", kind, len(text), 0, text))
		hash := sha1.Sum(raw) //nolint:gosec // Git object identity
		id := hex.EncodeToString(hash[:])
		if err := d.mkdirAll("objects/" + id[:2]); err != nil {
			return "", err
		}
		var b bytes.Buffer
		w := zlib.NewWriter(&b)
		if _, err := w.Write(raw); err != nil {
			return "", err
		}
		if err := w.Close(); err != nil {
			return "", err
		}
		return id, d.write("objects/"+id[:2]+"/"+id[2:], b.Bytes())
	}
	tree, err := object("tree", "")
	if err != nil {
		return bad()
	}
	commit := func(parents []string, n int) (string, error) {
		text := "tree " + tree + "\n"
		for _, parent := range parents {
			text += "parent " + parent + "\n"
		}
		text += fmt.Sprintf("author Probe <probe@invalid> %d +0000\ncommitter Probe <probe@invalid> %d +0000\n\nprobe %d\n", n, n, n)
		return object("commit", text)
	}
	a, err := commit(nil, 1)
	if err != nil {
		return bad()
	}
	b, err := commit([]string{a}, 2)
	if err != nil {
		return bad()
	}
	tip, err := commit([]string{b, a}, 3)
	if err != nil {
		return bad()
	}
	for name, value := range map[string]string{"refs/heads/probe": tip + "\n", "info/grafts": tip + "\n", "shallow": tip + "\n"} {
		if d.write(name, []byte(value)) != nil {
			return bad()
		}
	}
	read := func(empty bool, excludes ...string) (result, error) {
		if err := d.verify(); err != nil {
			return result{}, err
		}
		args := append(slices.Clone(scanArgs), "--git-dir="+d.path, "log", "--no-show-signature", "--no-decorate", "--format=%H", tip)
		args = append(args, excludes...)
		c := call{label: "log", dir: d.path, args: args}
		if empty {
			c.env = []string{"GIT_GRAFT_FILE=", "GIT_SHALLOW_FILE="}
		}
		return c.runWithGit(ctx, git)
	}
	control, err := read(false)
	if err != nil || control.code != 0 || strings.TrimSpace(string(control.out)) != tip {
		return bad()
	}
	full, err := read(true)
	if err != nil || full.code != 0 || strings.TrimSpace(string(full.out)) != tip+"\n"+b+"\n"+a {
		return bad()
	}
	incremental, err := read(true, "^"+b)
	if err != nil || incremental.code != 0 || strings.TrimSpace(string(incremental.out)) != tip {
		return bad()
	}
	// Corrupt an existing required parent, without replacing its identity.
	parentPath := "objects/" + a[:2] + "/" + a[2:]
	f, err := d.root.OpenFile(parentPath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return bad()
	}
	_, err = f.Write([]byte("not a Git object"))
	err = errors.Join(err, f.Close())
	if err != nil {
		return bad()
	}
	corrupt, err := read(true)
	if err != nil || corrupt.code == 0 {
		return bad()
	}
	if err := d.removeOwned(parentPath); err != nil {
		return bad()
	}
	missing, err := read(true)
	if err != nil || missing.code == 0 {
		return bad()
	}
	return nil
}
