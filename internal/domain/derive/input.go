package derive

// Input is the complete input of a derivation (data-model §6, §6.13,
// §6.13.1): the facts up to AsOfSeq, the current-value inputs of one
// command, and the parameters. Everything is given explicitly: the package
// reads no database, git, file, environment variable or clock, and it never
// modifies the input.
type Input struct {
	Facts   Facts
	Current Current
	// AsOfSeq is the fact cutoff: only facts with seq <= AsOfSeq count.
	AsOfSeq int64
}

// Facts are the recorded facts of one project, as rows. A seq of 0 stands
// for NULL in the optional seq columns (seq starts at 1, data-model §1).
// The order of the rows never matters (MVP-ST-05).
type Facts struct {
	ProjectID string
	// LatestSeq is the highest seq the store the facts were read from has
	// allocated. A cutoff above it means facts are missing (a database from
	// an older backup, for example), and the derivation is refused.
	LatestSeq int64

	Targets            []Target
	BaselineVersions   []BaselineVersion
	BaselineChanges    []BaselineChange
	Entities           []Entity
	ScopeConfirmations []ScopeConfirmation
	ScopeItems         []ScopeItem
	Sources            []Source
	SourceAcks         []SourceAck
	SourceRefs         []SourceRef
	Proposals          []Proposal
	ProposalItems      []ProposalItem
	Questions          []Question
	ClaimBatches       []ClaimBatch
	Claims             []Claim
	CheckRuns          []CheckRun
	Commits            []Commit
	CommitLinks        []CommitLink
	Decisions          []Decision
}

// EntityType is the type of a baseline entity, or of what a source
// reference or a question is about.
type EntityType string

// Entity types.
const (
	GoalEntity        EntityType = "goal"
	RequirementEntity EntityType = "requirement"
	CriterionEntity   EntityType = "criterion"
	QuestionEntity    EntityType = "question"
)

// Target is a delivery target (data-model §4.2).
type Target struct {
	ID, Name, Ref string
	CreatedSeq    int64
	ArchivedSeq   int64 // 0: not archived
}

// BaselineVersion is a confirmed baseline version (data-model §4.7).
type BaselineVersion struct {
	ID     string
	Seq    int64
	Number int
}

// BaselineChange records an entity after a change (data-model §4.7).
type BaselineChange struct {
	ID          string
	Seq         int64
	VersionID   string
	ChangeIndex int
	EntityType  EntityType
	EntityID    string
	// ProposalItemID is the reviewed item the change applies, or "" for a
	// cascade and for a criterion follow-up.
	ProposalItemID string
	After          EntityState
}

// EntityState is an entity's snapshot after a change (after_json): its
// revision, status and field values. Links are by key.
type EntityState struct {
	Revision int
	Status   string // active / retired
	Title    string // goals and requirements
	// Requirements.
	GoalKey  string // "" when the requirement belongs to no goal
	Priority string
	// Criteria.
	RequirementRevision int
	Verification        string // automated / manual
	Check               *Check // nil when the criterion has no check
}

// Check is a criterion's check configuration, as far as it identifies the
// check (data-model §4.4): the timeout and the rationale are not part of it.
type Check struct {
	Command string
	Workdir string
	Shell   *string
	Inputs  []string
}

// Entity is the part of a baseline entity's row that never changes after it
// is created: its key, origin and, for a criterion, its requirement
// (data-model §6.13). Everything else is read from its changes.
type Entity struct {
	ID            string
	Type          EntityType
	Key           string
	Origin        string // extracted / inferred_from_code / manual
	RequirementID string // criteria only
}

// ScopeConfirmation is a confirmed delivery scope (data-model §4.2).
type ScopeConfirmation struct {
	ID              string
	Seq             int64
	TargetID        string
	BaselineVersion int
	SourceSetHash   string
}

// ScopeItem is a requirement of a scope confirmation; its identity is
// (ConfirmationID, RequirementID).
type ScopeItem struct {
	ConfirmationID      string
	RequirementID       string
	RequirementRevision int
}

// Source is a registered requirement source (data-model §4.3). AckedSHA256
// is the row's value: the hash at registration, or at the latest
// acknowledgement.
type Source struct {
	ID, Path, Kind string
	RegisteredSeq  int64
	RemovedSeq     int64 // 0: not removed
	AckedSHA256    string
}

// SourceAck is a person's acknowledgement of a source change.
type SourceAck struct {
	ID             string
	Seq            int64
	SourceID       string
	PreviousSHA256 string
	AckedSHA256    string
}

// SourceRef is a reference from an entity revision to a source
// (data-model §4.5).
type SourceRef struct {
	ID             string
	Seq            int64
	SourceID       string
	EntityType     EntityType
	EntityID       string
	EntityRevision int
	Verification   string // verified / not_found / source_missing / not_checkable
}

// Proposal is a submitted proposal; only its seq matters to a derivation.
type Proposal struct {
	ID  string
	Seq int64
}

// ProposalItem is an item of a proposal. ReviewedSeq is 0 until a person
// reviews it.
type ProposalItem struct {
	ID          string
	ProposalID  string
	ItemIndex   int
	ReviewState string // pending / accepted / accepted_with_edits / rejected / conflict
	ReviewedSeq int64
}

// Question is a question raised in a proposal (data-model §4.8). A question
// about an item of its proposal has ProposalItemID; one about an existing
// entity has RelatedType and RelatedID.
type Question struct {
	ID, Key        string
	CreatedSeq     int64
	AnsweredSeq    int64 // 0: open
	Blocking       bool
	RelatedType    EntityType
	RelatedID      string
	ProposalItemID string
}

// ClaimBatch is a submitted batch of claims (data-model §4.10).
type ClaimBatch struct {
	ID   string
	Seq  int64
	Kind string // sync / audit / manual
}

// Claim is an agent's assertion about a criterion.
type Claim struct {
	ID                string
	Seq               int64
	BatchID           string
	TargetType        string // criterion / none
	CriterionID       string
	CriterionRevision int
	Assertion         string // done / partial / not_done / blocked / unknown
}

// CheckRun is a recorded check execution (data-model §4.11). An empty head
// is NULL.
type CheckRun struct {
	ID                  string
	Seq                 int64
	CriterionID         string
	CriterionRevision   int
	RequirementRevision int
	CheckConfigHash     string
	Inputs              InputRecord // nil: NULL
	WorktreeID          string
	EvidenceGrade       Grade
	HeadStart, HeadEnd  string
	TreeStateStart      string // clean / dirty / unknown
	TreeStateEnd        string
	Result              string // pass / fail / error
	FailureKind         string // "" or nonzero_exit / timeout / not_found / spawn_error / input_missing / context_changed
}

// InputEntry is the identity of one declared check input (data-model
// §4.11.2): the saved path and the tree entry's mode, type and object ID.
type InputEntry struct {
	Path, Mode, Type, OID string
}

// InputRecord is a run's or a decision's input record; nil is NULL.
type InputRecord []InputEntry

// Commit is a commit found by a git scan (data-model §4.12).
type Commit struct {
	ID  string
	Seq int64
	SHA string
}

// CommitLink links a commit to a requirement or a criterion; its identity
// is (CommitID, TargetType, TargetID).
type CommitLink struct {
	CommitID   string
	TargetType EntityType
	TargetID   string
}

// Decision is a person's decision on a criterion (data-model §4.13). Empty
// strings are NULL.
type Decision struct {
	ID                  string
	Seq                 int64
	CriterionID         string
	Decision            string // accepted / rejected / waived
	Mode                string // accepted only: normal / manual / exception
	CriterionRevision   int
	RequirementRevision int
	CheckConfigHash     string
	Inputs              InputRecord
	TargetID            string
	Snapshot            string
	EvidenceID          string
	EvidenceGrade       Grade
	Rationale           string
}

// Current holds the current-value inputs of one command (data-model
// §6.13.1). They have no seq; a command takes each of them once, and every
// derivation of that command uses the same values. Snapshot and Settings
// are required: a nil one is a missing input, never filled in.
type Current struct {
	Snapshot *Snapshot
	// Observations has one entry for each source in the source set at the
	// cutoff, and no other.
	Observations []Observation
	Settings     *Settings
	// Workspace is the current worktree's code context; it is optional and
	// used only for the workspace view.
	Workspace *Workspace
}

// Snapshot is S: the commit the active target's ref resolves to, or NULL
// (Commit "") when it does not resolve, there is no target or the project
// is not a git project.
type Snapshot struct {
	Commit string
}

// Observation is the on-disk hash of a source. SHA256 "" means the file was
// observed missing (NULL); that is different from having no observation of
// the source, which is a missing input.
type Observation struct {
	SourceID string
	SHA256   string
}

// Stage is the project's stage setting.
type Stage string

// Stages.
const (
	DocsOnly       Stage = "docs_only"
	InDevelopment  Stage = "in_development"
	IncompleteDocs Stage = "incomplete_docs"
)

// Settings are the project settings a derivation reads (data-model §6.13):
// only these three, never the rest of settings_json.
type Settings struct {
	Stage            Stage
	SourcesSkipped   bool
	InventorySkipped bool
}

// Workspace is a worktree's code context; an empty Head is NULL.
type Workspace struct {
	WorktreeID string
	Head       string
	TreeState  string
}
