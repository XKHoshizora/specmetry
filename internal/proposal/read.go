package proposal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
)

// Stored is a submitted proposal with its items, as stored.
type Stored struct {
	ID          string
	Seq         int64
	Origin      string
	Actor       string
	Channel     string
	Summary     string
	BaseVersion int
	Items       []StoredItem
}

// StoredItem is a stored proposal item and its review.
type StoredItem struct {
	ID             string
	Index          int
	Op             string
	Type           baseline.Type
	TargetID       string
	TargetKey      string
	TargetRevision int
	Ref            string
	// Payload is the submitted data; nil for retire and reaffirm.
	Payload            map[string]json.RawMessage
	SuggestedClass     string
	Sources            []baseline.SourceRef
	CodeRefs           []baseline.CodeRef
	Rationale          string
	Origin, Confidence string
	State              string
	FinalPayload       json.RawMessage
	ChangeClass        string
	ReviewedBy         string
	ReviewedChannel    string
	ReviewedSeq        int64
	ReviewNote         string
	AppliedVersionID   string
}

// Reviewed reports whether the item has been decided.
func (it StoredItem) Reviewed() bool { return it.ReviewedSeq != 0 }

// Load reads a proposal of the project with its items in order; found is
// false when the project has none with that ID.
func Load(ctx context.Context, q baseline.Querier, projectID, id string) (Stored, bool, error) {
	ps, err := load(ctx, q, projectID, id)
	if err != nil || len(ps) == 0 {
		return Stored{}, false, err
	}
	return ps[0], true, nil
}

// List reads every proposal of the project, newest first (by seq).
func List(ctx context.Context, q baseline.Querier, projectID string) ([]Stored, error) {
	return load(ctx, q, projectID, "")
}

func load(ctx context.Context, q baseline.Querier, projectID, id string) ([]Stored, error) {
	query := `SELECT id, seq, origin, actor, channel, summary, base_version FROM proposals WHERE project_id = ?`
	args := []any{projectID}
	if id != "" {
		query += ` AND id = ?`
		args = append(args, id)
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY seq DESC`, args...)
	if err != nil {
		return nil, err
	}
	var out []Stored
	for rows.Next() {
		var p Stored
		if err := rows.Scan(&p.ID, &p.Seq, &p.Origin, &p.Actor, &p.Channel, &p.Summary, &p.BaseVersion); err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Items, err = loadItems(ctx, q, projectID, out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func loadItems(ctx context.Context, q baseline.Querier, projectID, proposalID string) ([]StoredItem, error) {
	rows, err := q.QueryContext(ctx, `SELECT i.id, i.item_index, i.op, i.entity_type, i.target_id,
    coalesce(g.key, r.key, a.key), i.target_revision, i.ref, i.payload_json, i.suggested_change_class, i.sources_json,
    i.code_refs_json, i.rationale, i.origin, i.confidence, i.review_state, i.final_payload_json, i.change_class, i.reviewed_by,
    i.reviewed_channel, i.reviewed_seq, i.review_note, i.applied_version_id
FROM proposal_items i
LEFT JOIN goals g ON i.entity_type = 'goal' AND g.id = i.target_id
LEFT JOIN requirements r ON i.entity_type = 'requirement' AND r.id = i.target_id
LEFT JOIN acceptance_criteria a ON i.entity_type = 'criterion' AND a.id = i.target_id
WHERE i.project_id = ? AND i.proposal_id = ? ORDER BY i.item_index`, projectID, proposalID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []StoredItem
	for rows.Next() {
		var it StoredItem
		var typ, payload, sources, codeRefs string
		var target, targetKey, ref, suggested, rationale, origin, confidence, final, class, by, channel, note, applied sql.NullString
		var targetRevision, reviewedSeq sql.NullInt64
		if err := rows.Scan(&it.ID, &it.Index, &it.Op, &typ, &target, &targetKey, &targetRevision, &ref, &payload, &suggested,
			&sources, &codeRefs, &rationale, &origin, &confidence, &it.State, &final, &class, &by, &channel, &reviewedSeq, &note,
			&applied); err != nil {
			return nil, err
		}
		it.Type = baseline.Type(typ)
		it.TargetID, it.TargetKey, it.Ref = target.String, targetKey.String, ref.String
		it.TargetRevision, it.ReviewedSeq = int(targetRevision.Int64), reviewedSeq.Int64
		it.SuggestedClass, it.Rationale, it.Origin, it.Confidence = suggested.String, rationale.String, origin.String, confidence.String
		it.ChangeClass, it.ReviewedBy, it.ReviewedChannel, it.ReviewNote = class.String, by.String, channel.String, note.String
		it.AppliedVersionID = applied.String
		if final.Valid {
			it.FinalPayload = json.RawMessage(final.String)
		}
		if err := json.Unmarshal([]byte(payload), &it.Payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(sources), &it.Sources); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(codeRefs), &it.CodeRefs); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// InConflict reports whether a pending item's target changed after the
// proposal's base_version (data-model §5.1): by any change, an editorial
// one at the same revision included. Reviewed items and add items are
// never in conflict.
func InConflict(ctx context.Context, q baseline.Querier, projectID string, p Stored, it StoredItem) (bool, error) {
	if it.Reviewed() || it.TargetID == "" {
		return false, nil
	}
	if it.State == "conflict" {
		return true, nil
	}
	return baseline.ChangedSince(ctx, q, projectID, it.TargetID, p.BaseVersion)
}

// AppliedRefs maps the refs of the proposal's add items that reviews
// accepted to the entities they became, with their keys.
func AppliedRefs(ctx context.Context, q baseline.Querier, projectID, proposalID string) (ids, keys map[string]string, err error) {
	rows, err := q.QueryContext(ctx, `SELECT i.ref, c.entity_id, c.entity_type FROM proposal_items i
JOIN baseline_changes c ON c.project_id = i.project_id AND c.proposal_item_id = i.id AND c.op = 'add'
WHERE i.project_id = ? AND i.proposal_id = ? AND i.op = 'add'`, projectID, proposalID)
	if err != nil {
		return nil, nil, err
	}
	type entity struct{ ref, id, typ string }
	var found []entity
	for rows.Next() {
		var e entity
		if err := rows.Scan(&e.ref, &e.id, &e.typ); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		found = append(found, e)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	ids, keys = map[string]string{}, map[string]string{}
	for _, e := range found {
		r, ok, err := baseline.Load(ctx, q, projectID, baseline.Type(e.typ), e.id)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			ids[e.ref], keys[e.ref] = e.id, r.Key
		}
	}
	return ids, keys, nil
}

// ListDoc is the proposal-list.v1 document.
type ListDoc struct {
	Schema          string        `json:"schema"`
	BaselineVersion int           `json:"baseline_version"`
	Proposals       []ListedEntry `json:"proposals"`
}

// ListedEntry is one proposal of the list.
type ListedEntry struct {
	ID          string         `json:"id"`
	Seq         int64          `json:"seq"`
	Origin      string         `json:"origin"`
	Actor       string         `json:"actor"`
	Channel     string         `json:"channel"`
	Summary     string         `json:"summary"`
	BaseVersion int            `json:"base_version"`
	Counts      map[string]int `json:"counts"`
}

// ListDocument builds proposal-list.v1: every proposal of the project,
// newest first, or with pending only those with pending or conflict items.
// An item's state counts as conflict when its target changed after the
// proposal's base_version.
func ListDocument(ctx context.Context, q baseline.Querier, projectID string, pending bool) (ListDoc, error) {
	version, err := baseline.CurrentVersion(ctx, q, projectID)
	if err != nil {
		return ListDoc{}, err
	}
	ps, err := List(ctx, q, projectID)
	if err != nil {
		return ListDoc{}, err
	}
	doc := ListDoc{Schema: "specmetry.proposal-list/v1", BaselineVersion: version, Proposals: []ListedEntry{}}
	for _, p := range ps {
		counts := map[string]int{"pending": 0, "conflict": 0, "accepted": 0, "accepted_with_edits": 0, "rejected": 0}
		for _, it := range p.Items {
			state, err := DisplayState(ctx, q, projectID, p, it)
			if err != nil {
				return ListDoc{}, err
			}
			counts[state]++
		}
		if pending && counts["pending"]+counts["conflict"] == 0 {
			continue
		}
		doc.Proposals = append(doc.Proposals, ListedEntry{ID: p.ID, Seq: p.Seq, Origin: p.Origin, Actor: p.Actor, Channel: p.Channel,
			Summary: p.Summary, BaseVersion: p.BaseVersion, Counts: counts})
	}
	return doc, nil
}

// DisplayState is an item's review state as reviewers see it: conflict for
// a pending item whose target changed after the proposal's base_version.
func DisplayState(ctx context.Context, q baseline.Querier, projectID string, p Stored, it StoredItem) (string, error) {
	if it.State != "pending" {
		return it.State, nil
	}
	conflict, err := InConflict(ctx, q, projectID, p, it)
	if err != nil || !conflict {
		return it.State, err
	}
	return "conflict", nil
}

// ShowDoc is the proposal-show.v1 document.
type ShowDoc struct {
	Schema          string         `json:"schema"`
	BaselineVersion int            `json:"baseline_version"`
	Proposal        ShowProposal   `json:"proposal"`
	Items           []ShowItem     `json:"items"`
	Questions       []ShowQuestion `json:"questions"`
}

// ShowProposal is the proposal of proposal-show.v1.
type ShowProposal struct {
	ID          string `json:"id"`
	Seq         int64  `json:"seq"`
	Origin      string `json:"origin"`
	Actor       string `json:"actor"`
	Channel     string `json:"channel"`
	Summary     string `json:"summary"`
	BaseVersion int    `json:"base_version"`
}

// ShowItem is one item of proposal-show.v1.
type ShowItem struct {
	ItemID               string          `json:"item_id"`
	ItemIndex            int             `json:"item_index"`
	Op                   string          `json:"op"`
	Type                 string          `json:"type"`
	Ref                  *string         `json:"ref"`
	Target               *string         `json:"target"`
	TargetRevision       *int            `json:"target_revision"`
	ReviewState          string          `json:"review_state"`
	Proposed             any             `json:"proposed"`
	SuggestedChangeClass *string         `json:"suggested_change_class"`
	Origin               string          `json:"origin,omitempty"`
	Confidence           string          `json:"confidence,omitempty"`
	Rationale            string          `json:"rationale,omitempty"`
	Sources              []ShowSource    `json:"sources"`
	CodeRefs             []ShowCodeRef   `json:"code_refs"`
	Conflict             *ThreeWay       `json:"conflict"`
	Review               *ShowItemReview `json:"review"`
}

// ShowSource is a cited source; excerpt is null unless stored.
type ShowSource struct {
	SourceID      string  `json:"source_id"`
	Path          string  `json:"path"`
	Locator       *string `json:"locator"`
	Verification  string  `json:"verification"`
	ExcerptSHA256 string  `json:"excerpt_sha256"`
	Excerpt       *string `json:"excerpt"`
}

// ShowCodeRef is a code reference.
type ShowCodeRef struct {
	Path   string  `json:"path"`
	Lines  *string `json:"lines"`
	Exists bool    `json:"exists"`
}

// ThreeWay is a conflict's three-way comparison.
type ThreeWay struct {
	BaseRevision    int    `json:"base_revision"`
	CurrentRevision int    `json:"current_revision"`
	CurrentStatus   string `json:"current_status"`
	Base            any    `json:"base"`
	Current         any    `json:"current"`
	Proposed        any    `json:"proposed"`
}

// ShowItemReview is an item's review.
type ShowItemReview struct {
	Decision    string  `json:"decision"`
	ChangeClass *string `json:"change_class"`
	Final       any     `json:"final"`
	By          string  `json:"by"`
	Channel     string  `json:"channel"`
	Seq         int64   `json:"seq"`
	Note        *string `json:"note"`
}

// ShowQuestion is a question the proposal opened.
type ShowQuestion struct {
	Key      string  `json:"key"`
	Status   string  `json:"status"`
	Blocking bool    `json:"blocking"`
	About    *string `json:"about"`
}

// ShowDocument builds proposal-show.v1 for a proposal of the project, or
// not_found (exit code 2). Refs to items that reviews accepted show as the
// keys they were given. A conflict shows the target's values in the
// proposal's base_version, its values now, and the proposed ones.
func ShowDocument(ctx context.Context, q baseline.Querier, projectID, id string) (ShowDoc, error) {
	p, found, err := Load(ctx, q, projectID, id)
	if err != nil {
		return ShowDoc{}, err
	}
	if !found {
		return ShowDoc{}, apperr.Newf(apperr.NotFound, "the project has no proposal %s (see proposal list)", id)
	}
	version, err := baseline.CurrentVersion(ctx, q, projectID)
	if err != nil {
		return ShowDoc{}, err
	}
	_, keys, err := AppliedRefs(ctx, q, projectID, p.ID)
	if err != nil {
		return ShowDoc{}, err
	}
	doc := ShowDoc{Schema: "specmetry.proposal-show/v1", BaselineVersion: version, Items: []ShowItem{}, Questions: []ShowQuestion{},
		Proposal: ShowProposal{ID: p.ID, Seq: p.Seq, Origin: p.Origin, Actor: p.Actor, Channel: p.Channel, Summary: p.Summary,
			BaseVersion: p.BaseVersion}}
	paths := map[string]string{}
	for _, it := range p.Items {
		si := ShowItem{ItemID: it.ID, ItemIndex: it.Index, Op: it.Op, Type: string(it.Type), Origin: it.Origin, Confidence: it.Confidence,
			Rationale: it.Rationale, Sources: []ShowSource{}, CodeRefs: []ShowCodeRef{}}
		if it.Ref != "" {
			ref := it.Ref
			si.Ref = &ref
		}
		if it.TargetID != "" {
			key, rev := it.TargetKey, it.TargetRevision
			si.Target, si.TargetRevision = &key, &rev
		}
		if it.SuggestedClass != "" {
			class := it.SuggestedClass
			si.SuggestedChangeClass = &class
		}
		if it.Payload != nil {
			si.Proposed = resolved(it.Payload, keys)
		}
		if si.ReviewState, err = DisplayState(ctx, q, projectID, p, it); err != nil {
			return ShowDoc{}, err
		}
		for _, s := range it.Sources {
			path, ok := paths[s.SourceID]
			if !ok {
				if err := q.QueryRowContext(ctx, `SELECT path FROM sources WHERE project_id = ? AND id = ?`, projectID, s.SourceID).
					Scan(&path); err != nil {
					return ShowDoc{}, err
				}
				paths[s.SourceID] = path
			}
			si.Sources = append(si.Sources, ShowSource{SourceID: s.SourceID, Path: path, Locator: s.Locator,
				Verification: s.Verification, ExcerptSHA256: s.ExcerptSHA256})
		}
		for _, c := range it.CodeRefs {
			si.CodeRefs = append(si.CodeRefs, ShowCodeRef{Path: c.Path, Lines: c.Lines, Exists: c.Exists})
		}
		if si.ReviewState == "conflict" {
			if si.Conflict, err = threeWay(ctx, q, projectID, p, it, si.Proposed); err != nil {
				return ShowDoc{}, err
			}
		}
		if it.Reviewed() {
			r := &ShowItemReview{Decision: it.State, By: it.ReviewedBy, Channel: it.ReviewedChannel, Seq: it.ReviewedSeq}
			if it.ChangeClass != "" {
				class := it.ChangeClass
				r.ChangeClass = &class
			}
			if it.ReviewNote != "" {
				note := it.ReviewNote
				r.Note = &note
			}
			if len(it.FinalPayload) > 0 {
				var final any
				if err := json.Unmarshal(it.FinalPayload, &final); err != nil {
					return ShowDoc{}, err
				}
				r.Final = final
			}
			si.Review = r
		}
		doc.Items = append(doc.Items, si)
	}
	if doc.Questions, err = showQuestions(ctx, q, projectID, p, keys); err != nil {
		return ShowDoc{}, err
	}
	return doc, nil
}

// resolved is data with refs to accepted items replaced by their keys.
func resolved(data map[string]json.RawMessage, keys map[string]string) map[string]any {
	out := map[string]any{}
	for k, raw := range data {
		var v any
		_ = json.Unmarshal(raw, &v)
		if s, ok := v.(string); ok && (k == "goal" || k == "requirement") {
			if key, ok := keys[s]; ok {
				v = key
			}
		}
		out[k] = v
	}
	return out
}

func threeWay(ctx context.Context, q baseline.Querier, projectID string, p Stored, it StoredItem, proposed any) (*ThreeWay, error) {
	current, found, err := baseline.Load(ctx, q, projectID, it.Type, it.TargetID)
	if err != nil || !found {
		return nil, err
	}
	base, found, err := baseline.SnapshotAt(ctx, q, projectID, it.TargetID, p.BaseVersion)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("no snapshot of " + current.Key + " in version " + itoa(p.BaseVersion))
	}
	revision, _ := base["revision"].(float64)
	return &ThreeWay{BaseRevision: int(revision), CurrentRevision: current.Revision, CurrentStatus: current.Status,
		Base: base["values"], Current: current.Values(), Proposed: proposed}, nil
}

func showQuestions(ctx context.Context, q baseline.Querier, projectID string, p Stored, keys map[string]string) ([]ShowQuestion, error) {
	rows, err := q.QueryContext(ctx, `SELECT qu.key, qu.status, qu.blocking, coalesce(r.key, a.key), i.ref FROM questions qu
LEFT JOIN requirements r ON qu.related_type = 'requirement' AND r.id = qu.related_id
LEFT JOIN acceptance_criteria a ON qu.related_type = 'criterion' AND a.id = qu.related_id
LEFT JOIN proposal_items i ON i.id = qu.proposal_item_id
WHERE qu.project_id = ? AND qu.proposal_id = ?`, projectID, p.ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []ShowQuestion{}
	for rows.Next() {
		var sq ShowQuestion
		var about, ref sql.NullString
		if err := rows.Scan(&sq.Key, &sq.Status, &sq.Blocking, &about, &ref); err != nil {
			return nil, err
		}
		switch {
		case about.Valid:
			sq.About = &about.String
		case ref.Valid:
			a := ref.String
			if key, ok := keys[a]; ok {
				a = key
			}
			sq.About = &a
		}
		out = append(out, sq)
	}
	slices.SortFunc(out, func(a, b ShowQuestion) int { return compareQuestionKeys(a.Key, b.Key) })
	return out, rows.Err()
}

func compareQuestionKeys(a, b string) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
