// Package fromdump is the ready checker's front-end for factgen dumps: it
// maps rows into the checker's types. It parses the JSON columns and turns
// NULL into empty values; it never selects, filters by a cutoff or
// computes. It is written independently of the engine's converter
// (factgen/toderive) and drops the current projection columns.
package fromdump

import (
	"encoding/json"
	"fmt"

	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/readycheck"
)

// Facts maps a dump.
func Facts(d factgen.Dump) readycheck.Facts {
	var f readycheck.Facts
	for _, t := range d.DeliveryTargets {
		f.Targets = append(f.Targets, readycheck.Target{ID: t.ID, Created: t.CreatedSeq, Archived: orZero(t.ArchivedSeq)})
	}
	for _, v := range d.BaselineVersions {
		f.Versions = append(f.Versions, readycheck.Version{Seq: v.Seq, Number: v.Number})
	}
	for _, g := range d.Goals {
		f.Entities = append(f.Entities, readycheck.Entity{ID: g.ID, Key: g.Key, Kind: "goal"})
	}
	for _, r := range d.Requirements {
		f.Entities = append(f.Entities, readycheck.Entity{ID: r.ID, Key: r.Key, Kind: "requirement"})
	}
	for _, c := range d.Criteria {
		f.Entities = append(f.Entities, readycheck.Entity{ID: c.ID, Key: c.Key, Kind: "criterion", Requirement: c.RequirementID})
	}
	for _, c := range d.BaselineChanges {
		f.Changes = append(f.Changes, change(c))
	}
	for _, s := range d.ScopeConfirmations {
		f.Scopes = append(f.Scopes, readycheck.Scope{ID: s.ID, Seq: s.Seq, Target: s.TargetID, Version: s.BaselineVersion, SourceSetHash: s.SourceSetHash})
	}
	for _, i := range d.ScopeItems {
		f.ScopeItems = append(f.ScopeItems, readycheck.ScopeItem{Scope: i.ConfirmationID, Requirement: i.RequirementID})
	}
	for _, s := range d.Sources {
		f.Sources = append(f.Sources, readycheck.Source{ID: s.ID, Registered: s.RegisteredSeq, Removed: orZero(s.RemovedSeq), Hash: s.AckedSHA256})
	}
	for _, a := range d.SourceAcks {
		f.Acks = append(f.Acks, readycheck.Ack{ID: a.ID, Seq: a.Seq, Source: a.SourceID, Previous: a.PreviousSHA256, Hash: a.AckedSHA256})
	}
	for _, i := range d.ProposalItems {
		f.Items = append(f.Items, readycheck.Item{ID: i.ID, State: i.ReviewState, Reviewed: orZero(i.ReviewedSeq)})
	}
	for _, q := range d.Questions {
		f.Questions = append(f.Questions, readycheck.Question{
			ID: q.ID, Key: q.Key, Created: q.CreatedSeq, Answered: orZero(q.AnsweredSeq), Blocking: q.Blocking,
			Item: orEmpty(q.ProposalItemID), About: orEmpty(q.RelatedID),
		})
	}
	for _, r := range d.CheckRuns {
		f.Runs = append(f.Runs, readycheck.Run{
			ID: r.ID, Seq: r.Seq, Criterion: r.CriterionID, Revision: r.CriterionRevision, ConfigHash: r.CheckConfigHash,
			Inputs: inputs(r.InputsJSON), HeadStart: orEmpty(r.HeadStart), HeadEnd: orEmpty(r.HeadEnd),
			TreeStart: r.TreeStateStart, TreeEnd: r.TreeStateEnd, Result: r.Result, FailureKind: orEmpty(r.FailureKind), Grade: r.EvidenceGrade,
		})
	}
	for _, x := range d.Decisions {
		f.Decisions = append(f.Decisions, readycheck.Decision{
			ID: x.ID, Seq: x.Seq, Criterion: x.CriterionID, Kind: x.Decision, Mode: orEmpty(x.Mode), Revision: x.CriterionRevision,
			ConfigHash: orEmpty(x.CheckConfigHash), Inputs: inputs(x.InputsJSON), Snapshot: orEmpty(x.Snapshot), EvidenceID: orEmpty(x.EvidenceID),
		})
	}
	return f
}

// Current maps the current-value inputs; the checker reads no setting.
func Current(c factgen.Current) readycheck.Current {
	out := readycheck.Current{S: c.S, Observations: map[string]string{}}
	for _, o := range c.Observations {
		out.Observations[o.SourceID] = o.SHA256
	}
	return out
}

func orZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func orEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// snapshotJSON is an entity snapshot of baseline-history.v1.
type snapshotJSON struct {
	Revision            int    `json:"revision"`
	Status              string `json:"status"`
	RequirementRevision int    `json:"requirement_revision"`
	Values              struct {
		Verification string `json:"verification"`
		Check        *struct {
			Command string   `json:"command"`
			Workdir string   `json:"workdir"`
			Shell   *string  `json:"shell"`
			Inputs  []string `json:"inputs"`
		} `json:"check"`
	} `json:"values"`
}

func change(c factgen.ChangeRow) readycheck.Change {
	var s snapshotJSON
	if err := json.Unmarshal([]byte(c.AfterJSON), &s); err != nil {
		panic(fmt.Sprintf("after_json of change %s: %v", c.ID, err))
	}
	out := readycheck.Change{
		ID: c.ID, Seq: c.Seq, Index: c.ChangeIndex, Entity: c.EntityID, Item: orEmpty(c.ProposalItemID),
		Status: s.Status, Revision: s.Revision, RequirementRevision: s.RequirementRevision, Verification: s.Values.Verification,
	}
	if k := s.Values.Check; k != nil {
		out.Check = &readycheck.CheckConfig{Command: k.Command, Workdir: k.Workdir, Shell: k.Shell, Inputs: k.Inputs}
	}
	return out
}

func inputs(raw *string) []readycheck.InputEntry {
	if raw == nil {
		return nil
	}
	var rows []struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		OID  string `json:"oid"`
	}
	if err := json.Unmarshal([]byte(*raw), &rows); err != nil {
		panic(fmt.Sprintf("inputs_json %q: %v", *raw, err))
	}
	out := []readycheck.InputEntry{}
	for _, r := range rows {
		out = append(out, readycheck.InputEntry{Path: r.Path, Mode: r.Mode, Type: r.Type, OID: r.OID})
	}
	return out
}
