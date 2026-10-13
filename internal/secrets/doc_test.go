package secrets

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The built-in rules are the approved TD-21 rule set: their IDs and
// patterns must stay identical to the table in docs/architecture.md §10.8.
func TestBuiltinRulesMatchTheArchitectureTable(t *testing.T) {
	raw, err := os.ReadFile("../../docs/architecture.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	start := strings.Index(doc, "### 10.8 ")
	end := strings.Index(doc, "## 11. ")
	if start < 0 || end < start {
		t.Fatal("architecture §10.8 not found")
	}
	cell := regexp.MustCompile(`(?:\\\||[^|])+`)
	documented := map[string]string{}
	var order []string
	for _, line := range strings.Split(doc[start:end], "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := cell.FindAllString(strings.Trim(line, "|"), -1)
		if len(cells) != 4 {
			continue
		}
		id := strings.Trim(strings.TrimSpace(cells[0]), "`")
		pattern := strings.TrimSpace(cells[2])
		if !strings.HasPrefix(pattern, "`") {
			continue
		}
		documented[id] = strings.ReplaceAll(strings.Trim(pattern, "`"), `\|`, "|")
		order = append(order, id)
	}
	if strings.Join(order, ",") != strings.Join(BuiltinRules(), ",") {
		t.Fatalf("documented rules %v, implemented %v", order, BuiltinRules())
	}
	for _, r := range builtin {
		if documented[r.id] != r.re.String() {
			t.Errorf("%s: documented %q, implemented %q", r.id, documented[r.id], r.re.String())
		}
	}
}
