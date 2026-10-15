package baseline

import (
	"context"
	"database/sql"
	"encoding/json"
)

// ShowDoc is the baseline-show.v1 document.
type ShowDoc struct {
	Schema          string            `json:"schema"`
	AsOfSeq         int64             `json:"as_of_seq"`
	BaselineVersion int               `json:"baseline_version"`
	Goals           []ShowGoal        `json:"goals"`
	Requirements    []ShowRequirement `json:"requirements"`
}

// ShowGoal is a goal of baseline-show.v1.
type ShowGoal struct {
	Key         string  `json:"key"`
	Revision    int     `json:"revision"`
	Status      string  `json:"status"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Origin      string  `json:"origin"`
}

// ShowRequirement is a requirement of baseline-show.v1, with its criteria.
type ShowRequirement struct {
	Key         string          `json:"key"`
	Revision    int             `json:"revision"`
	Status      string          `json:"status"`
	Goal        *string         `json:"goal"`
	Title       string          `json:"title"`
	Description *string         `json:"description"`
	Kind        string          `json:"kind"`
	Priority    string          `json:"priority"`
	Origin      string          `json:"origin"`
	Confidence  *string         `json:"confidence"`
	Criteria    []ShowCriterion `json:"criteria"`
}

// ShowCriterion is a criterion of baseline-show.v1.
type ShowCriterion struct {
	Key                 string  `json:"key"`
	Revision            int     `json:"revision"`
	Status              string  `json:"status"`
	RequirementRevision int     `json:"requirement_revision"`
	Statement           string  `json:"statement"`
	Verification        string  `json:"verification"`
	Check               *Check  `json:"check"`
	CheckConfigHash     *string `json:"check_config_hash"`
	Origin              string  `json:"origin"`
	Confidence          *string `json:"confidence"`
}

// Show builds baseline-show.v1: the confirmed baseline as of the latest
// seq, active entities only unless includeRetired. Proposals waiting for
// review are not part of it.
func Show(ctx context.Context, q Querier, projectID string, includeRetired bool) (ShowDoc, error) {
	doc := ShowDoc{Schema: "specmetry.baseline-show/v1", Goals: []ShowGoal{}, Requirements: []ShowRequirement{}}
	if err := q.QueryRowContext(ctx, `SELECT seq_counter FROM meta WHERE id = 1`).Scan(&doc.AsOfSeq); err != nil {
		return ShowDoc{}, err
	}
	var err error
	if doc.BaselineVersion, err = CurrentVersion(ctx, q, projectID); err != nil {
		return ShowDoc{}, err
	}
	shown := func(r Record) bool { return includeRetired || r.Active() }
	goals, err := LoadAll(ctx, q, projectID, Goal)
	if err != nil {
		return ShowDoc{}, err
	}
	for _, g := range goals {
		if shown(g) {
			doc.Goals = append(doc.Goals, ShowGoal{Key: g.Key, Revision: g.Revision, Status: g.Status, Title: g.Title,
				Description: g.Description, Origin: g.Origin})
		}
	}
	criteria, err := LoadAll(ctx, q, projectID, Criterion)
	if err != nil {
		return ShowDoc{}, err
	}
	byRequirement := map[string][]ShowCriterion{}
	for _, c := range criteria {
		if !shown(c) {
			continue
		}
		sc := ShowCriterion{Key: c.Key, Revision: c.Revision, Status: c.Status, RequirementRevision: c.RequirementRevision,
			Statement: c.Statement, Verification: c.Verification, Check: c.Check, Origin: c.Origin, Confidence: c.Confidence}
		if c.Check != nil {
			hash := CheckConfigHash(*c.Check)
			sc.CheckConfigHash = &hash
		}
		byRequirement[c.RequirementID] = append(byRequirement[c.RequirementID], sc)
	}
	reqs, err := LoadAll(ctx, q, projectID, Requirement)
	if err != nil {
		return ShowDoc{}, err
	}
	for _, r := range reqs {
		if !shown(r) {
			continue
		}
		sr := ShowRequirement{Key: r.Key, Revision: r.Revision, Status: r.Status, Goal: r.GoalKey, Title: r.Title,
			Description: r.Description, Kind: r.Kind, Priority: r.Priority, Origin: r.Origin, Confidence: r.Confidence,
			Criteria: byRequirement[r.ID]}
		if sr.Criteria == nil {
			sr.Criteria = []ShowCriterion{}
		}
		doc.Requirements = append(doc.Requirements, sr)
	}
	return doc, nil
}

// HistoryDoc is the baseline-history.v1 document.
type HistoryDoc struct {
	Schema          string           `json:"schema"`
	BaselineVersion int              `json:"baseline_version"`
	Versions        []HistoryVersion `json:"versions"`
}

// HistoryVersion is one version and every change it made.
type HistoryVersion struct {
	Number     int             `json:"number"`
	Seq        int64           `json:"seq"`
	Summary    *string         `json:"summary"`
	Actor      string          `json:"actor"`
	Channel    string          `json:"channel"`
	ProposalID *string         `json:"proposal_id"`
	Changes    []HistoryChange `json:"changes"`
}

// HistoryChange is one change of a version.
type HistoryChange struct {
	EntityType     string          `json:"entity_type"`
	Key            string          `json:"key"`
	Op             string          `json:"op"`
	ChangeClass    *string         `json:"change_class"`
	Before         json.RawMessage `json:"before"`
	After          json.RawMessage `json:"after"`
	ProposalItemID *string         `json:"proposal_item_id"`
}

// History builds baseline-history.v1: every version after since, oldest
// first, with its changes in the order the version made them.
func History(ctx context.Context, q Querier, projectID string, since int) (HistoryDoc, error) {
	doc := HistoryDoc{Schema: "specmetry.baseline-history/v1", Versions: []HistoryVersion{}}
	var err error
	if doc.BaselineVersion, err = CurrentVersion(ctx, q, projectID); err != nil {
		return HistoryDoc{}, err
	}
	rows, err := q.QueryContext(ctx, `SELECT v.id, v.number, v.seq, v.summary, v.actor, v.channel,
    (SELECT i.proposal_id FROM proposal_items i WHERE i.applied_version_id = v.id LIMIT 1)
FROM baseline_versions v WHERE v.project_id = ? AND v.number > ? ORDER BY v.number`, projectID, since)
	if err != nil {
		return HistoryDoc{}, err
	}
	var ids []string
	for rows.Next() {
		var id, summary string
		var proposal sql.NullString
		var v HistoryVersion
		if err := rows.Scan(&id, &v.Number, &v.Seq, &summary, &v.Actor, &v.Channel, &proposal); err != nil {
			_ = rows.Close()
			return HistoryDoc{}, err
		}
		v.Summary = &summary
		if proposal.Valid {
			v.ProposalID = &proposal.String
		}
		ids = append(ids, id)
		doc.Versions = append(doc.Versions, v)
	}
	if err := rows.Close(); err != nil {
		return HistoryDoc{}, err
	}
	for i, id := range ids {
		if doc.Versions[i].Changes, err = changes(ctx, q, projectID, id); err != nil {
			return HistoryDoc{}, err
		}
	}
	return doc, nil
}

func changes(ctx context.Context, q Querier, projectID, versionID string) ([]HistoryChange, error) {
	rows, err := q.QueryContext(ctx, `SELECT c.entity_type, coalesce(g.key, r.key, a.key), c.op, c.change_class, c.before_json,
    c.after_json, c.proposal_item_id
FROM baseline_changes c
LEFT JOIN goals g ON c.entity_type = 'goal' AND g.id = c.entity_id
LEFT JOIN requirements r ON c.entity_type = 'requirement' AND r.id = c.entity_id
LEFT JOIN acceptance_criteria a ON c.entity_type = 'criterion' AND a.id = c.entity_id
WHERE c.project_id = ? AND c.version_id = ? ORDER BY c.change_index`, projectID, versionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []HistoryChange{}
	for rows.Next() {
		var ch HistoryChange
		var class, before, item sql.NullString
		var after string
		if err := rows.Scan(&ch.EntityType, &ch.Key, &ch.Op, &class, &before, &after, &item); err != nil {
			return nil, err
		}
		ch.ChangeClass = nullString(class)
		ch.Before = json.RawMessage("null")
		if before.Valid {
			ch.Before = json.RawMessage(before.String)
		}
		ch.After = json.RawMessage(after)
		ch.ProposalItemID = nullString(item)
		out = append(out, ch)
	}
	return out, rows.Err()
}
