package derive

// Grade is an evidence grade (data-model §4.11.1, §6.8).
type Grade string

// Evidence grades. NoEvidence is only a delivery or decision grade: no
// automated run supports the acceptance.
const (
	Observed         Grade = "observed"
	SnapshotIsolated Grade = "snapshot_isolated"
	NoEvidence       Grade = "none"
)

// CriterionStatus is a criterion's delivery status (data-model §6.3).
type CriterionStatus string

// Criterion delivery statuses.
const (
	CriterionNotAssessed         CriterionStatus = "not_assessed"
	CriterionInProgress          CriterionStatus = "in_progress"
	CriterionFailing             CriterionStatus = "failing"
	CriterionRejected            CriterionStatus = "rejected"
	CriterionNeedsReverification CriterionStatus = "needs_reverification"
	CriterionReadyForReview      CriterionStatus = "ready_for_review"
	CriterionAccepted            CriterionStatus = "accepted"
	CriterionWaived              CriterionStatus = "waived"
)

// RequirementStatus is a requirement's delivery status (data-model §6.7).
type RequirementStatus string

// Requirement delivery statuses.
const (
	RequirementNeedsCriteria       RequirementStatus = "needs_criteria"
	RequirementNotAssessed         RequirementStatus = "not_assessed"
	RequirementFailing             RequirementStatus = "failing"
	RequirementRejected            RequirementStatus = "rejected"
	RequirementInProgress          RequirementStatus = "in_progress"
	RequirementNeedsReverification RequirementStatus = "needs_reverification"
	RequirementReadyForReview      RequirementStatus = "ready_for_review"
	RequirementWaived              RequirementStatus = "waived"
	RequirementAccepted            RequirementStatus = "accepted"
)

// TargetStatus is the delivery status of the target (data-model §6.8).
type TargetStatus string

// Target delivery statuses.
const (
	TargetNotAssessed         TargetStatus = "not_assessed"
	TargetInProgress          TargetStatus = "in_progress"
	TargetNeedsReverification TargetStatus = "needs_reverification"
	TargetReadyForReview      TargetStatus = "ready_for_review"
	TargetReady               TargetStatus = "ready"
)

// Progress is a criterion's progress, from claims and commits only
// (data-model §6.5).
type Progress string

// Progress values.
const (
	ProgressClaimed    Progress = "claimed"
	ProgressBlocked    Progress = "blocked"
	ProgressInProgress Progress = "in_progress"
	ProgressNotStarted Progress = "not_started"
	ProgressUnknown    Progress = "unknown"
)

// Flag is an informational flag (data-model §6.6). Flags never block and
// never change a delivery status.
type Flag string

// Flags, in the order they are reported.
const (
	FlagConflict               Flag = "conflict"
	FlagClaimedWithoutEvidence Flag = "claimed_without_evidence"
	FlagInconsistentResults    Flag = "inconsistent_results"
	FlagExceptionAcceptance    Flag = "exception_acceptance"
	FlagObservedEvidence       Flag = "observed_evidence"
	FlagBlockingQuestion       Flag = "blocking_question"
	FlagRefNotFound            Flag = "ref_not_found"
	FlagInferred               Flag = "inferred"
	FlagSourceRemoved          Flag = "source_removed"
)

// SourceState compares a source's observation with its acknowledged hash.
type SourceState string

// Source states.
const (
	SourceUnchanged SourceState = "unchanged"
	SourceChanged   SourceState = "changed"
	SourceMissing   SourceState = "missing"
)

// Result is everything a derivation determines (data-model §6), for the
// status and gaps outputs. Lists are in key order.
type Result struct {
	AsOfSeq int64
	// Snapshot, Observations and Settings are the current-value inputs this
	// derivation used, so an output records exactly those (data-model
	// §6.13.1, MVP-ST-15). Observations are in source ID order.
	Snapshot     Snapshot
	Observations []Observation
	Settings     Settings

	BaselineVersion   int
	InitProgress      InitProgress
	InventoryCoverage *Ratio // nil when there is no active criterion
	SourceSetHash     string
	Target            *TargetView
	Scope             *ScopeView
	Delivery          Delivery
	Workspace         *WorkspaceView
	Goals             []GoalView
	Sources           []SourceView
	Requirements      []RequirementView
	OpenQuestions     []QuestionView
	Gaps              []Gap
	Metrics           Metrics
}

// Ratio is an exact fraction; Den is never 0.
type Ratio struct {
	Num, Den int
}

// TargetView is the active target and S.
type TargetView struct {
	ID, Name, Ref string
	Snapshot      string // "" when S is NULL
}

// ScopeView is the target's current scope confirmation.
type ScopeView struct {
	ConfirmationID  string
	BaselineVersion int
	SourceSetHash   string
	Stale           bool
	Requirements    []string // the keys of the confirmed requirements
}

// Delivery is the delivery judgement on the target (data-model §6.8).
type Delivery struct {
	Status        TargetStatus
	EvidenceGrade Grade
	BlockingGaps  []Gap
}

// WorkspaceView is the workspace view: informational only.
type WorkspaceView struct {
	WorktreeID       string
	Head             string
	TreeState        string
	IsTargetSnapshot bool // the worktree's HEAD is S
}

// GoalView is an active goal.
type GoalView struct {
	Key      string
	Revision int
	Title    string
}

// SourceView is a source in the source set at the cutoff.
type SourceView struct {
	ID, Path, Kind string
	AckedSHA256    string // acknowledged at the cutoff
	CurrentSHA256  string // the observation; "" when missing
	State          SourceState
}

// RequirementView is an active requirement and its active criteria.
type RequirementView struct {
	Key      string
	Revision int
	Goal     string // "" when none
	Title    string
	Priority string
	InScope  bool
	Status   RequirementStatus
	Flags    []Flag
	Criteria []CriterionView
}

// CriterionView is an active criterion.
type CriterionView struct {
	Key                 string
	Revision            int
	RequirementRevision int
	Verification        string
	HasCheck            bool
	CheckConfigHash     string // "" without a check
	Status              CriterionStatus
	Reason              string // "" when the status has no reason code
	Progress            Progress
	Flags               []Flag
	CurrentDecision     *DecisionView
	// SnapshotRun is V's run: the latest qualifying run on S for the current
	// revision and configuration.
	SnapshotRun *RunView
	// WorkspaceRun is the latest run of the current revision and
	// configuration in the current worktree at its HEAD, qualifying or not.
	WorkspaceRun *RunView
}

// DecisionView summarizes the current decision.
type DecisionView struct {
	ID            string
	Seq           int64
	Decision      string
	Mode          string
	Snapshot      string
	EvidenceGrade Grade
	Rationale     string
}

// RunView is a check run as the outputs report it.
type RunView struct {
	ID            string
	Seq           int64
	Result        string
	EvidenceGrade Grade
	FailureKind   string
	Qualifying    bool
}

// QuestionView is an open question at the cutoff.
type QuestionView struct {
	Key      string
	Blocking bool
	// ItemRejected is true when the question is about a proposal item that
	// was rejected: it no longer counts as blocking (data-model §4.8).
	ItemRejected bool
	// About is the key of the entity the question is about, or "".
	About string
}

// GapType is a gap's type (data-model §6.10).
type GapType string

// Gap types, in the order gaps are reported (§6.10: the project table,
// the entity table, then the warnings).
const (
	GapNoBaseline             GapType = "NO_BASELINE"
	GapNoTarget               GapType = "NO_TARGET"
	GapTargetUnresolved       GapType = "TARGET_UNRESOLVED"
	GapNoScope                GapType = "NO_SCOPE"
	GapEmptyScope             GapType = "EMPTY_SCOPE"
	GapAllWaived              GapType = "ALL_WAIVED"
	GapScopeStale             GapType = "SCOPE_STALE"
	GapSourceChanged          GapType = "SOURCE_CHANGED"
	GapBlockingQuestion       GapType = "BLOCKING_QUESTION"
	GapPendingReview          GapType = "PENDING_REVIEW"
	GapInventoryIncomplete    GapType = "INVENTORY_INCOMPLETE"
	GapNeedsCriteria          GapType = "NEEDS_CRITERIA"
	GapNoCheck                GapType = "NO_CHECK"
	GapClaimedWithoutEvidence GapType = "CLAIMED_WITHOUT_EVIDENCE"
	GapNotVerified            GapType = "NOT_VERIFIED"
	GapFailing                GapType = "FAILING"
	GapRejected               GapType = "REJECTED"
	GapParentRevised          GapType = "PARENT_REVISED"
	GapNeedsReverification    GapType = "NEEDS_REVERIFICATION"
	GapAwaitingReview         GapType = "AWAITING_REVIEW"
	GapConflict               GapType = "CONFLICT"
	GapExceptionAcceptance    GapType = "EXCEPTION_ACCEPTANCE"
	GapInconsistentResults    GapType = "INCONSISTENT_RESULTS"
	GapSourceRemoved          GapType = "SOURCE_REMOVED"
)

// Gap is a gap: what stands between the target and ready, or a warning.
type Gap struct {
	Type     GapType
	Blocking bool
	// TargetKey is the requirement, criterion, goal or question key, or the
	// source path; "" for a project-level gap.
	TargetKey  string
	Reason     string // "" when none
	NextAction string
}

// Metrics are the derived counts (data-model §6.11).
type Metrics struct {
	ScopeAcceptanceRatio        *Ratio // nil when the denominator is 0
	ACStatusCounts              map[CriterionStatus]int
	ClaimedWithoutEvidence      int
	PendingReviewItems          int
	OpenQuestions               int
	BlockingQuestions           int
	ExceptionAcceptances        int
	ObservedEvidenceAcceptances int
}

// Change is a change of a criterion's delivery status or progress between
// two cutoffs (--diff-since). From is "" when the criterion was not active
// at the start.
type Change struct {
	Criterion string
	Field     string // delivery_status / progress
	From, To  string
}
