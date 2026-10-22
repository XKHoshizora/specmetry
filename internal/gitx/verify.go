package gitx

import (
	"crypto/sha1" //nolint:gosec // git's own object IDs, not a security use
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Verified object reads. git does not check an object's hash when it reads
// it: a damaged loose object whose header still inflates comes out cut
// short, or padded with whatever was in memory, and one whose file holds
// another object's bytes comes out as that object, all with exit 0
// (scratchpad experiments trunc and trunc2, git 2.56; reviews 067 B1 and
// 068 B3). So a read Specmetry decides on goes through one cat-file --batch
// whose output objectVerifier checks against the IDs asked for, and only
// what that same read gave, once it has all been checked, is used.

// errObject is an object cat-file --batch did not give as the object its
// ID names. It is never shown: the caller reports that it cannot rely on
// the objects.
var errObject = errors.New("an object does not read as the object its ID names")

// maxBatchHeader bounds a header line of cat-file --batch: an ID, a type
// and a size.
const maxBatchHeader = 128

// readVerified reads the objects ids (full IDs, each once) whole through
// one cat-file --batch, run by run (the caller's controlled git: the scan's
// or the object reads'), and checks each against its ID (objectVerifier).
// sink, when not nil, is given each object's ID and type at its header and
// returns where its content also goes, or nil; what it received may be
// used only when readVerified reports the read whole and right (ok). The
// types read are returned then. err is run's own error (PD-26, a cancelled
// context); ok is false for anything else that went wrong.
func readVerified(run func(call) (result, error), worktree string, ids []string, sink func(id, typ string) io.Writer) (types map[string]string, ok bool, err error) {
	types = map[string]string{}
	if len(ids) == 0 {
		return types, true, nil
	}
	v := &objectVerifier{ids: ids, types: types, sink: sink}
	r, err := run(call{label: "cat-file", dir: worktree, args: []string{"cat-file", "--batch"},
		stdin: []byte(strings.Join(ids, "\n") + "\n"), stdout: v})
	if err != nil {
		return nil, false, err
	}
	if r.code != 0 || r.stderr || v.close() != nil {
		return nil, false, nil
	}
	return types, true, nil
}

// objectVerifier checks the output of cat-file --batch as it streams past,
// for the objects ids in order: each the header "<id> <type> <size>" of
// that ID with a known type, then exactly size bytes whose object ID,
// recomputed over "<type> <size>\x00" and them, is the ID, then a line
// break. It records each object's type, passes its content to sink's
// writer if there is one, and keeps nothing itself. A header line longer
// than maxBatchHeader is refused however the output is cut into writes
// (review 068 N1). After a mismatch it takes and drops the rest, so that
// git ends normally; close reports it.
type objectVerifier struct {
	ids   []string
	types map[string]string
	sink  func(id, typ string) io.Writer
	next  int       // the index of the object being read
	head  []byte    // the header line so far
	hash  hash.Hash // nil while a header is read
	out   io.Writer // where the current object's content also goes
	left  int64     // content bytes still to come
	err   error
}

func (v *objectVerifier) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 && v.err == nil {
		switch {
		case v.hash == nil:
			i := slices.Index(p, '\n')
			line := p
			if i >= 0 {
				line = p[:i]
			}
			if len(v.head)+len(line) > maxBatchHeader {
				v.err = errObject
				break
			}
			v.head = append(v.head, line...)
			if i < 0 {
				p = nil
				break
			}
			p = p[i+1:]
			v.begin()
		case v.left > 0:
			k := int64(len(p))
			if k > v.left {
				k = v.left
			}
			_, _ = v.hash.Write(p[:k])
			if v.out != nil {
				_, _ = v.out.Write(p[:k])
			}
			v.left -= k
			p = p[k:]
		default:
			if p[0] != '\n' {
				v.err = errObject
				break
			}
			p = p[1:]
			if hex.EncodeToString(v.hash.Sum(nil)) != v.ids[v.next] {
				v.err = errObject
				break
			}
			v.next++
			v.hash = nil
		}
	}
	return n, nil
}

// begin reads the header of the next object.
func (v *objectVerifier) begin() {
	f := strings.Fields(string(v.head))
	v.head = v.head[:0]
	if v.next >= len(v.ids) || len(f) != 3 || f[0] != v.ids[v.next] ||
		!slices.Contains([]string{"commit", "tree", "blob", "tag"}, f[1]) || !isDecimal(f[2]) {
		v.err = errObject
		return
	}
	size, err := strconv.ParseInt(f[2], 10, 64)
	if err != nil {
		v.err = errObject
		return
	}
	v.types[f[0]] = f[1]
	v.hash = sha1.New() //nolint:gosec // git's own object ID, not a security use
	_, _ = v.hash.Write([]byte(f[1] + " " + f[2] + "\x00"))
	v.out = nil
	if v.sink != nil {
		v.out = v.sink(f[0], f[1])
	}
	v.left = size
}

// close reports whether every object was given whole and as its ID names,
// with nothing else.
func (v *objectVerifier) close() error {
	if v.err == nil && (v.next != len(v.ids) || v.hash != nil || len(v.head) != 0) {
		v.err = errObject
	}
	return v.err
}
