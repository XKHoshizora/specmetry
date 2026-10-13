//go:build darwin

package datadir

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// extendedACLProblems reports entries with a macOS extended ACL that allows
// access. Such entries are evaluated before the mode bits, so a file with
// mode 0600 can still be readable by others. Go has no binding for
// getattrlist or acl_get_file without cgo, so this asks /bin/ls, whose -e
// option prints each ACL entry ("0: group:everyone allow read") under the
// file it belongs to. Deny entries only restrict access and are ignored.
func extendedACLProblems(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out, err := listACLs(paths...)
	if err != nil {
		return nil, err
	}
	if !hasAllowEntry(out) {
		return nil, nil
	}
	// At least one entry has an allow ACE; find which, one at a time, so
	// that file names never need to be parsed out of ls output.
	var problems []string
	for _, p := range paths {
		out, err := listACLs(p)
		if err != nil {
			return nil, err
		}
		if hasAllowEntry(out) {
			problems = append(problems, fmt.Sprintf("%s: has an access control list that allows access (remove it with: chmod -N %q)", p, p))
		}
	}
	return problems, nil
}

func listACLs(paths ...string) ([]byte, error) {
	args := append([]string{"-lde", "--"}, paths...)
	out, err := exec.Command("/bin/ls", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot list access control lists with /bin/ls: %w", err)
	}
	return out, nil
}

// hasAllowEntry looks for ACL lines, which ls -e prints indented and
// numbered: " 0: user:alice allow read".
func hasAllowEntry(out []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimLeft(line, " ")
		if len(trimmed) == len(line) {
			continue // a file line, not an ACL entry
		}
		if i := strings.Index(trimmed, ": "); i > 0 && isDigits(trimmed[:i]) && strings.Contains(trimmed[i:], " allow ") {
			return true
		}
	}
	return false
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
