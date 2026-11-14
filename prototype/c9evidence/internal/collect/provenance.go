package collect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// VerifyProvenance checks the source chain of record items in the order
// they were appended (plan §6.2; Review 147 B-147-4):
//
//   - every item's reference is the SHA-256 of its line;
//   - every derived item names at least one parent, and each parent
//     resolves to an earlier item with that name and SHA-256;
//   - every parent of a configuration's derived item belongs to the same
//     configuration;
//   - each kind of derived item names the parents it was computed from:
//     the closure the third call; a Context the second call; the
//     consumption record the third call and every raw I-4 record of the
//     configuration; the findings the consumption record and the third
//     call; the comparison the findings; the checks record the repository
//     files, the toolchain and every call of the configuration.
//
// It returns every problem; none means the chain resolves. It checks the
// association of names and digests, not the content's meaning.
func VerifyProvenance(items []Item) []string {
	var problems []string
	seen := map[string]string{}
	var order []string
	for i, it := range items {
		line := bytes.TrimSuffix(it.Line, []byte("\n"))
		if got := sha(line); got != it.Ref.SHA256 {
			problems = append(problems, fmt.Sprintf("item %d %s: reference %s is not the line's SHA-256 %s", i, it.Ref.Name, it.Ref.SHA256, got))
		}
		if it.File == "derived" {
			var body struct {
				Name    string `json:"name"`
				Parents []Ref  `json:"parents"`
			}
			if err := json.Unmarshal(line, &body); err != nil {
				problems = append(problems, fmt.Sprintf("item %d: not a record: %v", i, err))
				continue
			}
			problems = append(problems, checkDerived(body.Name, body.Parents, seen, order)...)
		}
		seen[it.Ref.Name] = sha(line)
		order = append(order, it.Ref.Name)
	}
	return problems
}

func checkDerived(name string, parents []Ref, seen map[string]string, order []string) []string {
	var out []string
	if len(parents) == 0 {
		return []string{name + ": no parent"}
	}
	have := map[string]bool{}
	for _, p := range parents {
		have[p.Name] = true
		got, ok := seen[p.Name]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s: parent %s is not an earlier record", name, p.Name))
		case got != p.SHA256:
			out = append(out, fmt.Sprintf("%s: parent %s digest %s, the record is %s", name, p.Name, p.SHA256, got))
		}
	}
	cfg, kind, ok := splitConfig(name)
	if !ok {
		return out
	}
	for _, p := range parents {
		if pc, _, pok := splitConfig(p.Name); !pok || pc != cfg {
			out = append(out, fmt.Sprintf("%s: parent %s belongs to another configuration", name, p.Name))
		}
	}
	var required []string
	earlier := func(prefix string) []string {
		var xs []string
		for _, n := range order {
			if strings.HasPrefix(n, prefix) {
				xs = append(xs, n)
			}
		}
		return xs
	}
	switch {
	case kind == "closure":
		required = []string{cfg + "/call3-list"}
	case strings.HasPrefix(kind, "context/"):
		required = []string{cfg + "/call2-context"}
	case kind == "consumption":
		required = append([]string{cfg + "/call3-list"}, earlier(cfg+"/I-4/")...)
	case kind == "findings":
		required = []string{cfg + "/consumption", cfg + "/call3-list"}
	case kind == "comparison":
		required = []string{cfg + "/findings"}
	case kind == "checks":
		required = []string{cfg + "/repo-files", cfg + "/toolchain"}
		required = append(required, earlier(cfg+"/call")...)
	}
	sort.Strings(required)
	for _, r := range required {
		if !have[r] {
			out = append(out, fmt.Sprintf("%s: required parent %s is missing", name, r))
		}
	}
	return out
}

// splitConfig splits "<goos>/<goarch>/<kind>" names.
func splitConfig(name string) (cfg, kind string, ok bool) {
	parts := strings.SplitN(name, "/", 3)
	if len(parts) != 3 {
		return "", "", false
	}
	switch parts[0] {
	case "linux", "darwin", "windows":
	default:
		return "", "", false
	}
	return parts[0] + "/" + parts[1], parts[2], true
}
