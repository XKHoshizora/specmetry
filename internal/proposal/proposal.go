// Package proposal checks and stores proposals (data-model.md §4.6, §8.3;
// architecture §6.1.4). A proposal never changes the baseline: its items
// wait as pending for a person's review (WP1.6), and its questions are
// opened at once. Source excerpts are only checked in memory; under
// hash_only their text is never stored, returned or reported.
package proposal

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
)

// Proposal is a proposal.v1 document that passed schema validation.
type Proposal struct {
	Origin      string     `json:"origin"`
	BaseVersion int        `json:"base_version"`
	Summary     string     `json:"summary"`
	Items       []Item     `json:"items"`
	Questions   []Question `json:"questions"`
}

// Item is one proposed change.
type Item struct {
	Op             string                     `json:"op"`
	Type           string                     `json:"type"`
	Ref            string                     `json:"ref"`
	Target         string                     `json:"target"`
	TargetRevision int                        `json:"target_revision"`
	ChangeClass    string                     `json:"change_class"`
	Data           map[string]json.RawMessage `json:"data"`
	Origin         string                     `json:"origin"`
	Confidence     string                     `json:"confidence"`
	Rationale      string                     `json:"rationale"`
	Sources        []SourceRef                `json:"sources"`
	CodeRefs       []CodeRef                  `json:"code_refs"`
}

// SourceRef cites a registered source.
type SourceRef struct {
	Source  string  `json:"source"`
	Locator *string `json:"locator"`
	Excerpt string  `json:"excerpt"`
}

// CodeRef names a file of the repository, and optionally its lines.
type CodeRef struct {
	Path  string `json:"path"`
	Lines string `json:"lines"`
}

// Question is a question to clarify.
type Question struct {
	Text     string      `json:"text"`
	Context  string      `json:"context"`
	About    string      `json:"about"`
	Blocking bool        `json:"blocking"`
	Sources  []SourceRef `json:"sources"`
}

// Parse decodes a document that passed proposal.v1 validation.
func Parse(raw []byte) (*Proposal, error) {
	var p Proposal
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, apperr.New(apperr.InvalidInput, "the proposal is not valid")
	}
	return &p, nil
}

var localRef = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// IsLocalRef reports whether s is a proposal-local ref rather than a key.
func IsLocalRef(s string) bool { return localRef.MatchString(s) }

// stringData returns the string at key in data, and whether it is one
// (absent and null are not).
func stringData(data map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := data[key]
	if !ok {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}

// check is a criterion's check, when data sets one.
type check struct {
	Workdir string   `json:"workdir"`
	Inputs  []string `json:"inputs"`
}

func checkData(data map[string]json.RawMessage) (*check, bool) {
	raw, ok := data["check"]
	if !ok || string(raw) == "null" {
		return nil, ok
	}
	var c check
	if json.Unmarshal(raw, &c) != nil {
		return nil, true
	}
	return &c, true
}

// insideRepository reports whether a repository-relative path stays in the
// repository once normalized: both separators, "." and ".." resolved
// lexically (V-P04). The schema already refuses absolute paths and ".."
// segments; this holds whatever the schema allows.
func insideRepository(p string) bool {
	clean := path.Clean(strings.ReplaceAll(p, `\`, "/"))
	return clean != ".." && !strings.HasPrefix(clean, "../") && !strings.HasPrefix(clean, "/") && !strings.Contains(p, "\x00")
}

// problems collects what is wrong with a proposal, each at a JSON Pointer.
type problems []apperr.Detail

func (ps *problems) add(pointer, format string, args ...any) {
	*ps = append(*ps, apperr.Detail{Pointer: pointer, Problem: fmt.Sprintf(format, args...)})
}

func (ps problems) err() error {
	if len(ps) == 0 {
		return nil
	}
	return apperr.New(apperr.InvalidInput, "the proposal is not valid; nothing was written").WithDetails(ps...)
}

func itemPointer(i int, rest ...string) string {
	return "/items/" + strconv.Itoa(i) + strings.Join(append([]string{""}, rest...), "/")
}

// CheckStatic applies the rules that need no database (V-P03 – V-P06,
// V-P08): no code fence in any string (doc is the decoded document), refs
// unique and of the right type, every target named once and of the item's
// type, paths inside the repository, line ranges in order, and excerpts
// with text besides whitespace. Errors name fields, never their values.
func CheckStatic(doc any, p *Proposal) error {
	var ps problems
	fences(doc, "", &ps)

	type added struct {
		index int
		typ   string
	}
	refs := map[string]added{}
	targets := map[string]int{}
	for i, it := range p.Items {
		if it.Op == "add" {
			if prev, dup := refs[it.Ref]; dup {
				ps.add(itemPointer(i, "ref"), "is the ref of item %d already; refs are unique in a proposal", prev.index)
			} else {
				refs[it.Ref] = added{i, it.Type}
			}
			continue
		}
		if prev, dup := targets[it.Target]; dup {
			ps.add(itemPointer(i, "target"), "is the target of item %d already; a proposal changes an entity once", prev)
		} else {
			targets[it.Target] = i
		}
		if typ, _ := baseline.TypeOfKey(it.Target); string(typ) != it.Type {
			ps.add(itemPointer(i, "target"), "is a %s key, but the item is about a %s", typ, it.Type)
		}
	}
	refersTo := func(pointer, ref string, types ...string) {
		a, ok := refs[ref]
		switch {
		case !ok:
			ps.add(pointer, "names no item added by this proposal")
		case !slices.Contains(types, a.typ):
			ps.add(pointer, "names a %s; it must name a %s", a.typ, strings.Join(types, " or "))
		}
	}
	for i, it := range p.Items {
		if goal, ok := stringData(it.Data, "goal"); ok && IsLocalRef(goal) {
			refersTo(itemPointer(i, "data", "goal"), goal, "goal")
		}
		if req, ok := stringData(it.Data, "requirement"); ok && IsLocalRef(req) {
			refersTo(itemPointer(i, "data", "requirement"), req, "requirement")
		}
		if c, _ := checkData(it.Data); c != nil {
			if c.Workdir != "" && !insideRepository(c.Workdir) {
				ps.add(itemPointer(i, "data", "check", "workdir"), "leaves the repository")
			}
			for k, in := range c.Inputs {
				if !insideRepository(in) {
					ps.add(itemPointer(i, "data", "check", "inputs", strconv.Itoa(k)), "leaves the repository")
				}
			}
		}
		for k, c := range it.CodeRefs {
			if !insideRepository(c.Path) {
				ps.add(itemPointer(i, "code_refs", strconv.Itoa(k), "path"), "leaves the repository")
			}
			if from, to, ok := lineRange(c.Lines); ok && from > to {
				ps.add(itemPointer(i, "code_refs", strconv.Itoa(k), "lines"), "ends before it starts")
			}
		}
		excerpts(it.Sources, itemPointer(i, "sources"), &ps)
	}
	for i, q := range p.Questions {
		pointer := "/questions/" + strconv.Itoa(i)
		if q.About != "" && IsLocalRef(q.About) {
			refersTo(pointer+"/about", q.About, "requirement", "criterion")
		}
		excerpts(q.Sources, pointer+"/sources", &ps)
	}
	return ps.err()
}

func excerpts(refs []SourceRef, pointer string, ps *problems) {
	for k, r := range refs {
		if Normalize(r.Excerpt) == "" {
			ps.add(pointer+"/"+strconv.Itoa(k)+"/excerpt", "has no text besides whitespace, so it would be found anywhere")
		}
	}
}

// fences reports every string value with a code fence (V-P05): text fields
// hold no code.
func fences(v any, pointer string, ps *problems) {
	switch v := v.(type) {
	case string:
		if strings.Contains(v, "```") {
			ps.add(pointerOrRoot(pointer), "contains a code fence (```); text fields hold no code")
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			fences(v[k], pointer+"/"+strings.NewReplacer("~", "~0", "/", "~1").Replace(k), ps)
		}
	case []any:
		for i, e := range v {
			fences(e, pointer+"/"+strconv.Itoa(i), ps)
		}
	}
}

func pointerOrRoot(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

// lineRange parses "a" or "a-b".
func lineRange(lines string) (from, to int, ok bool) {
	if lines == "" {
		return 0, 0, false
	}
	a, b, isRange := strings.Cut(lines, "-")
	from, err := strconv.Atoi(a)
	if err != nil {
		return 0, 0, false
	}
	to = from
	if isRange {
		if to, err = strconv.Atoi(b); err != nil {
			return 0, 0, false
		}
	}
	return from, to, true
}
