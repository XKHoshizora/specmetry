// Package factgen builds fact histories for derivation tests: a dump of
// database rows (JSON columns as text, NULL as nil) and the current-value
// inputs of one command. It is the test front-end of WP2.1 and WP2.2: it
// writes facts the way the production writes do, and it never selects a
// current record, computes an applicability or filters by a cutoff. Each
// implementation under test converts the dump on its own.
//
// A dump also carries the current projection columns (an entity's current
// status, a source's latest acknowledged hash, a question's backfilled
// link): an implementation that reconstructs the past must not use them.
package factgen

// Dump is the rows of one project.
type Dump struct {
	ProjectID string
	LatestSeq int64

	DeliveryTargets    []TargetRow
	BaselineVersions   []VersionRow
	BaselineChanges    []ChangeRow
	Goals              []EntityRow
	Requirements       []EntityRow
	Criteria           []EntityRow
	ScopeConfirmations []ScopeRow
	ScopeItems         []ScopeItemRow
	Sources            []SourceRow
	SourceAcks         []SourceAckRow
	SourceRefs         []SourceRefRow
	Proposals          []ProposalRow
	ProposalItems      []ProposalItemRow
	Questions          []QuestionRow
	ClaimBatches       []ClaimBatchRow
	Claims             []ClaimRow
	CheckRuns          []CheckRunRow
	Commits            []CommitRow
	CommitLinks        []CommitLinkRow
	Decisions          []DecisionRow
}

// TargetRow is a delivery_targets row.
type TargetRow struct {
	ID, Name, Ref string
	CreatedSeq    int64
	ArchivedSeq   *int64
}

// VersionRow is a baseline_versions row.
type VersionRow struct {
	ID     string
	Seq    int64
	Number int
}

// ChangeRow is a baseline_changes row; AfterJSON is the entity snapshot of
// baseline-history.v1.
type ChangeRow struct {
	ID             string
	Seq            int64
	VersionID      string
	ChangeIndex    int
	EntityType     string
	EntityID       string
	Op             string
	ChangeClass    *string
	BeforeJSON     *string
	AfterJSON      string
	ProposalItemID *string
}

// EntityRow is a goals, requirements or acceptance_criteria row. The
// mutable columns hold the current projection.
type EntityRow struct {
	ID, Key       string
	Origin        string
	RequirementID string // criteria only

	// Current projection.
	Status              string
	Revision            int
	RevisionSeq         int64
	RequirementRevision int // criteria only
	CheckConfigHash     *string
}

// ScopeRow is a scope_confirmations row.
type ScopeRow struct {
	ID              string
	Seq             int64
	TargetID        string
	BaselineVersion int
	SourceSetHash   string
}

// ScopeItemRow is a scope_items row.
type ScopeItemRow struct {
	ConfirmationID      string
	RequirementID       string
	RequirementRevision int
}

// SourceRow is a sources row; AckedSHA256 and RemovedSeq are current.
type SourceRow struct {
	ID, Path, Kind string
	RegisteredSeq  int64
	RemovedSeq     *int64
	AckedSHA256    string
}

// SourceAckRow is a source_acks row.
type SourceAckRow struct {
	ID             string
	Seq            int64
	SourceID       string
	PreviousSHA256 string
	AckedSHA256    string
}

// SourceRefRow is a source_refs row.
type SourceRefRow struct {
	ID             string
	Seq            int64
	SourceID       string
	EntityType     string
	EntityID       string
	EntityRevision int
	Verification   string
}

// ProposalRow is a proposals row.
type ProposalRow struct {
	ID  string
	Seq int64
}

// ProposalItemRow is a proposal_items row; the review columns are current.
type ProposalItemRow struct {
	ID          string
	ProposalID  string
	ItemIndex   int
	ReviewState string
	ReviewedSeq *int64
}

// QuestionRow is a questions row; the status and the link are current (the
// link is backfilled when the item it is about is accepted).
type QuestionRow struct {
	ID, Key        string
	CreatedSeq     int64
	AnsweredSeq    *int64
	Status         string
	Blocking       bool
	RelatedType    *string
	RelatedID      *string
	ProposalItemID *string
}

// ClaimBatchRow is a claim_batches row.
type ClaimBatchRow struct {
	ID             string
	Seq            int64
	Kind           string
	AuditScopeJSON *string
}

// ClaimRow is a claims row.
type ClaimRow struct {
	ID                string
	Seq               int64
	BatchID           string
	TargetType        string
	CriterionID       *string
	CriterionRevision *int
	Assertion         string
}

// CheckRunRow is a check_runs row.
type CheckRunRow struct {
	ID                  string
	Seq                 int64
	CriterionID         string
	CriterionRevision   int
	RequirementRevision int
	CheckConfigHash     string
	InputsJSON          *string
	WorktreeID          string
	EvidenceGrade       string
	HeadStart, HeadEnd  *string
	TreeStateStart      string
	TreeStateEnd        string
	Result              string
	FailureKind         *string
}

// CommitRow is a commits row.
type CommitRow struct {
	ID  string
	Seq int64
	SHA string
}

// CommitLinkRow is a commit_links row.
type CommitLinkRow struct {
	CommitID   string
	TargetType string
	TargetID   string
	LinkSource string
}

// DecisionRow is a decisions row.
type DecisionRow struct {
	ID                  string
	Seq                 int64
	CriterionID         string
	Decision            string
	Mode                *string
	CriterionRevision   int
	RequirementRevision int
	CheckConfigHash     *string
	InputsJSON          *string
	TargetID            string
	Snapshot            *string
	EvidenceID          *string
	EvidenceGrade       *string
	SeenDecisionID      *string
	Rationale           *string
}

// Current holds the current-value inputs of one command, as an output
// records them (data-model §6.13.1).
type Current struct {
	// S is the commit the target's ref resolves to; "" is NULL.
	S string
	// Observations are the source observations; a SHA256 of "" is a file
	// observed missing.
	Observations []Observation
	Stage        string
	// SourcesSkipped and InventorySkipped are the settings_json flags.
	SourcesSkipped   bool
	InventorySkipped bool
	Workspace        *Workspace
}

// Observation is one source's on-disk hash.
type Observation struct {
	SourceID string
	SHA256   string
}

// Workspace is the current worktree's code context.
type Workspace struct {
	WorktreeID string
	Head       string
	TreeState  string
}
