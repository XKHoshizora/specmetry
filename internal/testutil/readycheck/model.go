// Package readycheck is the independent ready checker of invariant #2
// (acceptance-criteria §3, §3.1, F5): test code that decides whether a
// delivery is ready from the facts, with the predicate R1 to R10 written
// from prd §3.2, the invariant and data-model §6.8, not from the
// derivation engine. It imports only the standard library and never the
// engine or a production package; its front-ends (fromdump, and later the
// E2E database reader) only map rows into the types below.
package readycheck

// Facts are the facts the checker reads, in its own representation. A seq
// of 0 is NULL, as is an empty string; the order of rows never matters.
type Facts struct {
	Targets    []Target
	Versions   []Version
	Entities   []Entity
	Changes    []Change
	Scopes     []Scope
	ScopeItems []ScopeItem
	Sources    []Source
	Acks       []Ack
	Items      []Item
	Questions  []Question
	Runs       []Run
	Decisions  []Decision
}

// Target is a delivery target's interval.
type Target struct {
	ID                string
	Created, Archived int64
}

// Version is a baseline version.
type Version struct {
	Seq    int64
	Number int
}

// Entity is a baseline entity; its kind and requirement never change.
type Entity struct {
	ID          string
	Key         string
	Kind        string // goal / requirement / criterion
	Requirement string // criteria
}

// Change is an entity's state after a change.
type Change struct {
	ID                  string
	Seq                 int64
	Index               int
	Entity              string
	Item                string // the proposal item it applies, or ""
	Status              string
	Revision            int
	RequirementRevision int
	Verification        string
	Check               *CheckConfig
}

// CheckConfig is a check's configuration.
type CheckConfig struct {
	Command, Workdir string
	Shell            *string
	Inputs           []string
}

// Scope is a scope confirmation.
type Scope struct {
	ID            string
	Seq           int64
	Target        string
	Version       int
	SourceSetHash string
}

// ScopeItem is a requirement of a scope confirmation.
type ScopeItem struct {
	Scope, Requirement string
}

// Source is a registered source; Hash is the row's hash (at registration,
// or at the latest acknowledgement).
type Source struct {
	ID                  string
	Registered, Removed int64
	Hash                string
}

// Ack is an acknowledgement of a source's new hash.
type Ack struct {
	ID       string
	Seq      int64
	Source   string
	Previous string
	Hash     string
}

// Item is a proposal item's review.
type Item struct {
	ID       string
	State    string
	Reviewed int64
}

// Question is a question and what it is about: a proposal item or an
// existing entity.
type Question struct {
	ID, Key           string
	Created, Answered int64
	Blocking          bool
	Item              string
	About             string
}

// Run is a check run, with the execution context F5 lists.
type Run struct {
	ID                  string
	Seq                 int64
	Criterion           string
	Revision            int
	RequirementRevision int
	ConfigHash          string
	Worktree            string
	Inputs              []InputEntry // nil: NULL
	HeadStart, HeadEnd  string
	TreeStart, TreeEnd  string
	Result              string
	FailureKind         string
	Grade               string
}

// InputEntry is one entry of an input record.
type InputEntry struct {
	Path, Mode, Type, OID string
}

// Decision is a person's decision, with the timing F5 lists: the
// evidence, the snapshot, the mode, the seq and the decision the person
// saw as current ("" when none). The predicate does not read SeenDecision
// or RequirementRevision.
type Decision struct {
	ID                  string
	Seq                 int64
	Criterion           string
	Kind                string // accepted / rejected / waived
	Mode                string
	Revision            int
	RequirementRevision int
	ConfigHash          string
	Inputs              []InputEntry
	Snapshot            string
	EvidenceID          string
	SeenDecision        string
}

// Current is the current-value input the checker needs: S and the
// observations ("" is a file observed missing). The predicate reads no
// project setting.
type Current struct {
	S            string
	Observations map[string]string
}

// Condition is one condition of the predicate.
type Condition string

// The conditions of the ready predicate (acceptance-criteria §3.1).
const (
	R1 Condition = "R1" // a baseline version exists
	R2 Condition = "R2" // an active target exists and S is not NULL
	R3 Condition = "R3" // the target's scope exists and is not stale
	R4 Condition = "R4" // RS is not empty
	R5 Condition = "R5" // every requirement of RS has an active criterion
	R6 Condition = "R6" // every criterion of AS has a decision that applies on S
	R7 Condition = "R7" // AS is not all waived
	R8 Condition = "R8" // no blocking question
	R9 Condition = "R9" // no unconfirmed source change
)

// Verdict is the checker's conclusion.
type Verdict struct {
	// Failed holds each condition that does not hold, with the keys or
	// IDs that make it fail.
	Failed map[Condition][]string
	// Grade is R10's recomputed delivery evidence grade: the weakest grade
	// of the runs the normal acceptances of AS rely on; "none" without one.
	Grade string
}

// Ready reports whether every condition holds.
func (v Verdict) Ready() bool { return len(v.Failed) == 0 }
