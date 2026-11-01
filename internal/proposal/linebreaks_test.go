package proposal_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/proposal"
)

// WP2.5 D3 and D4 on the propose path (spec §5.6, §5.7; MVP-EV-21,
// CX-29): the check an add item writes, and the check an update item
// gives, are refused at their command when a cmd command holds CR or LF or
// an sh, bash or shell-less command holds CR; an sh command with LF line
// breaks, a pwsh command and an update without a check pass. The problem
// never quotes the command.
func TestCheckStaticRefusesLineBreaks(t *testing.T) {
	const canary = "CANARY-5c21"
	raw := `{"origin":"agent_extracted","base_version":1,"summary":"checks","items":[
 {"op":"add","type":"criterion","ref":"a","data":{"requirement":"REQ-001","statement":"s","verification":"automated",
  "check":{"command":"echo one\necho ` + canary + `","shell":"cmd","rationale":"r"}}},
 {"op":"add","type":"criterion","ref":"b","data":{"requirement":"REQ-001","statement":"s","verification":"automated",
  "check":{"command":"echo one\necho two","rationale":"r"}}},
 {"op":"update","type":"criterion","target":"AC-001-1","target_revision":1,"change_class":"material",
  "data":{"check":{"command":"echo 'one\r'","rationale":"r"}}},
 {"op":"update","type":"criterion","target":"AC-001-2","target_revision":1,"change_class":"material",
  "data":{"check":{"command":"echo one\r\necho two","shell":"pwsh","rationale":"r"}}},
 {"op":"update","type":"criterion","target":"AC-001-3","target_revision":1,"change_class":"editorial",
  "data":{"statement":"the saved check is not read again"}},
 {"op":"add","type":"criterion","ref":"c","data":{"requirement":"REQ-001","statement":"s","verification":"automated",
  "check":{"command":"make\r","shell":"bash","rationale":"r"}}}
]}`
	p, err := proposal.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	e := apperr.From(proposal.CheckStatic(doc, p))
	if e == nil || e.Code != apperr.InvalidInput {
		t.Fatalf("CheckStatic = %v; want invalid input", e)
	}
	var pointers []string
	for _, d := range e.Details {
		pointers = append(pointers, d.Pointer)
		if strings.Contains(d.Problem, canary) || strings.Contains(d.Problem, "echo") {
			t.Errorf("%s quotes the command: %q", d.Pointer, d.Problem)
		}
	}
	want := []string{"/items/0/data/check/command", "/items/2/data/check/command", "/items/5/data/check/command"}
	if !slices.Equal(pointers, want) {
		t.Fatalf("problems at %v; want %v", pointers, want)
	}
}

// The same rule on data a reviewer edited (review accepted_with_edits):
// the edited check is refused at its command; an edit with LF line breaks
// in an sh check, or without a check, passes.
func TestCheckEditedDataRefusesLineBreaks(t *testing.T) {
	data := func(s string) map[string]json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	cases := []struct {
		data string
		want []string
	}{
		{`{"check":{"command":"dir\r\ndir","shell":"cmd","rationale":"r"}}`, []string{"/decisions/0/data/check/command"}},
		{`{"check":{"command":"ls\r","shell":"sh","rationale":"r"}}`, []string{"/decisions/0/data/check/command"}},
		{`{"check":{"command":"ls\nls","shell":"sh","rationale":"r"}}`, nil},
		{`{"statement":"no check"}`, nil},
	}
	for _, c := range cases {
		var got []string
		for _, d := range proposal.CheckEditedData(data(c.data), "/decisions/0/data") {
			got = append(got, d.Pointer)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: problems at %v; want %v", c.data, got, c.want)
		}
	}
}
