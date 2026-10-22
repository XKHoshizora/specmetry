package gitx

import (
	"context"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// commitPrefix is a commit reference of a claim (common.v1 commitShaPrefix).
var commitPrefix = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// ResolveCommits tells, for each commit ID prefix of a claim's references,
// whether it names exactly one commit the repository can read (WP2.4 spec
// §7, decision E4):
//   - true: git lists exactly one object whose ID starts with the prefix,
//     and it is a commit;
//   - false: git lists no object, the one object it lists is not a commit
//     (a tree, a blob, or a tag, even one that points to a commit), or it
//     lists several (a true ambiguity: a longer prefix or the full ID tells
//     them apart);
//   - an environment error (exit 6) otherwise, and no answer at all for any
//     prefix: an object git lists that does not read whole as the object
//     its ID names, anything git writes to stderr, an exit status or an
//     answer other than the expected ones, or a failed storage check. A
//     failed git command is never taken for false.
//
// Every object a prefix lists is read whole before any answer (review 067
// B1): not only a commit that would resolve, but also a lone non-commit
// and each candidate of an ambiguous prefix, since each decides the
// answer. git does not check an object's hash when it reads it: a damaged
// loose object whose header still inflates comes out cut short, or padded
// with whatever was in memory, and one holding another object's bytes
// comes out as that object, all with exit 0 (scratchpad experiments trunc
// and trunc2, git 2.56). So each object's header must be the type and size
// git listed it with, and its content, as it streams past and never kept,
// must be exactly that size and hash to its ID.
//
// The prefixes are 7 to 40 lower-case hexadecimal digits (claims.v1
// refuses anything else first); any other is an internal error, and never
// reaches git, which would read it as a revision or an option. The reads
// are the commit scan's (scanCall): a git shown never to fetch an object
// (PD-26), with the commit-graph and the multi-pack-index out of use, so
// that git lists the objects of the loose directories and of every pack
// index itself. Each object is read once, however many prefixes list it,
// in one cat-file --batch, which a cancelled context ends. The storage
// check (storage.go) runs once whenever there is a prefix: storage that is
// damaged, incomplete or unreadable can hide an object from git, which
// would make an ambiguous prefix look unique or a commit look absent.
// Nothing is written, and git's messages are not kept.
func ResolveCommits(ctx context.Context, s *Scratch, worktree string, prefixes []string) (map[string]bool, error) {
	var unique []string
	for _, p := range prefixes {
		if !commitPrefix.MatchString(p) {
			return nil, apperr.New(apperr.Internal, "ResolveCommits needs commit ID prefixes of 7 to 40 lower-case hexadecimal digits")
		}
		if !slices.Contains(unique, p) {
			unique = append(unique, p)
		}
	}
	got := map[string]bool{}
	if len(unique) == 0 {
		return got, nil
	}
	unreliable := apperr.New(apperr.Environment,
		"a commit reference cannot be resolved reliably: git cannot read an object it lists as the object its ID names, or the object storage is damaged, incomplete or unreadable; nothing was written; run git fsck in this directory")

	// 1. The objects each prefix names, as git lists them.
	candidates := map[string][]string{}
	var listed []string
	for _, p := range unique {
		r, err := scanCall(ctx, s, call{label: "rev-parse", dir: worktree, args: []string{"rev-parse", "--disambiguate=" + p}})
		if err != nil {
			return nil, err
		}
		if r.code != 0 || r.stderr {
			return nil, unreliable
		}
		var ids []string
		var lines []string
		if len(r.out) > 0 {
			lines = strings.Split(strings.TrimSuffix(string(r.out), "\n"), "\n")
		}
		for _, line := range lines {
			if !commitID.MatchString(line) || !strings.HasPrefix(line, p) {
				return nil, unreliable
			}
			if !slices.Contains(ids, line) {
				ids = append(ids, line)
			}
			if !slices.Contains(listed, line) {
				listed = append(listed, line)
			}
		}
		candidates[p] = ids
	}

	// 2. Every listed object read whole and checked against its ID.
	types, ok, err := readVerified(func(c call) (result, error) { return scanCall(ctx, s, c) }, worktree, listed, nil)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, unreliable
	}

	// 3. The storage git listed from, checked by Specmetry itself.
	objects, err := call{label: "rev-parse", dir: worktree,
		args: []string{"rev-parse", "--path-format=absolute", "--git-path", "objects"}}.must(ctx)
	if err != nil {
		return nil, err
	}
	dir := filepath.FromSlash(strings.TrimRight(string(objects), "\r\n"))
	if strings.ContainsAny(dir, "\r\n") || !filepath.IsAbs(dir) || checkStorage(dir) != nil {
		return nil, unreliable
	}

	// 4. The answers, only now that every object they rest on is checked.
	for _, p := range unique {
		ids := candidates[p]
		got[p] = len(ids) == 1 && types[ids[0]] == "commit"
	}
	return got, nil
}
