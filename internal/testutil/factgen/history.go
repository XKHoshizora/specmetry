package factgen

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// History writes one project's facts in the order a test performs its
// operations. Every operation is one logical write: it allocates one seq
// and the facts it writes share it, as the production writes do
// (data-model §6.13).
//
// A review or a person's edit is built as the production write builds it:
// every explicit change comes from a proposal item, a proposal names a
// target at most once (V-P03), and the changes are applied in the order of
// data-model §5.5 whatever order the test plans them in; a construction
// that breaks these rules panics. The other operations do not check their
// preconditions (decisions, claims, runs), so a test can still write a
// malformed history on purpose; the random generator performs only legal
// ones, which package legality checks independently.
type History struct {
	d       Dump
	seq     int64
	ids     int
	keys    map[string]int
	ents    map[string]*entity
	order   []string
	version int
	target  string
	sources map[string]*SourceRow
	runs    map[string]CheckRunRow
	current map[string]string // criterion ID + revision -> latest decision ID

	itemProposal map[string]string          // item ID -> its proposal
	targeted     map[string]map[string]bool // proposal -> the targets its items named
	trace        []string
}

type entity struct {
	typ, id, key string
	origin       string
	reqID        string // criteria
	reqNumber    int    // requirements: the number in the key
	status       string
	revision     int
	revisionSeq  int64
	title        string
	goalID       string
	priority     string
	statement    string
	verification string
	check        *baseline.Check
	reqRevision  int
}

// New starts the history of a project.
func New(projectID string) *History {
	return &History{
		d:       Dump{ProjectID: projectID},
		keys:    map[string]int{},
		ents:    map[string]*entity{},
		sources: map[string]*SourceRow{},
		runs:    map[string]CheckRunRow{},
		current: map[string]string{},

		itemProposal: map[string]string{},
		targeted:     map[string]map[string]bool{},
	}
}

// Seq is the latest allocated seq.
func (h *History) Seq() int64 { return h.seq }

// Trace is the operations performed so far, one line per write, to locate
// a failure in a generated history.
func (h *History) Trace() []string { return slices.Clone(h.trace) }

func (h *History) log(format string, args ...any) {
	h.trace = append(h.trace, fmt.Sprintf("seq %d: ", h.seq)+fmt.Sprintf(format, args...))
}

// short is a commit or hash prefix for the trace.
func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	if s == "" {
		return "NULL"
	}
	return s
}

func (h *History) next() int64 {
	h.seq++
	return h.seq
}

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// id returns a new ULID-shaped identifier. IDs identify; they never order.
func (h *History) id() string {
	h.ids++
	b := []byte("01JAAAAAAA0000000000000000")
	n := h.ids
	for i := len(b) - 1; n > 0; i-- {
		b[i] = crockford[n%32]
		n /= 32
	}
	return string(b)
}

// Commit is the SHA of a commit named in a test.
func Commit(name string) string {
	sum := sha1.Sum([]byte(name))
	return hex.EncodeToString(sum[:])
}

// Hash is a sha256 named in a test, as a source's content hash.
func Hash(name string) string {
	sum := sha256.Sum256([]byte(name))
	return hex.EncodeToString(sum[:])
}

// Key is an entity's key (assigned when the write that adds it applies).
func (h *History) Key(id string) string { return h.ents[id].key }

// Revision is an entity's current revision.
func (h *History) Revision(id string) int { return h.ents[id].revision }

// Active reports whether an entity exists and is not retired.
func (h *History) Active(id string) bool {
	e, ok := h.ents[id]
	return ok && e.status == "active"
}

// Entities are the IDs of the entities of a type, in creation order.
func (h *History) Entities(typ string) []string {
	var out []string
	for _, id := range h.order {
		if h.ents[id].typ == typ {
			out = append(out, id)
		}
	}
	return out
}

// Parent is a criterion's requirement.
func (h *History) Parent(criterion string) string { return h.ents[criterion].reqID }

// ParentRevised reports whether a criterion's requirement was revised after
// the criterion was confirmed for it.
func (h *History) ParentRevised(criterion string) bool {
	e := h.ents[criterion]
	return e.reqRevision != h.ents[e.reqID].revision
}

// Verification is a criterion's verification.
func (h *History) Verification(criterion string) string { return h.ents[criterion].verification }

// HasCheck reports whether a criterion has a check.
func (h *History) HasCheck(criterion string) bool { return h.ents[criterion].check != nil }

// ConfigHash is a criterion's current check configuration hash, or "".
func (h *History) ConfigHash(criterion string) string {
	if c := h.ents[criterion].check; c != nil {
		return baseline.CheckConfigHash(*c)
	}
	return ""
}

// Inputs are a criterion's declared inputs.
func (h *History) Inputs(criterion string) []string {
	if c := h.ents[criterion].check; c != nil {
		return c.Inputs
	}
	return nil
}

// Version is the current baseline version number.
func (h *History) Version() int { return h.version }

// ActiveTarget is the active delivery target's ID, or "".
func (h *History) ActiveTarget() string { return h.target }

// CurrentDecision is the latest decision on a criterion's current revision.
func (h *History) CurrentDecision(criterion string) string {
	return h.current[decisionKey(criterion, h.ents[criterion].revision)]
}

// RunRow is a recorded run.
func (h *History) RunRow(id string) CheckRunRow { return h.runs[id] }

// Runs are a criterion's recorded runs, in seq order.
func (h *History) Runs(criterion string) []CheckRunRow {
	var out []CheckRunRow
	for _, r := range h.d.CheckRuns {
		if r.CriterionID == criterion {
			out = append(out, r)
		}
	}
	return out
}

// SourceIDs are the sources not removed, in registration order.
func (h *History) SourceIDs() []string {
	var out []string
	for _, s := range h.d.Sources {
		if h.sources[s.ID].RemovedSeq == nil {
			out = append(out, s.ID)
		}
	}
	return out
}

// AckedHash is a source's latest acknowledged hash.
func (h *History) AckedHash(id string) string { return h.sources[id].AckedSHA256 }

// Questions are the IDs of the open questions.
func (h *History) Questions() []string {
	var out []string
	for _, q := range h.d.Questions {
		if q.AnsweredSeq == nil {
			out = append(out, q.ID)
		}
	}
	return out
}

func decisionKey(criterion string, revision int) string {
	return fmt.Sprintf("%s@%d", criterion, revision)
}

func ptr[T any](v T) *T { return &v }

// setHash is the source set hash of the sources not removed (data-model
// §4.3), with the production function.
func (h *History) setHash() string {
	var members []source.Member
	for _, s := range h.sources {
		if s.RemovedSeq == nil {
			members = append(members, source.Member{ID: s.ID, AckedSHA256: s.AckedSHA256})
		}
	}
	return source.SetHash(members)
}

// snapshot is the entity's after_json, in the production format
// (baseline.Record.Snapshot).
func (h *History) snapshot(e *entity) string {
	rec := baseline.Record{
		Type: baseline.Type(e.typ), Key: e.key, Status: e.status, Revision: e.revision,
		Title: e.title, Priority: e.priority, Kind: "functional",
		Statement: e.statement, Verification: e.verification, Check: e.check,
		RequirementRevision: e.reqRevision,
	}
	if e.goalID != "" {
		rec.GoalKey = ptr(h.ents[e.goalID].key)
	}
	if e.reqID != "" {
		rec.RequirementKey = h.ents[e.reqID].key
	}
	raw, err := json.Marshal(rec.Snapshot())
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// Dump returns the rows written so far, with the current projection
// columns.
func (h *History) Dump() Dump {
	d := h.d
	d.LatestSeq = h.seq
	d.Goals, d.Requirements, d.Criteria = nil, nil, nil
	for _, id := range h.order {
		e := h.ents[id]
		row := EntityRow{
			ID: e.id, Key: e.key, Origin: e.origin, RequirementID: e.reqID,
			Status: e.status, Revision: e.revision, RevisionSeq: e.revisionSeq, RequirementRevision: e.reqRevision,
		}
		if e.check != nil {
			row.CheckConfigHash = ptr(baseline.CheckConfigHash(*e.check))
		}
		switch e.typ {
		case "goal":
			d.Goals = append(d.Goals, row)
		case "requirement":
			d.Requirements = append(d.Requirements, row)
		default:
			d.Criteria = append(d.Criteria, row)
		}
	}
	d.Sources = nil
	for _, s := range h.d.Sources {
		d.Sources = append(d.Sources, *h.sources[s.ID])
	}
	// Copy every slice, so a test that edits a dump changes no other one.
	d.DeliveryTargets = slices.Clone(d.DeliveryTargets)
	d.BaselineVersions = slices.Clone(d.BaselineVersions)
	d.BaselineChanges = slices.Clone(d.BaselineChanges)
	d.ScopeConfirmations = slices.Clone(d.ScopeConfirmations)
	d.ScopeItems = slices.Clone(d.ScopeItems)
	d.SourceAcks = slices.Clone(d.SourceAcks)
	d.SourceRefs = slices.Clone(d.SourceRefs)
	d.Proposals = slices.Clone(d.Proposals)
	d.ProposalItems = slices.Clone(d.ProposalItems)
	d.Questions = slices.Clone(d.Questions)
	d.ClaimBatches = slices.Clone(d.ClaimBatches)
	d.Claims = slices.Clone(d.Claims)
	d.CheckRuns = slices.Clone(d.CheckRuns)
	d.Commits = slices.Clone(d.Commits)
	d.CommitLinks = slices.Clone(d.CommitLinks)
	d.Decisions = slices.Clone(d.Decisions)
	return d
}

// SetTarget sets the delivery target: the active one is archived and the
// new one created in one write (architecture §6.1.7).
func (h *History) SetTarget(name, ref string) string {
	seq := h.next()
	for i := range h.d.DeliveryTargets {
		if h.d.DeliveryTargets[i].ID == h.target {
			h.d.DeliveryTargets[i].ArchivedSeq = ptr(seq)
		}
	}
	id := h.id()
	h.d.DeliveryTargets = append(h.d.DeliveryTargets, TargetRow{ID: id, Name: name, Ref: ref, CreatedSeq: seq})
	h.target = id
	h.log("target set %s %s", name, ref)
	return id
}

// AddSource registers a source with its content hash.
func (h *History) AddSource(path, hash string) string {
	seq := h.next()
	id := h.id()
	row := SourceRow{ID: id, Path: path, Kind: "product_doc", RegisteredSeq: seq, AckedSHA256: hash}
	h.d.Sources = append(h.d.Sources, row)
	h.sources[id] = &row
	h.log("source add %s %s", path, short(hash))
	return id
}

// AckSource records a person's acknowledgement of a source's new hash.
func (h *History) AckSource(id, hash string) {
	seq := h.next()
	s := h.sources[id]
	h.d.SourceAcks = append(h.d.SourceAcks, SourceAckRow{ID: h.id(), Seq: seq, SourceID: id, PreviousSHA256: s.AckedSHA256, AckedSHA256: hash})
	s.AckedSHA256 = hash
	h.log("source ack %s %s", s.Path, short(hash))
}

// RemoveSource removes a source; its references stay.
func (h *History) RemoveSource(id string) {
	h.sources[id].RemovedSeq = ptr(h.next())
	h.log("source remove %s", h.sources[id].Path)
}

// ConfirmScope confirms the active target's scope at the current baseline
// version and source set hash.
func (h *History) ConfirmScope(requirements ...string) string {
	id := h.confirm(h.next(), requirements)
	h.log("scope confirm %s", h.keysOf(requirements))
	return id
}

func (h *History) keysOf(ids []string) []string {
	var out []string
	for _, id := range ids {
		out = append(out, h.ents[id].key)
	}
	return out
}

func (h *History) confirm(seq int64, requirements []string) string {
	id := h.id()
	h.d.ScopeConfirmations = append(h.d.ScopeConfirmations, ScopeRow{
		ID: id, Seq: seq, TargetID: h.target, BaselineVersion: h.version, SourceSetHash: h.setHash(),
	})
	for _, r := range requirements {
		h.d.ScopeItems = append(h.d.ScopeItems, ScopeItemRow{ConfirmationID: id, RequirementID: r, RequirementRevision: h.ents[r].revision})
	}
	return id
}

// QuestionSpec describes a question raised in a proposal: about one of its
// items (AboutItem, from 1), about an existing entity, or neither.
type QuestionSpec struct {
	Blocking    bool
	AboutItem   int
	AboutEntity string
}

// Propose submits a proposal with pending items and questions, all in one
// write (architecture §6.1.4). It returns the item and question IDs.
func (h *History) Propose(items int, questions ...QuestionSpec) (itemIDs, questionIDs []string) {
	seq := h.next()
	pid := h.id()
	h.d.Proposals = append(h.d.Proposals, ProposalRow{ID: pid, Seq: seq})
	for i := range items {
		id := h.id()
		h.d.ProposalItems = append(h.d.ProposalItems, ProposalItemRow{ID: id, ProposalID: pid, ItemIndex: i, ReviewState: "pending"})
		h.itemProposal[id] = pid
		itemIDs = append(itemIDs, id)
	}
	for _, q := range questions {
		h.keys["question"]++
		row := QuestionRow{ID: h.id(), Key: fmt.Sprintf("Q-%03d", h.keys["question"]), CreatedSeq: seq, Status: "open", Blocking: q.Blocking}
		switch {
		case q.AboutItem > 0:
			row.ProposalItemID = ptr(itemIDs[q.AboutItem-1])
		case q.AboutEntity != "":
			row.RelatedType, row.RelatedID = ptr(h.ents[q.AboutEntity].typ), ptr(q.AboutEntity)
		}
		h.d.Questions = append(h.d.Questions, row)
		questionIDs = append(questionIDs, row.ID)
	}
	h.log("propose %s: %d items, %d questions", pid, items, len(questions))
	return itemIDs, questionIDs
}

// Answer answers (or dismisses) an open question.
func (h *History) Answer(question string) {
	seq := h.next()
	for i := range h.d.Questions {
		if h.d.Questions[i].ID == question {
			h.d.Questions[i].AnsweredSeq, h.d.Questions[i].Status = ptr(seq), "answered"
			h.log("question answer %s", h.d.Questions[i].Key)
		}
	}
}

// ClaimSpec is one claim of a batch. Criterion "" is unmapped work;
// Revision 0 is the criterion's current revision.
type ClaimSpec struct {
	Criterion string
	Assertion string
	Revision  int
}

// Claims submits a batch: the batch and its claims share one seq
// (data-model §4.10).
func (h *History) Claims(kind string, claims ...ClaimSpec) int64 {
	seq := h.next()
	batch := h.id()
	row := ClaimBatchRow{ID: batch, Seq: seq, Kind: kind}
	if kind == "audit" {
		row.AuditScopeJSON = ptr(`{"all":true}`)
	}
	h.d.ClaimBatches = append(h.d.ClaimBatches, row)
	for _, c := range claims {
		claim := ClaimRow{ID: h.id(), Seq: seq, BatchID: batch, TargetType: "none", Assertion: c.Assertion}
		if c.Criterion != "" {
			rev := c.Revision
			if rev == 0 {
				rev = h.ents[c.Criterion].revision
			}
			claim.TargetType, claim.CriterionID, claim.CriterionRevision = "criterion", ptr(c.Criterion), ptr(rev)
		}
		h.d.Claims = append(h.d.Claims, claim)
	}
	h.log("claims %s %v", kind, claims)
	return seq
}

// RunSpec describes a check run. Zero values mean: HeadEnd = Head, clean
// at both ends, pass, observed, the main worktree, the criterion's current
// revision and configuration, no input record.
type RunSpec struct {
	Head, HeadEnd      string
	TreeStart, TreeEnd string
	Result             string
	FailureKind        string
	Grade              string
	Worktree           string
	Inputs             []InputEntry
	Revision           int
	ConfigHash         string
}

// InputEntry is one entry of an input record (inputs_json).
type InputEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	OID  string `json:"oid"`
}

// MainWorktree is the default worktree of a run.
const MainWorktree = "01JAAAAAAAWRKTREE000000000"

// Run records a check run; each run is one write (data-model §4.11).
func (h *History) Run(criterion string, s RunSpec) string {
	seq := h.next()
	e := h.ents[criterion]
	if s.HeadEnd == "" {
		s.HeadEnd = s.Head
	}
	s.TreeStart, s.TreeEnd = or(s.TreeStart, "clean"), or(s.TreeEnd, "clean")
	s.Result, s.Grade, s.Worktree = or(s.Result, "pass"), or(s.Grade, "observed"), or(s.Worktree, MainWorktree)
	if s.Revision == 0 {
		s.Revision = e.revision
	}
	if s.ConfigHash == "" {
		s.ConfigHash = h.ConfigHash(criterion)
	}
	row := CheckRunRow{
		ID: h.id(), Seq: seq, CriterionID: criterion, CriterionRevision: s.Revision,
		RequirementRevision: e.reqRevision, CheckConfigHash: s.ConfigHash, WorktreeID: s.Worktree,
		EvidenceGrade: s.Grade, TreeStateStart: s.TreeStart, TreeStateEnd: s.TreeEnd, Result: s.Result,
	}
	if s.Head != "" {
		row.HeadStart = ptr(s.Head)
	}
	if s.HeadEnd != "" {
		row.HeadEnd = ptr(s.HeadEnd)
	}
	if s.FailureKind != "" {
		row.FailureKind = ptr(s.FailureKind)
	}
	if s.Inputs != nil {
		row.InputsJSON = ptr(inputsJSON(s.Inputs))
	}
	h.d.CheckRuns = append(h.d.CheckRuns, row)
	h.runs[row.ID] = row
	h.log("run %s %s at %s..%s %s/%s %s/%s %s", row.ID, e.key, short(s.Head), short(s.HeadEnd), s.TreeStart, s.TreeEnd, s.Result, s.FailureKind, s.Grade)
	return row.ID
}

func inputsJSON(in []InputEntry) string {
	sorted := slices.Clone(in)
	slices.SortFunc(sorted, func(a, b InputEntry) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
	raw, err := json.Marshal(sorted)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func or(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// DecisionSpec describes a decision. Evidence names the run relied on; its
// grade, configuration and inputs are copied from it unless given.
// Revision 0 is the criterion's current revision.
type DecisionSpec struct {
	Decision   string
	Mode       string
	Snapshot   string
	Evidence   string
	Grade      string
	ConfigHash string
	Inputs     []InputEntry
	Rationale  string
	Revision   int
}

// Decide records a person's decision on a criterion (data-model §4.13),
// with seen_decision_id the current decision.
func (h *History) Decide(criterion string, s DecisionSpec) string {
	seq := h.next()
	e := h.ents[criterion]
	if s.Revision == 0 {
		s.Revision = e.revision
	}
	row := DecisionRow{
		ID: h.id(), Seq: seq, CriterionID: criterion, Decision: s.Decision,
		CriterionRevision: s.Revision, RequirementRevision: h.ents[e.reqID].revision, TargetID: h.target,
	}
	if s.Mode != "" {
		row.Mode = ptr(s.Mode)
	}
	if s.Snapshot != "" {
		row.Snapshot = ptr(s.Snapshot)
	}
	if s.Evidence != "" {
		run := h.runs[s.Evidence]
		row.EvidenceID = ptr(s.Evidence)
		s.Grade, s.ConfigHash = or(s.Grade, run.EvidenceGrade), or(s.ConfigHash, run.CheckConfigHash)
		if s.Inputs == nil && run.InputsJSON != nil {
			row.InputsJSON = ptr(*run.InputsJSON)
		}
	}
	if s.ConfigHash == "" {
		s.ConfigHash = h.ConfigHash(criterion)
	}
	if s.ConfigHash != "" {
		row.CheckConfigHash = ptr(s.ConfigHash)
	}
	if s.Grade != "" {
		row.EvidenceGrade = ptr(s.Grade)
	}
	if s.Inputs != nil {
		row.InputsJSON = ptr(inputsJSON(s.Inputs))
	}
	if s.Rationale != "" || s.Decision != "accepted" || s.Mode == "exception" {
		row.Rationale = ptr(or(s.Rationale, "a reason"))
	}
	k := decisionKey(criterion, s.Revision)
	if cur := h.current[k]; cur != "" {
		row.SeenDecisionID = ptr(cur)
	}
	h.current[k] = row.ID
	h.d.Decisions = append(h.d.Decisions, row)
	h.log("decision %s %s %s %s on %s evidence %s", row.ID, e.key, s.Decision, s.Mode, short(s.Snapshot), s.Evidence)
	return row.ID
}

// CommitSpec is a commit found by a scan and the entities it references.
type CommitSpec struct {
	SHA   string
	Links []string
}

// Scan records the commits one git scan finds: they share one seq
// (data-model §4.12).
func (h *History) Scan(commits ...CommitSpec) int64 {
	seq := h.next()
	for _, c := range commits {
		id := h.id()
		h.d.Commits = append(h.d.Commits, CommitRow{ID: id, Seq: seq, SHA: c.SHA})
		for _, l := range c.Links {
			h.d.CommitLinks = append(h.d.CommitLinks, CommitLinkRow{CommitID: id, TargetType: h.ents[l].typ, TargetID: l, LinkSource: "trailer"})
		}
		h.log("scan %s %v", short(c.SHA), h.keysOf(c.Links))
	}
	return seq
}

// Current is the current-value inputs with every source in the current
// source set observed at its acknowledged hash.
func (h *History) Current(s, stage string) Current {
	c := Current{S: s, Stage: stage}
	for _, id := range h.SourceIDs() {
		c.Observations = append(c.Observations, Observation{SourceID: id, SHA256: h.sources[id].AckedSHA256})
	}
	return c
}
