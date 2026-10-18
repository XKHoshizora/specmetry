package gitx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// State is a worktree state as Clean reads it.
type State int

const (
	StateClean State = iota
	StateDirty
	// StateUndecidable: telling clean from dirty would need a program the
	// repository configures (WP2.3 spec §4.3 step 5), or the index is a
	// sparse index (step 2). It is recorded as unknown, never as clean
	// (decision P1).
	StateUndecidable
)

// Clean reads the worktree state of the worktree root (data-model §1, D1):
// clean when `git status --porcelain --untracked-files=all
// --ignore-submodules=none -- . ':(exclude).specmetry'` would print
// nothing. It never runs that command on the repository: status would run
// the clean and process filters the repository configures, and in each
// submodule another status (review 050 B3). Instead (WP2.3 spec §4.3), for
// the superproject and then each initialized submodule:
//
//  1. a private git directory of Specmetry's own: the worktree's HEAD,
//     alternates to the repository's objects, copies of the index (and the
//     shared index it links to) and of info/exclude, info/attributes and
//     info/sparse-checkout, and a configuration with only whitelisted keys,
//     so that nothing the repository configures can run;
//  2. the sparse gate: Specmetry parses the index copy itself; a sparse
//     index is undecidable, with no git command run on it (P2), since git
//     reading it may expand it and rewrite the repository's object files;
//  3. status, ls-files, diff-index and diff-files on the private directory,
//     with --ignore-submodules=all so that git never enters a submodule;
//  4. each submodule: missing or not a directory is a change, as is a HEAD
//     other than the gitlink; an initialized one is read in turn, behind
//     its own gate;
//  5. a filtered path the definition's status would compare (stat changed,
//     or racy) under a driver of the local configuration is undecidable.
//
// --ignore-submodules=all hides more than a submodule's own state: a staged
// gitlink, a gitlink removed from the index, a submodule directory removed
// or replaced. The definition shows those, so Clean compares the index's
// gitlinks with HEAD's (diff-index --cached) and looks at each gitlink's
// path itself (an implementation finding, reported with WP2.3: spec §4.3
// step 4 compares a submodule's HEAD with the gitlink only).
//
// Any failure is an error (exit 6), never clean. The private directories
// are removed, and one that cannot be removed is an error too.
func Clean(ctx context.Context, worktree string) (State, error) {
	removeStalePrivateDirs()
	return cleanRepo(ctx, worktree, true)
}

// cleanRepo reads one repository: the superproject (top), or a submodule.
func cleanRepo(ctx context.Context, worktree string, top bool) (State, error) {
	p, err := prepare(ctx, worktree)
	if err != nil {
		return 0, err
	}
	state, links, err := p.read(ctx, top)
	if cerr := p.close(); err == nil {
		err = cerr
	}
	if err != nil || state == StateUndecidable {
		return state, err
	}
	for _, l := range links {
		sub, err := submoduleState(ctx, p.worktree, l)
		if err != nil || sub == StateUndecidable {
			return sub, err
		}
		if sub == StateDirty {
			state = StateDirty
		}
	}
	return state, nil
}

// whitelist is the configuration a private git directory takes from the
// repository's local (and worktree) configuration: the settings that
// decide what the definition's status compares and how, none of them able
// to run a program.
var whitelist = map[string]bool{
	"core.autocrlf": true, "core.eol": true, "core.safecrlf": true, "core.ignorecase": true, "core.precomposeunicode": true,
	"core.protecthfs": true, "core.protectntfs": true, "core.symlinks": true, "core.filemode": true, "core.checkstat": true,
	"core.trustctime": true, "core.excludesfile": true, "core.attributesfile": true, "core.sparsecheckout": true,
	"core.sparsecheckoutcone": true, "index.sparse": true, "core.longpaths": true, "core.fscache": true,
	"core.checkroundtripencoding": true,
}

// privateRepo is a repository's private git directory, ready to read.
type privateRepo struct {
	dir      *privateDir
	worktree string // the worktree root, canonical
	gitDir   string // dir.path, the private git directory
	head     string // the worktree's HEAD commit, "" when unborn
	drivers  map[string]bool
	// indexTime is the original index file's modification time (zero
	// when there is none), for the racy rule.
	indexTime time.Time
}

// repoPaths is what git says of a worktree's repository.
type repoPaths struct {
	top, gitDir, objects, index string
	info                        map[string]string // info/exclude, info/attributes, info/sparse-checkout
	format                      string
}

func readPaths(ctx context.Context, worktree string) (repoPaths, error) {
	infos := []string{"info/exclude", "info/attributes", "info/sparse-checkout"}
	args := []string{"rev-parse", "--path-format=absolute", "--show-toplevel", "--absolute-git-dir",
		"--git-path", "objects", "--git-path", "index"}
	for _, f := range infos {
		args = append(args, "--git-path", f)
	}
	args = append(args, "--show-object-format")
	out, err := call{label: "rev-parse", dir: worktree, args: args}.must(ctx)
	if err != nil {
		return repoPaths{}, err
	}
	lines := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n"), "\n")
	if len(lines) != 5+len(infos) {
		return repoPaths{}, apperr.New(apperr.Environment, "git rev-parse printed paths Specmetry cannot read")
	}
	path := func(s string) string { return filepath.FromSlash(s) }
	p := repoPaths{top: path(lines[0]), gitDir: path(lines[1]), objects: path(lines[2]), index: path(lines[3]),
		info: map[string]string{}, format: lines[len(lines)-1]}
	for i, f := range infos {
		p.info[f] = path(lines[4+i])
	}
	for _, q := range append([]string{p.top, p.gitDir, p.objects, p.index}, p.info["info/exclude"]) {
		if !filepath.IsAbs(q) {
			return repoPaths{}, apperr.New(apperr.Environment, "git rev-parse printed paths Specmetry cannot read")
		}
	}
	return p, nil
}

// prepare makes the private git directory of the repository whose worktree
// root is worktree (step 1). It runs only git commands that read no index:
// rev-parse for paths and HEAD, symbolic-ref, config.
func prepare(ctx context.Context, worktree string) (*privateRepo, error) {
	paths, err := readPaths(ctx, worktree)
	if err != nil {
		return nil, err
	}
	if paths.format != "sha1" {
		return nil, apperr.New(apperr.Environment, "Specmetry reads only SHA-1 repositories (TD-26)")
	}
	if Canonical(paths.top) != Canonical(worktree) {
		return nil, apperr.New(apperr.Internal, "Clean needs the root of a worktree")
	}
	head, err := worktreeHead(ctx, worktree)
	if err != nil {
		return nil, err
	}
	config, drivers, err := privateConfig(ctx, worktree)
	if err != nil {
		return nil, err
	}
	d, err := newPrivateDir()
	if err != nil {
		return nil, err
	}
	p := &privateRepo{dir: d, worktree: Canonical(worktree), gitDir: d.path, drivers: drivers}
	if head.ref == "" {
		p.head = head.commit
	}
	if err := p.fill(paths, head, config); err != nil {
		_ = d.remove()
		return nil, err
	}
	return p, nil
}

// headState is a worktree's HEAD: a commit, or an unborn branch.
type headState struct{ commit, ref string }

func worktreeHead(ctx context.Context, worktree string) (headState, error) {
	r, err := call{label: "rev-parse", dir: worktree, objects: true, args: []string{"rev-parse", "-q", "--verify", "HEAD"}}.run(ctx)
	if err != nil {
		return headState{}, err
	}
	switch r.code {
	case 0:
		c := strings.TrimSpace(string(r.out))
		if !commitID.MatchString(c) {
			return headState{}, apperr.New(apperr.Environment, "git rev-parse printed something other than a commit ID")
		}
		return headState{commit: c}, nil
	case 1:
	default:
		return headState{}, failed("rev-parse", r.code)
	}
	out, err := call{label: "symbolic-ref", dir: worktree, args: []string{"symbolic-ref", "-q", "HEAD"}}.must(ctx)
	if err != nil {
		return headState{}, err
	}
	ref := strings.TrimSpace(string(out))
	if !strings.HasPrefix(ref, "refs/heads/") || strings.ContainsAny(ref, "\r\n") {
		return headState{}, apperr.New(apperr.Environment, "cannot read the HEAD of the worktree")
	}
	return headState{ref: ref}, nil
}

// privateConfig reads the local configuration (and, with
// extensions.worktreeConfig, the worktree's, which wins) and returns the
// private configuration and the filter drivers it defines: the names with
// a clean or a process filter, or required (step 5's D).
func privateConfig(ctx context.Context, worktree string) ([]byte, map[string]bool, error) {
	entries, err := configList(ctx, worktree, "--local")
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if e.key == "extensions.worktreeconfig" && truthy(e) {
			wt, err := configList(ctx, worktree, "--worktree")
			if err != nil {
				return nil, nil, err
			}
			entries = append(entries, wt...)
			break
		}
	}
	drivers := map[string]bool{}
	var core, index bytes.Buffer
	core.WriteString("[core]\n\trepositoryformatversion = 0\n\tbare = false\n\tuntrackedCache = false\n")
	for _, e := range entries {
		if name, ok := strings.CutPrefix(e.key, "filter."); ok {
			for _, v := range []string{".clean", ".process"} {
				if n, ok := strings.CutSuffix(name, v); ok && n != "" {
					drivers[n] = true
				}
			}
			if n, ok := strings.CutSuffix(name, ".required"); ok && n != "" && truthy(e) {
				drivers[n] = true
			}
		}
		if !whitelist[e.key] {
			continue
		}
		section, key, _ := strings.Cut(e.key, ".")
		b := &core
		if section == "index" {
			b = &index
		}
		b.WriteString("\t" + key)
		if e.hasValue {
			b.WriteString(" = " + quoteConfig(e.value))
		}
		b.WriteString("\n")
	}
	if index.Len() > 0 {
		core.WriteString("[index]\n")
		core.Write(index.Bytes())
	}
	return core.Bytes(), drivers, nil
}

// configEntry is one entry of `git config --list -z`: a key (git gives its
// section and name in lower case) with or without a value.
type configEntry struct {
	key, value string
	hasValue   bool
}

func configList(ctx context.Context, worktree, scope string) ([]configEntry, error) {
	r, err := call{label: "config", dir: worktree, args: []string{"config", "--list", scope, "--includes", "-z"}}.run(ctx)
	if err != nil {
		return nil, err
	}
	if r.code != 0 {
		return nil, failed("config", r.code)
	}
	var out []configEntry
	for _, raw := range strings.Split(string(r.out), "\x00") {
		if raw == "" {
			continue
		}
		k, v, ok := strings.Cut(raw, "\n")
		out = append(out, configEntry{key: strings.ToLower(k), value: v, hasValue: ok})
	}
	return out, nil
}

// truthy reads a configuration boolean leniently: anything that is not
// clearly false is true (conservative for filter.<name>.required).
func truthy(e configEntry) bool {
	if !e.hasValue {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(e.value)) {
	case "false", "no", "off", "0", "":
		return false
	}
	return true
}

// quoteConfig quotes a configuration value for a config file.
func quoteConfig(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\b", `\b`)
	return `"` + r.Replace(v) + `"`
}

// fill writes the private git directory's files.
func (p *privateRepo) fill(paths repoPaths, head headState, config []byte) error {
	for _, d := range []string{"refs/heads", "refs/tags", "objects/info", "objects/pack", "info"} {
		if err := os.MkdirAll(p.dir.join(filepath.FromSlash(d)), 0o700); err != nil {
			return privateProblem(err)
		}
	}
	headFile := head.commit + "\n"
	if head.ref != "" {
		headFile = "ref: " + head.ref + "\n"
	}
	objects := filepath.ToSlash(paths.objects)
	if strings.ContainsAny(objects, "\r\n") || strings.HasPrefix(objects, `"`) {
		return apperr.New(apperr.Environment, "the repository's object directory has a path Specmetry cannot name in an alternates file")
	}
	for name, data := range map[string][]byte{
		"HEAD":                    []byte(headFile),
		"config":                  config,
		"objects/info/alternates": []byte(objects + "\n"),
	} {
		if err := writeNew(p.dir.join(filepath.FromSlash(name)), data); err != nil {
			return privateProblem(err)
		}
	}
	for name, from := range paths.info {
		if _, err := copyIfExists(from, p.dir.join(filepath.FromSlash(name))); err != nil {
			return err
		}
	}
	found, err := copyIfExists(paths.index, p.dir.join("index"))
	if err != nil || !found.exists {
		return err // no index: an empty one (review 057 N1)
	}
	p.indexTime = found.modTime
	data, err := os.ReadFile(p.dir.join("index"))
	if err != nil {
		return privateProblem(err)
	}
	info, err := parseIndex(data)
	if err != nil || info.shared == nil {
		return err
	}
	name := sharedIndexName(*info.shared)
	shared, err := copyIfExists(filepath.Join(filepath.Dir(paths.index), name), p.dir.join(name))
	if err != nil {
		return err
	}
	if !shared.exists {
		return indexProblem("the shared index of a split index is missing")
	}
	return nil
}

// writeNew writes a file that must not exist yet, owner-only.
func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// copied is what copyIfExists found.
type copied struct {
	exists  bool
	modTime time.Time
}

// copyIfExists copies the regular file from to to, as it was when opened
// (its modification time from the same handle). A file that does not exist
// is not copied; any other failure is an error.
func copyIfExists(from, to string) (copied, error) {
	src, err := os.Open(from)
	if errors.Is(err, fs.ErrNotExist) {
		return copied{}, nil
	}
	if err != nil {
		return copied{}, privateProblem(err)
	}
	defer func() { _ = src.Close() }()
	fi, err := src.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return copied{}, apperr.New(apperr.Environment, "a file of the repository's git directory that Specmetry copies is not a regular file")
	}
	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return copied{}, privateProblem(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return copied{}, privateProblem(err)
	}
	if err := dst.Close(); err != nil {
		return copied{}, privateProblem(err)
	}
	return copied{exists: true, modTime: fi.ModTime()}, nil
}

func privateProblem(err error) error {
	return apperr.New(apperr.Environment, "cannot copy the repository's index or configuration to read its state ("+osProblem(err)+")")
}

// args returns git's arguments for a command on the private directory.
func (p *privateRepo) args(sub ...string) []string {
	return append([]string{"--git-dir=" + p.gitDir, "--work-tree=" + p.worktree}, sub...)
}

func (p *privateRepo) close() error { return p.dir.remove() }

// gitlink is an index entry for a submodule.
type gitlink struct{ path, oid string }

// read does steps 2, 3 and 5 for the repository, and returns its own state
// and its gitlinks.
func (p *privateRepo) read(ctx context.Context, top bool) (State, []gitlink, error) {
	sparse, err := IndexIsSparse(p.dir.join("index"))
	if err != nil {
		return 0, nil, err
	}
	if sparse {
		return StateUndecidable, nil, nil
	}
	pathspec := []string{"--", "."}
	if top {
		pathspec = append(pathspec, ":(exclude).specmetry")
	}
	state := StateClean
	out, err := call{label: "status", dir: p.worktree, objects: true,
		args: p.args(append([]string{"status", "--porcelain", "-z", "--untracked-files=all", "--ignore-submodules=all"}, pathspec...)...)}.must(ctx)
	if err != nil {
		return 0, nil, err
	}
	if len(out) > 0 {
		state = StateDirty
	}
	out, err = call{label: "ls-files", dir: p.worktree, list: true, args: p.args("ls-files", "-z", "--sparse", "--stage")}.must(ctx)
	if err != nil {
		return 0, nil, err
	}
	entries, err := parseStage(out)
	if err != nil {
		return 0, nil, err
	}
	var links []gitlink
	var paths []string
	for _, e := range entries {
		paths = append(paths, e.path)
		if e.mode != "160000" {
			continue
		}
		if e.stage != "0" {
			state = StateDirty // an unmerged gitlink
			continue
		}
		links = append(links, gitlink{e.path, e.oid})
	}
	// The gitlinks of the index against HEAD's, which status hides.
	switch {
	case p.head == "" && len(links) > 0:
		state = StateDirty
	case p.head != "":
		out, err = call{label: "diff-index", dir: p.worktree, objects: true,
			args: p.args(append([]string{"diff-index", "--cached", "--raw", "-z", "--ignore-submodules=none", p.head}, pathspec...)...)}.must(ctx)
		if err != nil {
			return 0, nil, err
		}
		if len(out) > 0 {
			state = StateDirty
		}
	}
	undecidable, err := p.filtered(ctx, paths)
	if err != nil {
		return 0, nil, err
	}
	if undecidable {
		return StateUndecidable, nil, nil
	}
	return state, links, nil
}

// stageEntry is one line of ls-files --stage.
type stageEntry struct{ mode, oid, stage, path string }

func parseStage(out []byte) ([]stageEntry, error) {
	bad := apperr.New(apperr.Environment, "git ls-files printed an index entry Specmetry cannot read")
	var entries []stageEntry
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		meta, p, ok := strings.Cut(rec, "\t")
		f := strings.Split(meta, " ")
		if !ok || p == "" || len(f) != 3 || !treeMode.MatchString(f[0]) || !commitID.MatchString(f[1]) || len(f[2]) != 1 || f[2] < "0" || f[2] > "3" {
			return nil, bad
		}
		entries = append(entries, stageEntry{f[0], f[1], f[2], p})
	}
	return entries, nil
}

// filtered applies step 5: a path whose filter attribute names a driver of
// the local configuration, and that the definition's status would compare
// (listed by diff-files, or racy: not older than the original index),
// makes the state undecidable.
func (p *privateRepo) filtered(ctx context.Context, paths []string) (bool, error) {
	if len(p.drivers) == 0 || len(paths) == 0 {
		return false, nil
	}
	out, err := call{label: "check-attr", dir: p.worktree, objects: true,
		args: p.args("check-attr", "-z", "--stdin", "filter"), stdin: []byte(strings.Join(paths, "\x00") + "\x00")}.must(ctx)
	if err != nil {
		return false, err
	}
	f := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	if len(f)%3 != 0 {
		return false, apperr.New(apperr.Environment, "git check-attr printed attributes Specmetry cannot read")
	}
	var filtered []string
	for i := 0; i+2 < len(f); i += 3 {
		if f[i+1] != "filter" {
			return false, apperr.New(apperr.Environment, "git check-attr printed attributes Specmetry cannot read")
		}
		if p.drivers[f[i+2]] {
			filtered = append(filtered, f[i])
		}
	}
	if len(filtered) == 0 {
		return false, nil
	}
	out, err = call{label: "diff-files", dir: p.worktree, objects: true,
		args: p.args("diff-files", "--name-only", "-z", "--ignore-submodules=all")}.must(ctx)
	if err != nil {
		return false, err
	}
	changed := strings.Split(string(out), "\x00")
	for _, path := range filtered {
		if slices.Contains(changed, path) {
			return true, nil
		}
		fi, err := os.Lstat(filepath.Join(p.worktree, filepath.FromSlash(path)))
		if err != nil {
			return true, nil // gone or unreadable: not something to call clean
		}
		if !fi.ModTime().Before(p.indexTime) {
			return true, nil
		}
	}
	return false, nil
}

// submoduleState applies step 4 to one gitlink of the repository at
// worktree.
func submoduleState(ctx context.Context, worktree string, l gitlink) (State, error) {
	dir := filepath.Join(worktree, filepath.FromSlash(l.path))
	fi, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return StateDirty, nil // the definition: " D <path>"
	case err != nil:
		return 0, apperr.New(apperr.Environment, "cannot look at a submodule of the worktree ("+osProblem(err)+")")
	case !fi.IsDir():
		return StateDirty, nil // " T <path>"
	}
	if _, err := os.Lstat(filepath.Join(dir, ".git")); errors.Is(err, fs.ErrNotExist) {
		return StateClean, nil // not initialized: status does not read it either
	} else if err != nil {
		return 0, apperr.New(apperr.Environment, "cannot look at a submodule of the worktree ("+osProblem(err)+")")
	}
	// Its .git is often a gitfile: resolve the git directory it names
	// (N1-g), then read its HEAD through that directory.
	out, err := call{label: "rev-parse", dir: dir, args: []string{"rev-parse", "--path-format=absolute", "--absolute-git-dir"}}.must(ctx)
	if err != nil {
		return 0, err
	}
	gitDir := filepath.FromSlash(strings.TrimRight(string(out), "\r\n"))
	if strings.ContainsAny(gitDir, "\r\n") || !filepath.IsAbs(gitDir) {
		return 0, apperr.New(apperr.Environment, "git rev-parse printed paths Specmetry cannot read")
	}
	r, err := call{label: "rev-parse", dir: dir, objects: true, args: []string{"--git-dir=" + gitDir, "rev-parse", "-q", "--verify", "HEAD"}}.run(ctx)
	if err != nil {
		return 0, err
	}
	state := StateClean
	switch r.code {
	case 0:
		if strings.TrimSpace(string(r.out)) != l.oid {
			state = StateDirty
		}
	case 1:
		state = StateDirty // no commit yet: not the gitlink
	default:
		return 0, failed("rev-parse", r.code)
	}
	sub, err := cleanRepo(ctx, dir, false)
	if err != nil || sub != StateClean {
		return sub, err
	}
	return state, nil
}
