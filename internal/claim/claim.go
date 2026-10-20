// Package claim checks and stores claim batches (data-model.md §4.10,
// WP2.4 spec). A claim is an Agent's self-report about an acceptance
// criterion; it never makes a criterion accepted or ready on its own.
package claim

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/proposal"
)

// Batch kinds.
const (
	Sync   = "sync"
	Audit  = "audit"
	Manual = "manual"
)

// Batch is a claims.v1 document that passed schema validation.
type Batch struct {
	Kind         string     `json:"kind"`
	BaseVersion  int        `json:"base_version"`
	Summary      string     `json:"summary"`
	AuditScope   *Scope     `json:"audit_scope"`
	Claims       []Claim    `json:"claims"`
	UnmappedWork []Unmapped `json:"unmapped_work"`
}

// Scope is an audit batch's declared scope: all requirements, or some.
type Scope struct {
	All          bool     `json:"all"`
	Requirements []string `json:"requirements"`
}

// Claim is an assertion about a criterion at a revision.
type Claim struct {
	Target         string `json:"target"`
	TargetRevision int    `json:"target_revision"`
	Assertion      string `json:"assertion"`
	Summary        string `json:"summary"`
	Refs           Refs   `json:"refs"`
}

// Unmapped is work that maps to no criterion (MVP-INV-06, P1).
type Unmapped struct {
	Summary string `json:"summary"`
	Refs    Refs   `json:"refs"`
}

// Refs are a claim's references: files with optional line ranges, test
// names and commit ID prefixes.
type Refs struct {
	Files   []proposal.CodeRef `json:"files"`
	Tests   []string           `json:"tests"`
	Commits []string           `json:"commits"`
}

// Parse decodes a document that passed claims.v1 validation.
func Parse(raw []byte) (*Batch, error) {
	var b Batch
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, apperr.New(apperr.InvalidInput, "the claim batch is not valid")
	}
	return &b, nil
}

// problems collects what is wrong with a batch, each at a JSON Pointer.
type problems []apperr.Detail

func (ps *problems) add(pointer, format string, args ...any) {
	*ps = append(*ps, apperr.Detail{Pointer: pointer, Problem: fmt.Sprintf(format, args...)})
}

func (ps problems) err() error {
	if len(ps) == 0 {
		return nil
	}
	return apperr.New(apperr.InvalidInput, "the claim batch is not valid; nothing was written").WithDetails(ps...)
}

// blankLine is claims.v1's test for a summary of more than one paragraph
// (common.v1 shortSummary).
var blankLine = regexp.MustCompile(`\n[ \t]*\n`)

func summary(pointer, s string, ps *problems) {
	if strings.Contains(s, "```") {
		ps.add(pointer, "contains a code fence; a summary holds no code")
	}
	if blankLine.MatchString(s) {
		ps.add(pointer, "has a blank line; a summary is a single paragraph")
	}
}

func refs(pointer string, r Refs, ps *problems) {
	for k, f := range r.Files {
		*ps = append(*ps, proposal.CodeRefProblems(pointer+"/refs/files/"+strconv.Itoa(k), f)...)
	}
}

// CheckStatic applies the rules that need no database (WP2.4 spec §5 step
// 4): a criterion claimed once in a batch (V-C03, the later claim pointed
// at), no summary with a code fence or a blank line (V-C04, checked again
// past the schema), and every file reference inside the repository with a
// valid line range, by the rules a proposal's code references follow
// (V-P04). All the problems are reported together; none quotes a value.
func CheckStatic(b *Batch) error {
	var ps problems
	summary("/summary", b.Summary, &ps)
	first := map[string]int{}
	for i, c := range b.Claims {
		pointer := "/claims/" + strconv.Itoa(i)
		if prev, dup := first[c.Target]; dup {
			ps.add(pointer+"/target", "is the target of claim %d already; a batch claims a criterion once", prev)
		} else {
			first[c.Target] = i
		}
		summary(pointer+"/summary", c.Summary, &ps)
		refs(pointer, c.Refs, &ps)
	}
	for i, u := range b.UnmappedWork {
		pointer := "/unmapped_work/" + strconv.Itoa(i)
		summary(pointer+"/summary", u.Summary, &ps)
		refs(pointer, u.Refs, &ps)
	}
	return ps.err()
}
