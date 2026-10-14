package testutil

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/contract"
)

// proposalProblems checks the recorded proposal of fx-docs-only against the
// brief and returns what is wrong.
func proposalProblems(t *testing.T, raw []byte, brief string) []string {
	t.Helper()
	var problems []string
	problemf := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	v, err := contract.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := v.ValidateJSON("proposal.v1", raw); err != nil {
		return []string{"not a valid proposal.v1: " + err.Error()}
	}
	var p struct {
		Origin      string `json:"origin"`
		BaseVersion int    `json:"base_version"`
		Items       []struct {
			Op   string `json:"op"`
			Type string `json:"type"`
			Ref  string `json:"ref"`
			Data struct {
				Requirement string `json:"requirement"`
			} `json:"data"`
			Sources []proposalSource `json:"sources"`
		} `json:"items"`
		Questions []struct {
			About    string           `json:"about"`
			Blocking bool             `json:"blocking"`
			Sources  []proposalSource `json:"sources"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return []string{err.Error()}
	}
	if p.Origin != "skill_init" || p.BaseVersion != 0 {
		problemf("origin %q, base_version %d", p.Origin, p.BaseVersion)
	}
	normBrief := normalizeSpace(brief)
	checkExcerpt := func(where, source, excerpt string) {
		if source != "docs/client-brief.md" {
			problemf("%s cites %q", where, source)
		}
		if !strings.Contains(normBrief, normalizeSpace(excerpt)) {
			problemf("%s: excerpt %q is not in the brief", where, excerpt)
		}
	}
	sections := briefSections(brief)
	// sectionOf says which section of the brief an entry cites: the oracle
	// below is about the brief's sections, never about the proposal's refs.
	sectionOf := func(where string, sources []proposalSource) string {
		if len(sources) == 0 {
			problemf("%s cites nothing", where)
			return ""
		}
		var found []string
		for name, text := range sections {
			if strings.Contains(text, normalizeSpace(sources[0].Excerpt)) {
				found = append(found, name)
			}
		}
		if len(found) != 1 {
			problemf("%s: excerpt %q is in sections %v, want exactly one", where, sources[0].Excerpt, found)
			return ""
		}
		return found[0]
	}

	goals := 0
	requirementSection := map[string]string{} // requirement ref -> brief section
	criteriaPerSection := map[string]int{}
	type criterion struct{ where, requirement, section string }
	var criteria []criterion
	for i, it := range p.Items {
		where := "item " + strconv.Itoa(i) + " (" + it.Ref + ")"
		if it.Op != "add" {
			problemf("%s: op %q in an initial proposal", where, it.Op)
		}
		for _, src := range it.Sources {
			checkExcerpt(where, src.Source, src.Excerpt)
		}
		switch it.Type {
		case "goal":
			goals++
		case "requirement":
			sec := sectionOf(where, it.Sources)
			for ref, other := range requirementSection {
				if other == sec {
					problemf("%s and %s both stand for brief section %s", ref, it.Ref, sec)
				}
			}
			requirementSection[it.Ref] = sec
		case "criterion":
			criteria = append(criteria, criterion{where, it.Data.Requirement, sectionOf(where, it.Sources)})
		}
	}
	for _, c := range criteria {
		sec, ok := requirementSection[c.requirement]
		switch {
		case !ok:
			problemf("%s belongs to %q, which is not a requirement of the proposal", c.where, c.requirement)
		case sec != c.section:
			problemf("%s cites section %s but belongs to %s (section %s)", c.where, c.section, c.requirement, sec)
		default:
			criteriaPerSection[sec]++
		}
	}
	if goals != briefOracle.goals {
		problemf("%d goals, the brief yields %d", goals, briefOracle.goals)
	}
	if len(requirementSection) != len(briefOracle.criteria) {
		problemf("%d requirements, the brief has %d", len(requirementSection), len(briefOracle.criteria))
	}
	for sec, want := range briefOracle.criteria {
		if got := criteriaPerSection[sec]; got != want {
			problemf("section %s has %d criteria, the brief yields %d", sec, got, want)
		}
	}

	if len(p.Questions) != len(briefOracle.questions) {
		problemf("%d questions, the brief raises %d", len(p.Questions), len(briefOracle.questions))
	}
	matched := map[string]bool{}
	for i, q := range p.Questions {
		where := "question " + strconv.Itoa(i)
		for _, src := range q.Sources {
			checkExcerpt(where, src.Source, src.Excerpt)
		}
		sec := sectionOf(where, q.Sources)
		want, ok := briefOracle.questions.question(sec)
		if !ok || matched[sec] {
			problemf("%s cites section %s, which raises no other question", where, sec)
			continue
		}
		matched[sec] = true
		if q.Blocking != want.blocking {
			problemf("%s (section %s): blocking = %v, want %v", where, sec, q.Blocking, want.blocking)
		}
		if want.about == "" && q.About != "" {
			problemf("%s (section %s) must be project-level, not about %q", where, sec, q.About)
		}
		if want.about != "" && requirementSection[q.About] != want.about {
			problemf("%s (section %s) is about %q, want the requirement of section %s", where, sec, q.About, want.about)
		}
		if want.phrase != "" && !strings.Contains(q.Sources[0].Excerpt, want.phrase) {
			problemf("%s (section %s) must quote the ambiguity %q", where, sec, want.phrase)
		}
	}
	return problems
}

// codex-review-012 N1: the check must catch these mutations of the recorded
// proposal, which a weaker check let through.
func TestRecordedProposalCheckCatchesMutations(t *testing.T) {
	source := filepath.Join(fixturesRoot(t), "fx-docs-only")
	raw := readFile(t, filepath.Join(source, "proposal.json"))
	brief := string(readFile(t, filepath.Join(source, "repo", "docs", "client-brief.md")))
	if p := proposalProblems(t, raw, brief); len(p) != 0 {
		t.Fatalf("the recorded proposal itself: %v", p)
	}
	mutations := map[string]func(doc map[string]any){
		"c3 deleted": func(doc map[string]any) {
			doc["items"] = slices.DeleteFunc(doc["items"].([]any), func(it any) bool { return it.(map[string]any)["ref"] == "c3" })
		},
		"the non-blocking questions deleted": func(doc map[string]any) {
			doc["questions"] = slices.DeleteFunc(doc["questions"].([]any), func(q any) bool { return q.(map[string]any)["blocking"] == false })
		},
		"c9 attached to r8": func(doc map[string]any) {
			for _, it := range doc["items"].([]any) {
				if item := it.(map[string]any); item["ref"] == "c9" {
					item["data"].(map[string]any)["requirement"] = "r8"
				}
			}
		},
	}
	for name, mutate := range mutations {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		mutate(doc)
		mutated, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		p := proposalProblems(t, mutated, brief)
		if len(p) == 0 {
			t.Errorf("%s: not detected", name)
		}
		t.Logf("%s: %v", name, p)
	}
}

type proposalSource struct {
	Source  string `json:"source"`
	Excerpt string `json:"excerpt"`
}

type oracleQuestion struct {
	section  string // the brief section the question comes from
	about    string // the requirement section it is about; "" for project-level
	blocking bool
	phrase   string // the ambiguous wording it must quote, if any
}

// briefOracle is what the fx-docs-only brief supports, written down from the
// brief itself, not derived from the proposal: the criteria each requirement
// section yields (R8 has no decidable acceptance description, so none), and
// the questions it raises. The only blocking question is the project-level
// one of the "待确认" section.
var briefOracle = struct {
	goals     int
	criteria  map[string]int
	questions oracleQuestionSet
}{
	goals:    1,
	criteria: map[string]int{"R1": 1, "R2": 2, "R3": 1, "R4": 1, "R5": 2, "R6": 1, "R7": 1, "R8": 0},
	questions: oracleQuestionSet{
		{section: "R3", about: "R3", phrase: "合适数量"},
		{section: "R7", about: "R7", phrase: "尽快"},
		{section: "R8", about: "R8"},
		{section: "待确认", blocking: true},
	},
}

func init() {
	// Keep the oracle consistent with itself: 9 criteria, one blocking
	// question.
	total, blocking := 0, 0
	for _, n := range briefOracle.criteria {
		total += n
	}
	for _, q := range briefOracle.questions {
		if q.blocking {
			blocking++
		}
	}
	if total != 9 || blocking != 1 {
		panic("briefOracle: want 9 criteria and one blocking question")
	}
}

func (o oracleQuestionSet) question(section string) (oracleQuestion, bool) {
	for _, q := range o {
		if q.section == section {
			return q, true
		}
	}
	return oracleQuestion{}, false
}

type oracleQuestionSet []oracleQuestion

var sectionHeading = regexp.MustCompile(`(?m)^#{2,3} (\S+)`)

// briefSections splits the brief at its "##" and "###" headings and returns
// each section's whitespace-normalised text, keyed by the heading's first
// word (R1 ... R8, 背景, 需求, 待确认).
func briefSections(brief string) map[string]string {
	out := map[string]string{}
	locs := sectionHeading.FindAllStringSubmatchIndex(brief, -1)
	for i, loc := range locs {
		end := len(brief)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		out[brief[loc[2]:loc[3]]] = normalizeSpace(brief[loc[1]:end])
	}
	return out
}
