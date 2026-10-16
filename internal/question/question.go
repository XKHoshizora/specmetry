// Package question reads the questions to clarify and closes them
// (data-model §4.8). A proposal opens a question (WP1.5); only a person
// answers or dismisses it (PD-14), once: a closed question never changes,
// which the questions_closed_is_final trigger also enforces. Answering
// does not change the baseline; a change it calls for goes through a
// proposal.
package question

import (
	"context"
	"database/sql"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

// Status is the state of a question.
type Status string

// The statuses of a question.
const (
	Open      Status = "open"
	Answered  Status = "answered"
	Dismissed Status = "dismissed"
)

// Action closes a question: an answer, or a dismissal with its rationale.
type Action string

// The actions of question-answer.v1.
const (
	Answer  Action = "answer"
	Dismiss Action = "dismiss"
)

// Closes is the status the action leaves.
func (a Action) Closes() Status {
	if a == Dismiss {
		return Dismissed
	}
	return Answered
}

// ListDoc is the question-list.v1 document.
type ListDoc struct {
	Schema    string     `json:"schema"`
	Questions []Question `json:"questions"`
}

// Question is a question as question-list.v1 shows it.
type Question struct {
	Key      string  `json:"key"`
	Text     string  `json:"text"`
	Context  *string `json:"context"`
	Status   Status  `json:"status"`
	Blocking bool    `json:"blocking"`
	// About is the key of the related requirement or criterion; the
	// proposal-local ref of an item not accepted, rejected ones included
	// (the question keeps naming it, data-model §4.8); or nil for the
	// project.
	About      *string `json:"about"`
	ProposalID string  `json:"proposal_id"`
	CreatedSeq int64   `json:"created_seq"`
	// Answer is nil while the question is open.
	Answer *Closing `json:"answer"`
}

// Closing is how a question was closed: the answer, or the dismissal and
// its rationale, by whom, on which channel, at which seq.
type Closing struct {
	Action  Action `json:"action"`
	Text    string `json:"text"`
	By      string `json:"by"`
	Channel string `json:"channel"`
	Seq     int64  `json:"seq"`
}

// The key of an accepted item's entity is found through the change that
// added it, so a question closed before its item was accepted shows the
// key too, as proposal show does.
const selectQuestions = `SELECT qu.key, qu.text, qu.context, qu.status, qu.blocking,
    coalesce(r.key, a.key, ar.key, aa.key), i.ref, qu.proposal_id, qu.created_seq,
    qu.answer, qu.answered_by, qu.answered_channel, qu.answered_seq
FROM questions qu
LEFT JOIN requirements r ON qu.related_type = 'requirement' AND r.project_id = qu.project_id AND r.id = qu.related_id
LEFT JOIN acceptance_criteria a ON qu.related_type = 'criterion' AND a.project_id = qu.project_id AND a.id = qu.related_id
LEFT JOIN proposal_items i ON i.project_id = qu.project_id AND i.id = qu.proposal_item_id
LEFT JOIN baseline_changes c ON qu.related_id IS NULL AND c.project_id = qu.project_id AND c.proposal_item_id = qu.proposal_item_id
    AND c.op = 'add'
LEFT JOIN requirements ar ON c.entity_type = 'requirement' AND ar.project_id = c.project_id AND ar.id = c.entity_id
LEFT JOIN acceptance_criteria aa ON c.entity_type = 'criterion' AND aa.project_id = c.project_id AND aa.id = c.entity_id
WHERE qu.project_id = ?`

// List reads the project's questions in key order: the open ones, or with
// all every one.
func List(ctx context.Context, q baseline.Querier, projectID string, all bool) ([]Question, error) {
	query := selectQuestions
	if !all {
		query += ` AND qu.status = 'open'`
	}
	return scan(q.QueryContext(ctx, query+` ORDER BY length(qu.key), qu.key`, projectID))
}

// ListDocument is the question-list.v1 document.
func ListDocument(ctx context.Context, q baseline.Querier, projectID string, all bool) (ListDoc, error) {
	list, err := List(ctx, q, projectID, all)
	return ListDoc{Schema: "specmetry.question-list/v1", Questions: list}, err
}

// Find reads the question with key.
func Find(ctx context.Context, q baseline.Querier, projectID, key string) (Question, bool, error) {
	list, err := scan(q.QueryContext(ctx, selectQuestions+` AND qu.key = ?`, projectID, key))
	if err != nil || len(list) == 0 {
		return Question{}, false, err
	}
	return list[0], true, nil
}

func scan(rows *sql.Rows, err error) ([]Question, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Question{}
	for rows.Next() {
		var x Question
		var context, about, ref, answer, by, channel sql.NullString
		var seq sql.NullInt64
		if err := rows.Scan(&x.Key, &x.Text, &context, &x.Status, &x.Blocking, &about, &ref, &x.ProposalID, &x.CreatedSeq,
			&answer, &by, &channel, &seq); err != nil {
			return nil, err
		}
		if context.Valid {
			x.Context = &context.String
		}
		switch {
		case about.Valid:
			x.About = &about.String
		case ref.Valid:
			x.About = &ref.String
		}
		if x.Status != Open {
			action := Answer
			if x.Status == Dismissed {
				action = Dismiss
			}
			x.Answer = &Closing{Action: action, Text: answer.String, By: by.String, Channel: channel.String, Seq: seq.Int64}
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// RequireOpen reads the question with key, in the caller's transaction,
// and checks it is still open (data-model §7.1): not_found (exit code 2)
// without one, stale_precondition (exit code 5) once it is closed.
func RequireOpen(ctx context.Context, q baseline.Querier, projectID, key string) (Question, error) {
	x, found, err := Find(ctx, q, projectID, key)
	switch {
	case err != nil:
		return Question{}, err
	case !found:
		// The key passed question-answer.v1 (Q- and digits): it is safe
		// to show.
		return Question{}, apperr.Newf(apperr.NotFound, "the project has no question %s (see question list)", key)
	case x.Status != Open:
		return x, Stale(x)
	}
	return x, nil
}

// Stale is the error for a question closed already: its status and the
// seq that closed it, never the answer.
func Stale(x Question) *apperr.Error {
	verb := "answered"
	if x.Status == Dismissed {
		verb = "dismissed"
	}
	var seq int64
	if x.Answer != nil {
		seq = x.Answer.Seq
	}
	e := apperr.Newf(apperr.StalePrecondition, "%s was %s at seq %d and does not change (see question list --all)", x.Key, verb, seq)
	e.Current = map[string]any{"question": x.Key, "status": string(x.Status), "answered_seq": seq}
	return e
}

// Close records the answer or dismissal of the open question with key, in
// the caller's write transaction (BEGIN IMMEDIATE, so no other connection
// closes it in between): c's action and text (the answer, or the
// dismissal's rationale), by and channel. It takes one seq, once the
// question is known to be open, and returns it; c.Seq is ignored. It
// writes nothing else: no baseline version, no entity. A question that is
// not open is stale, and nothing is written.
func Close(ctx context.Context, tx *sql.Tx, projectID, key string, c Closing) (int64, error) {
	if _, err := RequireOpen(ctx, tx, projectID, key); err != nil {
		return 0, err
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE questions SET status = ?, answer = ?, answered_by = ?, answered_channel = ?, answered_seq = ?
WHERE project_id = ? AND key = ? AND status = 'open'`, string(c.Action.Closes()), c.Text, c.By, c.Channel, seq, projectID, key)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, apperr.Newf(apperr.Internal, "closing %s changed %d questions", key, n)
	}
	return seq, nil
}
