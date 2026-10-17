// Package toderive converts a factgen dump to the derivation engine's input.
// It only maps rows: it parses the JSON columns and turns NULL into the
// engine's zero values, and it never selects, filters or computes. The
// current projection columns of the dump are not carried over: the engine
// reconstructs the past from the facts (data-model §6.13).
package toderive

import (
	"encoding/json"
	"fmt"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
)

// Input converts a dump, the current-value inputs and a cutoff.
func Input(d factgen.Dump, c factgen.Current, asOf int64) derive.Input {
	return derive.Input{Facts: Facts(d), Current: Current(c), AsOfSeq: asOf}
}

// Current converts the current-value inputs; every one of them is present.
func Current(c factgen.Current) derive.Current {
	out := derive.Current{
		Snapshot: &derive.Snapshot{Commit: c.S},
		Settings: &derive.Settings{Stage: derive.Stage(c.Stage), SourcesSkipped: c.SourcesSkipped, InventorySkipped: c.InventorySkipped},
	}
	for _, o := range c.Observations {
		out.Observations = append(out.Observations, derive.Observation{SourceID: o.SourceID, SHA256: o.SHA256})
	}
	if c.Workspace != nil {
		out.Workspace = &derive.Workspace{WorktreeID: c.Workspace.WorktreeID, Head: c.Workspace.Head, TreeState: c.Workspace.TreeState}
	}
	return out
}

// Facts converts the rows.
func Facts(d factgen.Dump) derive.Facts {
	f := derive.Facts{ProjectID: d.ProjectID, LatestSeq: d.LatestSeq}
	for _, t := range d.DeliveryTargets {
		f.Targets = append(f.Targets, derive.Target{ID: t.ID, Name: t.Name, Ref: t.Ref, CreatedSeq: t.CreatedSeq, ArchivedSeq: seq(t.ArchivedSeq)})
	}
	for _, v := range d.BaselineVersions {
		f.BaselineVersions = append(f.BaselineVersions, derive.BaselineVersion{ID: v.ID, Seq: v.Seq, Number: v.Number})
	}
	for _, c := range d.BaselineChanges {
		f.BaselineChanges = append(f.BaselineChanges, derive.BaselineChange{
			ID: c.ID, Seq: c.Seq, VersionID: c.VersionID, ChangeIndex: c.ChangeIndex,
			EntityType: derive.EntityType(c.EntityType), EntityID: c.EntityID,
			ProposalItemID: str(c.ProposalItemID), After: snapshot(c.AfterJSON),
		})
	}
	for typ, rows := range map[derive.EntityType][]factgen.EntityRow{
		derive.GoalEntity: d.Goals, derive.RequirementEntity: d.Requirements, derive.CriterionEntity: d.Criteria,
	} {
		for _, e := range rows {
			f.Entities = append(f.Entities, derive.Entity{ID: e.ID, Type: typ, Key: e.Key, Origin: e.Origin, RequirementID: e.RequirementID})
		}
	}
	for _, s := range d.ScopeConfirmations {
		f.ScopeConfirmations = append(f.ScopeConfirmations, derive.ScopeConfirmation{
			ID: s.ID, Seq: s.Seq, TargetID: s.TargetID, BaselineVersion: s.BaselineVersion, SourceSetHash: s.SourceSetHash,
		})
	}
	for _, i := range d.ScopeItems {
		f.ScopeItems = append(f.ScopeItems, derive.ScopeItem{ConfirmationID: i.ConfirmationID, RequirementID: i.RequirementID, RequirementRevision: i.RequirementRevision})
	}
	for _, s := range d.Sources {
		f.Sources = append(f.Sources, derive.Source{
			ID: s.ID, Path: s.Path, Kind: s.Kind, RegisteredSeq: s.RegisteredSeq, RemovedSeq: seq(s.RemovedSeq), AckedSHA256: s.AckedSHA256,
		})
	}
	for _, a := range d.SourceAcks {
		f.SourceAcks = append(f.SourceAcks, derive.SourceAck{ID: a.ID, Seq: a.Seq, SourceID: a.SourceID, PreviousSHA256: a.PreviousSHA256, AckedSHA256: a.AckedSHA256})
	}
	for _, r := range d.SourceRefs {
		f.SourceRefs = append(f.SourceRefs, derive.SourceRef{
			ID: r.ID, Seq: r.Seq, SourceID: r.SourceID, EntityType: derive.EntityType(r.EntityType), EntityID: r.EntityID,
			EntityRevision: r.EntityRevision, Verification: r.Verification,
		})
	}
	for _, p := range d.Proposals {
		f.Proposals = append(f.Proposals, derive.Proposal{ID: p.ID, Seq: p.Seq})
	}
	for _, i := range d.ProposalItems {
		f.ProposalItems = append(f.ProposalItems, derive.ProposalItem{
			ID: i.ID, ProposalID: i.ProposalID, ItemIndex: i.ItemIndex, ReviewState: i.ReviewState, ReviewedSeq: seq(i.ReviewedSeq),
		})
	}
	for _, q := range d.Questions {
		f.Questions = append(f.Questions, derive.Question{
			ID: q.ID, Key: q.Key, CreatedSeq: q.CreatedSeq, AnsweredSeq: seq(q.AnsweredSeq), Blocking: q.Blocking,
			RelatedType: derive.EntityType(str(q.RelatedType)), RelatedID: str(q.RelatedID), ProposalItemID: str(q.ProposalItemID),
		})
	}
	for _, b := range d.ClaimBatches {
		f.ClaimBatches = append(f.ClaimBatches, derive.ClaimBatch{ID: b.ID, Seq: b.Seq, Kind: b.Kind})
	}
	for _, c := range d.Claims {
		claim := derive.Claim{ID: c.ID, Seq: c.Seq, BatchID: c.BatchID, TargetType: c.TargetType, CriterionID: str(c.CriterionID), Assertion: c.Assertion}
		if c.CriterionRevision != nil {
			claim.CriterionRevision = *c.CriterionRevision
		}
		f.Claims = append(f.Claims, claim)
	}
	for _, r := range d.CheckRuns {
		f.CheckRuns = append(f.CheckRuns, derive.CheckRun{
			ID: r.ID, Seq: r.Seq, CriterionID: r.CriterionID, CriterionRevision: r.CriterionRevision,
			RequirementRevision: r.RequirementRevision, CheckConfigHash: r.CheckConfigHash, Inputs: inputs(r.InputsJSON),
			WorktreeID: r.WorktreeID, EvidenceGrade: derive.Grade(r.EvidenceGrade), HeadStart: str(r.HeadStart), HeadEnd: str(r.HeadEnd),
			TreeStateStart: r.TreeStateStart, TreeStateEnd: r.TreeStateEnd, Result: r.Result, FailureKind: str(r.FailureKind),
		})
	}
	for _, c := range d.Commits {
		f.Commits = append(f.Commits, derive.Commit{ID: c.ID, Seq: c.Seq, SHA: c.SHA})
	}
	for _, l := range d.CommitLinks {
		f.CommitLinks = append(f.CommitLinks, derive.CommitLink{CommitID: l.CommitID, TargetType: derive.EntityType(l.TargetType), TargetID: l.TargetID})
	}
	for _, x := range d.Decisions {
		f.Decisions = append(f.Decisions, derive.Decision{
			ID: x.ID, Seq: x.Seq, CriterionID: x.CriterionID, Decision: x.Decision, Mode: str(x.Mode),
			CriterionRevision: x.CriterionRevision, RequirementRevision: x.RequirementRevision,
			CheckConfigHash: str(x.CheckConfigHash), Inputs: inputs(x.InputsJSON), TargetID: x.TargetID,
			Snapshot: str(x.Snapshot), EvidenceID: str(x.EvidenceID), EvidenceGrade: derive.Grade(str(x.EvidenceGrade)),
			Rationale: str(x.Rationale),
		})
	}
	return f
}

func seq(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func str(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// afterJSON is the entity snapshot of baseline-history.v1.
type afterJSON struct {
	Revision            int    `json:"revision"`
	Status              string `json:"status"`
	RequirementRevision int    `json:"requirement_revision"`
	Values              struct {
		Title        string  `json:"title"`
		Goal         *string `json:"goal"`
		Priority     string  `json:"priority"`
		Verification string  `json:"verification"`
		Check        *struct {
			Command string   `json:"command"`
			Workdir string   `json:"workdir"`
			Shell   *string  `json:"shell"`
			Inputs  []string `json:"inputs"`
		} `json:"check"`
	} `json:"values"`
}

func snapshot(raw string) derive.EntityState {
	var a afterJSON
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		panic(fmt.Sprintf("after_json %q: %v", raw, err))
	}
	s := derive.EntityState{
		Revision: a.Revision, Status: a.Status, RequirementRevision: a.RequirementRevision,
		Title: a.Values.Title, GoalKey: str(a.Values.Goal), Priority: a.Values.Priority, Verification: a.Values.Verification,
	}
	if c := a.Values.Check; c != nil {
		s.Check = &derive.Check{Command: c.Command, Workdir: c.Workdir, Shell: c.Shell, Inputs: c.Inputs}
	}
	return s
}

func inputs(raw *string) derive.InputRecord {
	if raw == nil {
		return nil
	}
	var entries []factgen.InputEntry
	if err := json.Unmarshal([]byte(*raw), &entries); err != nil {
		panic(fmt.Sprintf("inputs_json %q: %v", *raw, err))
	}
	out := derive.InputRecord{}
	for _, e := range entries {
		out = append(out, derive.InputEntry{Path: e.Path, Mode: e.Mode, Type: e.Type, OID: e.OID})
	}
	return out
}
