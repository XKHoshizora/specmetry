package gitscan

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	testutil.DispatchFakeGit()
	testutil.DispatchHelper(map[string]func() int{"scan": helperScan})
	os.Exit(m.Run())
}

const (
	projectA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	projectB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
)

// env is a project A whose repository is repo, in a data directory of its
// own, with the requirements and criteria keys names.
type env struct {
	t    *testing.T
	home datadir.Dir
	db   *sqlite.DB
	repo *testutil.GitRepo
	// scratch is in the database's own data directory, as a command
	// would make it.
	scratch *gitx.Scratch
	ids     map[string]string // by key
	n       int
	clock   int64
	files   int
}

func newEnv(t *testing.T, keys ...string) *env {
	t.Helper()
	home := testutil.TempHome(t)
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := testutil.NewGitRepo(t)
	scratch, err := gitx.NewScratch(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = scratch.Close() })
	e := &env{t: t, home: home, db: db, repo: repo, scratch: scratch, ids: map[string]string{}, clock: 1_700_000_000_000}
	common := gitx.Canonical(filepath.Join(repo.Dir, ".git"))
	e.exec(`INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES (?1, 'a', ?2, '01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', 'in_development', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', ?1, ?3, ?2, 1, 1, 1);
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES (?4, 'b', '/r/b/.git', '01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', 'in_development', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', ?4, '/r/b', '/r/b/.git', 1, 2, 1);
UPDATE meta SET seq_counter = 2 WHERE id = 1;`, projectA, common, repo.Dir, projectB)
	for _, k := range keys {
		e.addKey(projectA, k, "active")
	}
	return e
}

func (e *env) exec(query string, args ...any) {
	e.t.Helper()
	if err := e.db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) rows(query string, args ...any) []string {
	e.t.Helper()
	var out []string
	if err := e.db.WriteTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]sql.NullString, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			var f []string
			for _, v := range vals {
				if v.Valid {
					f = append(f, v.String)
				} else {
					f = append(f, "NULL")
				}
			}
			out = append(out, strings.Join(f, " "))
		}
		return rows.Err()
	}); err != nil {
		e.t.Fatal(err)
	}
	return out
}

// addKey adds a requirement or a criterion (and the criterion's
// requirement, active, if it has none yet) to a project.
func (e *env) addKey(project, key, status string) {
	e.t.Helper()
	e.n++
	id := fmt.Sprintf("01J9Z3K5Q8R2T4V6W8X0Y2Z%03d", e.n)
	retired := any(nil)
	if status == "retired" {
		retired = 1
	}
	if strings.HasPrefix(key, "AC-") {
		parts := strings.Split(key, "-")
		req := "REQ-" + parts[1]
		if _, ok := e.ids[project+req]; !ok {
			e.addKey(project, req, "active")
		}
		e.exec(`INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, retired_version, updated_seq, requirement_id, requirement_revision, statement, verification)
VALUES (?, ?, ?, ?, 1, 1, 'extracted', 1, 1, ?, 1, ?, 1, 'it works', 'manual')`, id, project, key, status, retired, e.ids[project+req])
	} else {
		e.exec(`INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, retired_version, updated_seq, title, kind, priority)
VALUES (?, ?, ?, ?, 1, 1, 'extracted', 1, 1, ?, 1, 't', 'functional', 'must')`, id, project, key, status, retired)
	}
	e.ids[project+key] = id
}

// commit commits a new file with message in dir (the main worktree when
// "").
func (e *env) commit(dir, message string) string {
	e.t.Helper()
	e.files++
	if dir == "" {
		dir = e.repo.Dir
	}
	name := filepath.Join(dir, fmt.Sprintf("f%03d.txt", e.files))
	if err := os.WriteFile(name, []byte(message), 0o644); err != nil {
		e.t.Fatal(err)
	}
	e.repo.GitIn(dir, "add", "-A")
	e.repo.GitIn(dir, "commit", "-q", "-m", message)
	return e.repo.GitIn(dir, "rev-parse", "HEAD")
}

func (e *env) scan() (Result, error) {
	e.t.Helper()
	scanner, err := secrets.New(nil)
	if err != nil {
		e.t.Fatal(err)
	}
	return Scan(ctx, e.db, projectA, e.repo.Dir, Options{Secrets: scanner, Scratch: e.scratch, Now: func() time.Time {
		e.clock += 1000
		return time.UnixMilli(e.clock)
	}})
}

func (e *env) mustScan() Result {
	e.t.Helper()
	r, err := e.scan()
	if err != nil {
		e.t.Fatalf("Scan: %v", err)
	}
	return r
}

// dbState is everything a scan could write: the seq counter, the
// commits, links and cursors, and the row count of every table.
type dbState struct {
	Seq     string
	Commits []string
	Links   []string
	Cursors []string
	Tables  []string
}

func (e *env) state() dbState {
	e.t.Helper()
	var s dbState
	s.Seq = e.rows(`SELECT seq_counter FROM meta WHERE id = 1`)[0]
	s.Commits = e.rows(`SELECT project_id, sha, seq, committed_at FROM commits ORDER BY project_id, sha`)
	s.Links = e.rows(`SELECT c.sha, l.target_type, coalesce(r.key, a.key), l.link_source FROM commit_links l
JOIN commits c ON c.id = l.commit_id
LEFT JOIN requirements r ON l.target_type = 'requirement' AND r.id = l.target_id
LEFT JOIN acceptance_criteria a ON l.target_type = 'criterion' AND a.id = l.target_id
ORDER BY c.sha, l.target_type, 3`)
	s.Cursors = e.rows(`SELECT kind, name, last_tip, scanned_seq, scanned_at FROM git_scan_cursors ORDER BY kind, name`)
	for _, table := range e.rows(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`) {
		s.Tables = append(s.Tables, table+" "+e.rows(`SELECT count(*) FROM "` + table + `"`)[0])
	}
	return s
}

func (e *env) seq() int64 {
	e.t.Helper()
	var n int64
	_, _ = fmt.Sscan(e.state().Seq, &n)
	return n
}

func requireUnchanged(t *testing.T, name string, before, after dbState) {
	t.Helper()
	b, _ := json.MarshalIndent(before, "", " ")
	a, _ := json.MarshalIndent(after, "", " ")
	if string(a) != string(b) {
		t.Errorf("%s changed the database:\nbefore %s\nafter %s", name, b, a)
	}
}

func requireSafeGit(t *testing.T) {
	t.Helper()
	if os.Getenv("SPECMETRY_TEST_GIT_OBJECTS") == "unsafe" {
		t.Skip("the git on PATH may not read objects (SPECMETRY_TEST_GIT_OBJECTS=unsafe)")
	}
}

func loose(repo *testutil.GitRepo, oid string) string {
	return filepath.Join(repo.Dir, ".git", "objects", oid[:2], oid[2:])
}

func removeObject(t *testing.T, repo *testutil.GitRepo, oid string) {
	t.Helper()
	p := loose(repo, oid)
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
}

func ms(t *testing.T, repo *testutil.GitRepo, sha string) string {
	t.Helper()
	var s int64
	if _, err := fmt.Sscan(repo.Git("log", "-1", "--format=%ct", sha), &s); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprint(s * 1000)
}

// L19, M1 to M11, MVP-EV-14 and EV-16: a scan reads the new commits and
// stores, in one write transaction with one seq, those that mention at
// least one key of the project (retired ones included), with their links
// (a trailer mention over a message one), and the cursors; the SHA, the
// committer time in milliseconds and the links only.
func TestScanStoresTheCommitsThatMentionKeys(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001", "REQ-002", "AC-001-1")
	e.addKey(projectA, "REQ-003", "retired")
	e.addKey(projectB, "REQ-009", "active")
	e.commit("", "init")
	c2 := e.commit("", "fix REQ-001\n\nSpecmetry-Ref: AC-001-1")
	e.commit("", "only REQ-009, another project's, and REQ-999, nobody's")
	c4 := e.commit("", "retire REQ-003 and REQ-002\n\nSpecmetry-Ref: REQ-002")
	before := e.seq()
	r := e.mustScan()
	if r != (Result{Read: 4, Stored: 2}) {
		t.Errorf("Result = %+v", r)
	}
	s := fmt.Sprint(before + 1)
	st := e.state()
	if st.Seq != s {
		t.Errorf("seq counter %s, want %s (one seq for the scan)", st.Seq, s)
	}
	want := []string{projectA + " " + c2 + " " + s + " " + ms(t, e.repo, c2), projectA + " " + c4 + " " + s + " " + ms(t, e.repo, c4)}
	slices.Sort(want)
	if !slices.Equal(st.Commits, want) {
		t.Errorf("commits %q\nwant %q", st.Commits, want)
	}
	wantLinks := []string{
		c2 + " criterion AC-001-1 trailer", c2 + " requirement REQ-001 message",
		c4 + " requirement REQ-002 trailer", c4 + " requirement REQ-003 message",
	}
	slices.Sort(wantLinks)
	if !slices.Equal(st.Links, wantLinks) {
		t.Errorf("links %q\nwant %q", st.Links, wantLinks)
	}
	if len(st.Cursors) != 1 || !strings.HasPrefix(st.Cursors[0], "branch refs/heads/main "+c4+" "+s+" ") {
		t.Errorf("cursors %q", st.Cursors)
	}
}

// L4 and WP2.3 spec §5.5 (decision 3 of 2026-10-18): a scan that stores no
// new commit allocates no seq and leaves the counter alone, but moves the
// cursors; a first scan without matches makes cursors with scanned_seq
// NULL; scanned_seq is the seq of the last scan that stored commits and
// wrote the cursor.
func TestScanWithoutNewCommits(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	e.commit("", "nothing")
	seq := e.state().Seq
	r := e.mustScan()
	st := e.state()
	tip := e.repo.Git("rev-parse", "HEAD")
	if r != (Result{Read: 1}) || st.Seq != seq || len(st.Commits) != 0 ||
		len(st.Cursors) != 1 || !strings.HasPrefix(st.Cursors[0], "branch refs/heads/main "+tip+" NULL ") {
		t.Fatalf("first scan without matches: %+v, %+v", r, st)
	}
	e.commit("", "still nothing")
	e.mustScan()
	st2 := e.state()
	tip = e.repo.Git("rev-parse", "HEAD")
	if st2.Seq != seq || !strings.HasPrefix(st2.Cursors[0], "branch refs/heads/main "+tip+" NULL ") || st2.Cursors[0] == st.Cursors[0] {
		t.Errorf("a scan without matches: %+v", st2)
	}
	e.commit("", "REQ-001")
	e.mustScan()
	s := e.state()
	if s.Seq == seq || !strings.Contains(s.Cursors[0], " "+s.Seq+" ") {
		t.Errorf("a scan with a match: %+v", s)
	}
	e.commit("", "nothing again")
	r = e.mustScan()
	s2 := e.state()
	if r.Stored != 0 || s2.Seq != s.Seq || !strings.Contains(s2.Cursors[0], " "+s.Seq+" ") {
		t.Errorf("scanned_seq after a scan without matches: %+v", s2)
	}
	// Nothing new at all: the cursor's scanned_at still moves.
	r = e.mustScan()
	s3 := e.state()
	if r.Read != 0 || s3.Seq != s.Seq || s3.Cursors[0] == s2.Cursors[0] {
		t.Errorf("a scan with nothing to read: %+v, %+v", r, s3)
	}
}

// L1, L2, L3 and review 044 N1 (WP2.3 spec §5.4, MVP-EV-15): a commit's
// links are fixed when it is first stored and never added to; a commit
// not stored is read again only when a later scan's range takes it in (a
// rewrite), and is then stored with that scan's seq.
func TestScanLinkLifecycle(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	x := e.commit("", "X names REQ-020, not a key yet")
	w := e.commit("", "W names REQ-001 and REQ-021")
	e.mustScan()
	if st := e.state(); len(st.Commits) != 1 || !strings.Contains(st.Commits[0], w) {
		t.Fatalf("L1: X is not stored, W is: %+v", st.Commits)
	}
	firstLinks := e.state().Links
	e.addKey(projectA, "REQ-020", "active")
	e.addKey(projectA, "REQ-021", "active")
	e.commit("", "Y")
	if r := e.mustScan(); r.Read != 1 || r.Stored != 0 {
		t.Errorf("L1: the increment reads only Y: %+v", r)
	}
	if st := e.state(); !slices.Equal(st.Links, firstLinks) || len(st.Commits) != 1 {
		t.Errorf("L2: links added to a stored commit, or X stored by an increment: %+v", st)
	}
	// L3: a rewrite keeps X and W in the history; the cursor's tip Y is no
	// ancestor of the new tip, so the whole history is read again.
	e.repo.Git("reset", "-q", "--hard", w)
	e.commit("", "Z")
	seq := e.seq()
	if r := e.mustScan(); r.Read != 3 || r.Stored != 1 {
		t.Errorf("L3: the rescan: %+v", r)
	}
	st := e.state()
	want := []string{projectA + " " + w, projectA + " " + x}
	slices.Sort(want)
	got := []string{}
	for _, c := range st.Commits {
		f := strings.Fields(c)
		got = append(got, f[0]+" "+f[1])
		if f[1] == x && f[2] != fmt.Sprint(seq+1) {
			t.Errorf("L3: X stored with seq %s, want this scan's %d", f[2], seq+1)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("L3: commits %q, want %q", got, want)
	}
	wantLinks := append(slices.Clone(firstLinks), x+" requirement REQ-020 message")
	slices.Sort(wantLinks)
	if !slices.Equal(st.Links, wantLinks) {
		t.Errorf("L2 and L3: links %q\nwant %q (W's REQ-021 never added)", st.Links, wantLinks)
	}
}

// L5 to L8 and L20 (D2, D4, MVP-WT-04, MVP-WT-06): every branch and every
// worktree's detached HEAD has its cursor; a commit two refs share is
// stored once; a new branch off a scanned one reads only its own commits;
// a reset branch is read again without duplicates; a moved worktree is a
// new cursor and one made again at the old path takes the old cursor.
func TestScanRefs(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001", "REQ-002")
	shared := e.commit("", "shared REQ-001")
	e.repo.Git("branch", "feature")
	e.mustScan()
	st := e.state()
	if len(st.Commits) != 1 || len(st.Links) != 1 || len(st.Cursors) != 2 {
		t.Fatalf("L5: %+v", st)
	}
	// L6: a new branch off main with one commit.
	e.repo.Git("checkout", "-q", "-b", "topic")
	e.commit("", "topic REQ-002")
	if r := e.mustScan(); r.Read != 1 || r.Stored != 1 {
		t.Errorf("L6: %+v", r)
	}
	// L7: feature reset and committed again.
	e.repo.Git("checkout", "-q", "feature")
	e.commit("", "feature one")
	e.mustScan()
	e.repo.Git("reset", "-q", "--hard", shared)
	again := e.commit("", "feature again REQ-001")
	r := e.mustScan()
	if r.Stored != 1 {
		t.Errorf("L7: %+v", r)
	}
	if n := len(e.rows(`SELECT 1 FROM commits WHERE sha = ?`, shared)); n != 1 {
		t.Errorf("L7: the shared commit stored %d times", n)
	}
	if n := len(e.rows(`SELECT 1 FROM commits WHERE sha = ?`, again)); n != 1 {
		t.Errorf("L7: the new commit stored %d times", n)
	}
	// L8: an unregistered worktree's detached HEAD.
	wtPath := filepath.Join(filepath.Dir(e.repo.Dir), "wt-detached")
	e.repo.Git("worktree", "add", "-q", "--detach", wtPath, shared)
	detached := e.commit(wtPath, "detached REQ-002")
	if r := e.mustScan(); r.Stored != 1 {
		t.Errorf("L8: %+v", r)
	}
	name := gitx.Canonical(wtPath)
	if c := e.rows(`SELECT last_tip FROM git_scan_cursors WHERE kind = 'detached' AND name = ?`, name); len(c) != 1 || c[0] != detached {
		t.Errorf("L8: the detached cursor %q", c)
	}
	// L20: moved: a new cursor, the old one kept; a new worktree at the old
	// path takes the old cursor, read as an increment.
	moved := filepath.Join(filepath.Dir(e.repo.Dir), "wt-moved")
	e.repo.Git("worktree", "move", wtPath, moved)
	if r := e.mustScan(); r.Read != 0 || r.Stored != 0 {
		t.Errorf("L20 moved: %+v", r)
	}
	if n := len(e.rows(`SELECT 1 FROM git_scan_cursors WHERE kind = 'detached'`)); n != 2 {
		t.Errorf("L20: %d detached cursors, want 2", n)
	}
	e.repo.Git("worktree", "add", "-q", "--detach", wtPath, detached)
	e.commit(wtPath, "again at the old path")
	if r := e.mustScan(); r.Read != 1 {
		t.Errorf("L20 the old path again: %+v", r)
	}
}

// L11 (D7): a branch name or a detached worktree's path that the secret
// rules hit refuses the whole scan (invalid_input, exit 2) before any
// commit is read: nothing written, nothing echoed.
func TestScanRefusesSecretNames(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name string
		make func(e *env, secret string)
	}{
		{"a branch", func(e *env, secret string) { e.repo.Git("branch", "feature/"+secret) }},
		{"a detached worktree's path", func(e *env, secret string) {
			e.repo.Git("worktree", "add", "-q", "--detach", filepath.Join(filepath.Dir(e.repo.Dir), "wt-"+secret), "HEAD")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t, "REQ-001")
			e.commit("", "REQ-001")
			e.mustScan()
			e.commit("", "REQ-001 again")
			secret := testutil.NewSecretCanary(t)
			c.make(e, secret)
			before := e.state()
			r, err := e.scan()
			if apperr.ExitCode(err) != apperr.ExitInvalidInput || r != (Result{}) {
				t.Fatalf("Scan = %+v, %v; want invalid_input", r, err)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(fmt.Sprintf("%+v", apperr.From(err)), secret) {
				t.Error("the error echoes the secret")
			}
			requireUnchanged(t, "a refused scan", before, e.state())
			if hits := testutil.FindInTree(t, e.home.Path, secret); len(hits) != 0 {
				t.Errorf("the secret is in the data directory: %v", hits)
			}
		})
	}
}

// L12 (D5): a project with no requirement or criterion yet is not
// scanned at all: no git command runs, no cursor is made.
func TestScanWithoutKeys(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t)
	e.commit("", "REQ-001")
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	before := e.state()
	r, err := e.scan()
	if err != nil || r != (Result{NotScanned: true}) {
		t.Errorf("Scan = %+v, %v; want NotScanned", r, err)
	}
	requireUnchanged(t, "a scan of a project without keys", before, e.state())
	if c := calls(); len(c) != 0 {
		t.Errorf("git ran: %v", c)
	}
}

// L10, L14, L16, L18, L21 to L25 (review 050 B1, review 052 B1-R and
// B2-R, review 055 B2-R2, P3): whatever makes a read fail or the state of
// the objects unsure fails the whole scan with an environment error (exit
// 6), and it writes nothing: no commit, link, seq or cursor, not even the
// commits read before the failure; the next scan starts from the same
// cursors.
func TestScanFailuresWriteNothing(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name string
		// damage sets the case up after a first scan of A (REQ-001) and B
		// (REQ-001, the cursor's tip), with C (REQ-001) committed on top.
		damage func(t *testing.T, e *env, a, b, cc string) (undo func())
	}{
		{"L10 a git without worktree list -z", func(t *testing.T, _ *env, _, _, _ string) func() {
			dir, env, _ := testutil.PassthroughGit(t, testutil.PassWithoutWorktreeListZ)
			for _, kv := range env {
				k, v, _ := strings.Cut(kv, "=")
				t.Setenv(k, v)
			}
			path := os.Getenv("PATH")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+path)
			return func() { t.Setenv("PATH", path) }
		}},
		{"L14 the branch's tip unreadable", func(t *testing.T, e *env, _, _, cc string) func() {
			saved, err := os.ReadFile(loose(e.repo, cc))
			if err != nil {
				t.Fatal(err)
			}
			removeObject(t, e.repo, cc)
			return func() {
				if err := os.WriteFile(loose(e.repo, cc), saved, 0o444); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"L16 the cursor's old tip a damaged loose object", func(t *testing.T, e *env, _, b, _ string) func() {
			if err := os.Chmod(loose(e.repo, b), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(loose(e.repo, b), []byte("not zlib"), 0o644); err != nil {
				t.Fatal(err)
			}
			return nil
		}},
		{"a commit of the range missing (is-ancestor fails, or says no and the whole history is read)", func(t *testing.T, e *env, _, _, _ string) func() {
			mid := e.commit("", "D REQ-001")
			e.commit("", "E REQ-001")
			removeObject(t, e.repo, mid)
			return nil
		}},
		{"L22 the old tip's .pack lost its .idx", func(t *testing.T, e *env, _, _, _ string) func() {
			e.repo.Git("repack", "-adq")
			e.repo.Git("prune-packed")
			idx, _ := filepath.Glob(filepath.Join(e.repo.Dir, ".git", "objects", "pack", "*.idx"))
			if len(idx) != 1 {
				t.Fatalf("packs: %v", idx)
			}
			away := filepath.Join(t.TempDir(), "away.idx")
			if err := os.Rename(idx[0], away); err != nil {
				t.Fatal(err)
			}
			return func() {
				if err := os.Rename(away, idx[0]); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"L23 grafts hiding a missing parent", func(t *testing.T, e *env, _, _, _ string) func() {
			// A rewritten branch (no ancestor of the cursor) is read whole:
			// grafts that cut it at its tip would hide a missing root.
			e.repo.Git("checkout", "-q", "--orphan", "other")
			root := e.commit("", "root REQ-001")
			tip := e.commit("", "tip REQ-001")
			e.repo.Git("checkout", "-q", "main")
			e.repo.Git("update-ref", "refs/heads/main", tip)
			grafts := filepath.Join(e.repo.Dir, ".git", "info", "grafts")
			if err := os.MkdirAll(filepath.Dir(grafts), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(grafts, []byte(tip+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			removeObject(t, e.repo, root)
			return nil
		}},
		{"L24 a shallow clone", func(t *testing.T, e *env, _, b, _ string) func() {
			p := filepath.Join(e.repo.Dir, ".git", "shallow")
			if err := os.WriteFile(p, []byte(b+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return func() {
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"L25 a shallow entry that is a directory", func(t *testing.T, e *env, _, _, _ string) func() {
			if err := os.Mkdir(filepath.Join(e.repo.Dir, ".git", "shallow"), 0o755); err != nil {
				t.Fatal(err)
			}
			return nil
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t, "REQ-001")
			canary := testutil.NewCanary(t)
			a := e.commit("", "A REQ-001 "+canary)
			b := e.commit("", "B REQ-001")
			e.mustScan()
			cc := e.commit("", "C REQ-001\n\n"+canary+"\n\nSpecmetry-Ref: REQ-001")
			undo := c.damage(t, e, a, b, cc)
			before := e.state()
			r, err := e.scan()
			if apperr.ExitCode(err) != apperr.ExitEnvironment || r != (Result{}) {
				t.Fatalf("Scan = %+v, %v; want an environment error", r, err)
			}
			if strings.Contains(err.Error(), canary) {
				t.Error("the error echoes a commit message")
			}
			requireUnchanged(t, c.name, before, e.state())
			if undo == nil {
				return
			}
			undo()
			if r := e.mustScan(); r.Stored != 1 {
				t.Errorf("the next scan, once git can read again: %+v; want C stored", r)
			}
		})
	}
}

// L21 (G3): git log may print commits before it fails on a missing one;
// the scan keeps none of them, nor writes a cursor. Here the reads have
// no ancestry check before them: a first scan (the whole history) and a
// new ref (its tip, excluding the old tips).
func TestScanDropsAPartialRead(t *testing.T) {
	requireSafeGit(t)
	t.Run("a first scan", func(t *testing.T) {
		e := newEnv(t, "REQ-001")
		e.commit("", "A REQ-001")
		b := e.commit("", "B REQ-001")
		e.commit("", "C REQ-001")
		removeObject(t, e.repo, b)
		before := e.state()
		if r, err := e.scan(); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Fatalf("Scan = %+v, %v; want an environment error", r, err)
		}
		requireUnchanged(t, "a first scan that failed", before, e.state())
	})
	t.Run("a new ref", func(t *testing.T) {
		e := newEnv(t, "REQ-001")
		e.commit("", "A REQ-001")
		e.mustScan()
		e.repo.Git("checkout", "-q", "-b", "feature")
		f1 := e.commit("", "F1 REQ-001")
		e.commit("", "F2 REQ-001")
		removeObject(t, e.repo, f1)
		before := e.state()
		if r, err := e.scan(); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Fatalf("Scan = %+v, %v; want an environment error", r, err)
		}
		requireUnchanged(t, "a new ref's read that failed", before, e.state())
	})
}

// L18 (review 052 B1-R, G1): an exclusion classified present and then
// gone (here: removed after the plan, with a commit-graph that still knows
// it) fails the read; the commit-graph does not hide it.
func TestScanExclusionGoneAfterThePlan(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	e.commit("", "A")
	b := e.commit("", "B REQ-001")
	e.mustScan()
	e.repo.Git("branch", "feature")
	e.commit("", "C REQ-001")
	e.repo.Git("commit-graph", "write", "--reachable")
	testHookBeforeRead = func() { removeObject(t, e.repo, b) }
	t.Cleanup(func() { testHookBeforeRead = nil })
	before := e.state()
	if r, err := e.scan(); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("Scan = %+v, %v; want an environment error", r, err)
	}
	requireUnchanged(t, "L18", before, e.state())
}

// L15 and L17 (B2): a cursor whose old tip was reclaimed after a rewrite
// is read as rewritten (the whole history, stored commits not stored
// again); for a new ref, a reclaimed old tip excludes nothing, the others
// still do.
func TestScanReclaimedOldTips(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	x := e.commit("", "X REQ-001")
	e.repo.Git("checkout", "-q", "-b", "gone")
	e.commit("", "on gone")
	e.repo.Git("checkout", "-q", "main")
	y := e.commit("", "Y")
	e.mustScan()
	// main rewritten: Y dropped and reclaimed; gone deleted and reclaimed.
	e.repo.Git("reset", "-q", "--hard", x)
	e.commit("", "Z REQ-001")
	e.repo.Git("branch", "-D", "gone")
	e.repo.Git("reflog", "expire", "--expire=now", "--all")
	e.repo.Git("gc", "-q", "--prune=now")
	if _, err := os.Stat(loose(e.repo, y)); err == nil {
		t.Fatal("gc kept Y")
	}
	e.repo.Git("branch", "feature", x)
	e.repo.Git("checkout", "-q", "feature")
	e.commit("", "F REQ-001")
	r := e.mustScan()
	// main: X and Z (Y's cursor reclaimed: whole history); feature: new,
	// excluding nothing present but main's old tip Y, which is gone, and
	// gone's, also gone: its whole history, X and F.
	if r.Read != 3 || r.Stored != 2 {
		t.Errorf("Result = %+v; want 3 read (X, Z, F), 2 stored (Z, F)", r)
	}
	if n := len(e.rows(`SELECT 1 FROM commits WHERE sha = ?`, x)); n != 1 {
		t.Errorf("X stored %d times", n)
	}
}

// P3 (H3, H4, L26): refused while shallow, with no cursor made or moved;
// once the repository is complete again, a repository never scanned is
// read whole, and one scanned before only from its cursor.
func TestScanShallowLifecycle(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	e.commit("", "A REQ-001")
	b := e.commit("", "B")
	e.commit("", "C REQ-001")
	shallow := filepath.Join(e.repo.Dir, ".git", "shallow")
	if err := os.WriteFile(shallow, []byte(b+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.scan(); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("H3 shallow: %v", err)
	}
	if st := e.state(); len(st.Cursors) != 0 {
		t.Fatalf("a cursor made while shallow: %q", st.Cursors)
	}
	if err := os.Remove(shallow); err != nil {
		t.Fatal(err)
	}
	if r := e.mustScan(); r.Read != 3 || r.Stored != 2 {
		t.Errorf("H3 complete again, the first scan: %+v", r)
	}
	if err := os.WriteFile(shallow, []byte(b+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := e.state()
	if _, err := e.scan(); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("H4 shallow again: %v", err)
	}
	requireUnchanged(t, "H4", before, e.state())
	if err := os.Remove(shallow); err != nil {
		t.Fatal(err)
	}
	e.commit("", "D REQ-001")
	if r := e.mustScan(); r.Read != 1 || r.Stored != 1 {
		t.Errorf("H4 complete again, the increment: %+v", r)
	}
}

// L13 (MVP-EV-11, architecture §10.2 item 6): a canary in a commit's
// subject, body and trailer value is in no file of the data directory
// (database, WAL, shared memory) after the scan stored the commit.
func TestScanKeepsNoMessage(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	subject, body, trailer := testutil.NewCanary(t), testutil.NewCanary(t), testutil.NewCanary(t)
	e.commit("", "fix REQ-001 "+subject+"\n\n"+body+"\n\nSpecmetry-Ref: REQ-001 "+trailer)
	if r := e.mustScan(); r.Stored != 1 {
		t.Fatalf("Result = %+v", r)
	}
	if hits := testutil.FindInTree(t, e.home.Path, subject, body, trailer); len(hits) != 0 {
		t.Errorf("message text in the data directory: %v", hits)
	}
}

// L9 (D6): when the cursors changed between the plan and the write (here:
// another scan ran in between, moving one and making another), nothing is
// written and the result says so; the other scan's writes stand.
func TestScanConcurrentSkip(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	e.commit("", "A REQ-001")
	e.mustScan()
	e.commit("", "B REQ-001")
	e.repo.Git("branch", "feature")
	var inner Result
	// The state the inner scan left, taken before the outer one goes on to
	// its write (review 058 N1: taken after the outer one returned, it
	// could not show an outer write).
	var afterInner dbState
	testHookBeforeWrite = func() {
		testHookBeforeWrite = nil
		inner = e.mustScan()
		afterInner = e.state()
	}
	t.Cleanup(func() { testHookBeforeWrite = nil })
	r, err := Scan(ctx, e.db, projectA, e.repo.Dir, Options{Secrets: mustScanner(t), Scratch: e.scratch, Now: time.Now})
	if err != nil || !r.Concurrent || r.Stored != 0 {
		t.Fatalf("the outer scan: %+v, %v; want a concurrent skip", r, err)
	}
	if inner.Stored != 1 {
		t.Errorf("the inner scan: %+v", inner)
	}
	if n := len(e.rows(`SELECT 1 FROM commits`)); n != 2 {
		t.Errorf("%d commits, want 2", n)
	}
	requireUnchanged(t, "the outer scan, after the inner one", afterInner, e.state())
}

func mustScanner(t *testing.T) *secrets.Scanner {
	t.Helper()
	s, err := secrets.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Two processes (WP2.3 spec §11): one plans, then waits while the other
// scans and writes; the first then finds its plan out of date and writes
// nothing. The database ends as the second process left it.
func TestScanConcurrentProcesses(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	e.commit("", "A REQ-001")
	dir := t.TempDir()
	planned, proceed := filepath.Join(dir, "planned"), filepath.Join(dir, "proceed")
	cmd := testutil.HelperCommand(t, "scan", "SCAN_PLANNED="+planned, "SCAN_PROCEED="+proceed,
		"SCAN_PROJECT="+projectA, "SCAN_WORKTREE="+e.repo.Dir)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if !waitFor(planned, 60*time.Second) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("the helper did not plan: %s", out.String())
	}
	second := e.mustScan()
	afterSecond := e.state()
	if err := os.WriteFile(proceed, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("the helper failed: %v: %s", err, out.String())
	}
	var first Result
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &first); err != nil {
		t.Fatalf("the helper printed %q: %v", out.String(), err)
	}
	if !first.Concurrent || first.Stored != 0 || second.Stored != 1 {
		t.Errorf("first %+v, second %+v", first, second)
	}
	requireUnchanged(t, "the first process", afterSecond, e.state())
}

func waitFor(p string, d time.Duration) bool {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// helperScan is the first process of TestScanConcurrentProcesses: it
// scans, telling the test when it has planned and waiting for it before
// it writes, then prints the result.
func helperScan() int {
	home, err := datadir.Resolve()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer func() { _ = db.Close() }()
	testHookBeforeWrite = func() {
		_ = os.WriteFile(os.Getenv("SCAN_PLANNED"), nil, 0o600)
		waitFor(os.Getenv("SCAN_PROCEED"), 60*time.Second)
	}
	scanner, err := secrets.New(nil)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	scratch, err := gitx.NewScratch(home)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer func() { _ = scratch.Close() }()
	r, err := Scan(ctx, db, os.Getenv("SCAN_PROJECT"), os.Getenv("SCAN_WORKTREE"), Options{Secrets: scanner, Scratch: scratch})
	if err != nil {
		fmt.Println(err)
		return 1
	}
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
	return 0
}

// A worktree of another repository than the project's is refused before
// anything is read or written.
func TestScanChecksTheRepository(t *testing.T) {
	requireSafeGit(t)
	e := newEnv(t, "REQ-001")
	e.commit("", "REQ-001")
	other := testutil.NewGitRepo(t)
	other.WriteFile("a", "REQ-001")
	other.Commit("REQ-001")
	before := e.state()
	if r, err := Scan(ctx, e.db, projectA, other.Dir, Options{Secrets: mustScanner(t), Scratch: e.scratch}); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("Scan = %+v, %v; want an internal error", r, err)
	}
	requireUnchanged(t, "another repository", before, e.state())
	if _, err := Scan(ctx, e.db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4ZZ", e.repo.Dir, Options{Secrets: mustScanner(t), Scratch: e.scratch}); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("an unknown project: %v", err)
	}
	if _, err := Scan(ctx, e.db, projectA, e.repo.Dir, Options{}); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("no secret scanner: %v", err)
	}
	if _, err := Scan(ctx, e.db, projectA, e.repo.Dir, Options{Secrets: mustScanner(t)}); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("no Scratch: %v", err)
	}
}
