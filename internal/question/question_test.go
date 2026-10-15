package question_test

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/question"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

const (
	projectA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	projectB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
)

// store opens a fresh database with projects A and B, a proposal each,
// and open questions: Q-001, Q-002 and Q-010 in A, Q-001 in B.
func store(t *testing.T) (datadir.Dir, *sqlite.DB) {
	t.Helper()
	home := testutil.TempHome(t)
	db := open(t, home)
	write(t, db, `
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'a', '/r/a/.git', '01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', 'docs_only', '{}', 1),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4A7', 'b', '/r/b/.git', '01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', 'docs_only', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', '/r/a', '/r/a/.git', 1, 1, 1),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A7', '/r/b', '/r/b/.git', 1, 2, 1);
INSERT INTO proposals (id, seq, project_id, origin, actor, channel, worktree_id, summary, base_version)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', 3, '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'skill_init', 'agent:claude_code', 'agent_cli',
        '01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', 'initial', 0),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4P2', 4, '01J9Z3K5Q8R2T4V6W8X0Y2Z4A7', 'skill_init', 'agent:claude_code', 'agent_cli',
        '01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', 'initial', 0);
INSERT INTO questions (id, project_id, key, text, context, blocking, origin, proposal_id, created_seq)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4Q1', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'Q-001', 'e-mail or phone?', 'the brief says both', 1,
        'skill_init', '01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', 3),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4Q2', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'Q-002', 'which format?', NULL, 0,
        'skill_init', '01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', 3),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4Q3', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'Q-010', 'how many per page?', NULL, 0,
        'skill_init', '01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', 3),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4Q4', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A7', 'Q-001', 'another project''s question', NULL, 1,
        'skill_init', '01J9Z3K5Q8R2T4V6W8X0Y2Z4P2', 4);
UPDATE meta SET seq_counter = 4 WHERE id = 1;`)
	return home, db
}

func open(t *testing.T, home datadir.Dir) *sqlite.DB {
	t.Helper()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func write(t *testing.T, db *sqlite.DB, query string, args ...any) {
	t.Helper()
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// closeQuestion closes key of project A in its own transaction.
func closeQuestion(db *sqlite.DB, key string, action question.Action, text string) (int64, error) {
	var seq int64
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		seq, err = question.Close(ctx, tx, projectA, key, question.Closing{Action: action, Text: text, By: "human:me", Channel: "human_cli"})
		return err
	})
	return seq, err
}

// rows is what the questions table and the seq counter hold, read with SQL
// of its own.
func rows(t *testing.T, db *sqlite.DB) string {
	t.Helper()
	var out []string
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		r, err := tx.QueryContext(ctx, `SELECT project_id || ' ' || key || ' ' || status || ' ' || coalesce(answer, '-') || ' ' ||
    coalesce(answered_by, '-') || ' ' || coalesce(answered_channel, '-') || ' ' || coalesce(answered_seq, '-')
FROM questions ORDER BY project_id, key`)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		for r.Next() {
			var s string
			if err := r.Scan(&s); err != nil {
				return err
			}
			out = append(out, s)
		}
		var seq string
		if err := tx.QueryRowContext(ctx, `SELECT seq_counter FROM meta`).Scan(&seq); err != nil {
			return err
		}
		out = append(out, "seq "+seq)
		return r.Err()
	}); err != nil {
		t.Fatal(err)
	}
	return strings.Join(out, "\n")
}

func list(t *testing.T, db *sqlite.DB, project string, all bool) []question.Question {
	t.Helper()
	var out []question.Question
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		out, err = question.List(ctx, tx, project, all)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func keys(qs []question.Question) string {
	var out []string
	for _, q := range qs {
		out = append(out, q.Key+":"+string(q.Status))
	}
	return strings.Join(out, " ")
}

// An answer and a dismissal close a question once each, with one seq; the
// question of another project with the same key is not touched. A closed
// question is stale (exit code 5) for either action, with its status and
// seq, never its answer; nothing is written and no seq is taken. An
// unknown key is not_found (exit code 2).
func TestCloseOnce(t *testing.T) {
	_, db := store(t)
	seq, err := closeQuestion(db, "Q-001", question.Answer, "e-mail only")
	if err != nil || seq != 5 {
		t.Fatalf("answer: seq %d, %v", seq, err)
	}
	if seq, err = closeQuestion(db, "Q-002", question.Dismiss, "not needed now"); err != nil || seq != 6 {
		t.Fatalf("dismiss: seq %d, %v", seq, err)
	}
	want := strings.Join([]string{
		projectA + " Q-001 answered e-mail only human:me human_cli 5",
		projectA + " Q-002 dismissed not needed now human:me human_cli 6",
		projectA + " Q-010 open - - - -",
		projectB + " Q-001 open - - - -",
		"seq 6",
	}, "\n")
	if got := rows(t, db); got != want {
		t.Fatalf("questions:\n%s\nwant\n%s", got, want)
	}
	for _, c := range []struct {
		key    string
		action question.Action
		status string
		seq    int64
	}{
		{"Q-001", question.Answer, "answered", 5}, {"Q-001", question.Dismiss, "answered", 5},
		{"Q-002", question.Answer, "dismissed", 6}, {"Q-002", question.Dismiss, "dismissed", 6},
	} {
		_, err := closeQuestion(db, c.key, c.action, "again")
		e := apperr.From(err)
		if apperr.ExitCode(err) != apperr.ExitStalePrecondition || e.Current["status"] != c.status ||
			e.Current["question"] != c.key || e.Current["answered_seq"] != c.seq {
			t.Errorf("%s %s again: %v %+v", c.action, c.key, err, e)
		}
		if strings.Contains(e.Message, "e-mail only") || strings.Contains(e.Message, "not needed now") {
			t.Errorf("the error shows the answer: %s", e.Message)
		}
	}
	if _, err := closeQuestion(db, "Q-099", question.Answer, "x"); apperr.ExitCode(err) != apperr.ExitInvalidInput ||
		apperr.From(err).Code != apperr.NotFound {
		t.Errorf("unknown question: %v", err)
	}
	if got := rows(t, db); got != want {
		t.Fatalf("written by a refused close:\n%s\nwant\n%s", got, want)
	}
}

// List gives the open questions in key order (Q-010 after Q-002), or with
// all every one, with its answer; other projects' are not listed.
func TestList(t *testing.T) {
	_, db := store(t)
	if _, err := closeQuestion(db, "Q-002", question.Dismiss, "not needed now"); err != nil {
		t.Fatal(err)
	}
	if got := keys(list(t, db, projectA, false)); got != "Q-001:open Q-010:open" {
		t.Fatalf("open = %s", got)
	}
	all := list(t, db, projectA, true)
	if got := keys(all); got != "Q-001:open Q-002:dismissed Q-010:open" {
		t.Fatalf("all = %s", got)
	}
	q1, q2 := all[0], all[1]
	if q1.Answer != nil || q1.About != nil || q1.Context == nil || *q1.Context != "the brief says both" || !q1.Blocking ||
		q1.ProposalID != "01J9Z3K5Q8R2T4V6W8X0Y2Z4P1" || q1.CreatedSeq != 3 {
		t.Errorf("Q-001 = %+v", q1)
	}
	if a := q2.Answer; q2.Context != nil || a == nil || *a != (question.Closing{Action: question.Dismiss, Text: "not needed now", By: "human:me",
		Channel: "human_cli", Seq: 5}) {
		t.Errorf("Q-002 = %+v %+v", q2, a)
	}
	if got := keys(list(t, db, projectB, true)); got != "Q-001:open" {
		t.Errorf("project B = %s", got)
	}
}

// Two connections close the same question at once: one succeeds and the
// other is stale (exit code 5), whichever comes second; the answer is the
// first's, and one seq is taken.
func TestCloseRace(t *testing.T) {
	home, first := store(t)
	second := open(t, home)
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	for i, db := range []*sqlite.DB{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = closeQuestion(db, "Q-001", question.Answer, []string{"first", "second"}[i])
		}()
	}
	close(start)
	wg.Wait()
	won := -1
	for i, err := range errs {
		switch {
		case err == nil:
			won = i
		case apperr.ExitCode(err) != apperr.ExitStalePrecondition:
			t.Fatalf("connection %d: %v", i, err)
		}
	}
	if won < 0 || (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("results %v, want one success and one stale", errs)
	}
	text := []string{"first", "second"}[won]
	got := rows(t, first)
	if !strings.Contains(got, projectA+" Q-001 answered "+text+" human:me human_cli 5\n") || !strings.HasSuffix(got, "seq 5") {
		t.Fatalf("questions:\n%s", got)
	}
}

// The trigger keeps a closed question as it is even for a write that does
// not check its status (the second line of defense).
func TestClosedQuestionIsFinal(t *testing.T) {
	_, db := store(t)
	if _, err := closeQuestion(db, "Q-001", question.Answer, "e-mail only"); err != nil {
		t.Fatal(err)
	}
	before := rows(t, db)
	for _, change := range []string{
		`UPDATE questions SET answer = 'phone' WHERE key = 'Q-001' AND project_id = ?`,
		`UPDATE questions SET status = 'dismissed' WHERE key = 'Q-001' AND project_id = ?`,
	} {
		err := db.WriteTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, change, projectA)
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "an answered or dismissed question does not change") {
			t.Errorf("%s: %v", change, err)
		}
	}
	if after := rows(t, db); after != before {
		t.Fatalf("changed:\n%s", after)
	}
}
