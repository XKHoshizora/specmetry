package gitscan

import (
	"slices"
	"testing"
)

// WP2.3 spec §5.3 and §9 M1 to M11 (TD-17, MVP-EV-14): a mention is a
// whole ASCII word ([A-Za-z0-9_-]+) that is exactly a requirement or a
// criterion key; the format is not loosened and other kinds are not
// linked; a key in a Specmetry-Ref trailer value is a trailer mention,
// and a trailer line is part of the message too, so it never also counts
// as a message mention.
func TestMentions(t *testing.T) {
	type m = Mention
	req := func(key, source string) m { return m{Key: key, Type: TargetRequirement, Source: source} }
	ac := func(key, source string) m { return m{Key: key, Type: TargetCriterion, Source: source} }
	for _, c := range []struct {
		name, trailers, message string
		want                    []m
	}{
		{"M1 a subject", "", "fix REQ-001\n", []m{req("REQ-001", SourceMessage)}},
		{"M2 a trailer", "REQ-010, AC-010-1", "x\n\nSpecmetry-Ref: REQ-010, AC-010-1\n",
			[]m{ac("AC-010-1", SourceTrailer), req("REQ-010", SourceTrailer)}},
		{"M3 trailer keys in any case", "REQ-011 REQ-012,REQ-013", "x\n\nspecmetry-ref: REQ-011 REQ-012\nSPECMETRY-REF:REQ-013\n",
			[]m{req("REQ-011", SourceTrailer), req("REQ-012", SourceTrailer), req("REQ-013", SourceTrailer)}},
		{"M4 not the last paragraph: not a trailer", "", "x\n\nSpecmetry-Ref: REQ-030\n\nmore text\n", []m{req("REQ-030", SourceMessage)}},
		{"M5 in the message and a trailer", "REQ-010", "fix REQ-010\n\nSpecmetry-Ref: REQ-010\n", []m{req("REQ-010", SourceTrailer)}},
		{"M6 not keys", "", "XREQ-002 REQ-0003x REQ-005-fix req-006 AC-007-0 REQ-1 AC-001 AC-01-1 REQ_007 REQ-008_x\n", nil},
		{"M7 punctuation and other scripts are boundaries", "", "(REQ-004) REQ-004. 完成REQ-060了\n", []m{req("REQ-004", SourceMessage), req("REQ-060", SourceMessage)}},
		{"M8 trailer values split into words", "AC-040-1; REQ-041", "x\n\nSpecmetry-Ref: AC-040-1; REQ-041\n",
			[]m{ac("AC-040-1", SourceTrailer), req("REQ-041", SourceTrailer)}},
		{"M9 other kinds", "GOAL-1", "Q-001 GOAL-1\n\nSpecmetry-Ref: GOAL-1\n", nil},
		{"M10 any key, known or not (resolution is the write's)", "", "REQ-999\n", []m{req("REQ-999", SourceMessage)}},
		{"CR and tabs", "", "fix\tREQ-100\r\nAC-100-12\r\n", []m{ac("AC-100-12", SourceMessage), req("REQ-100", SourceMessage)}},
		{"long numbers", "", "REQ-0001 REQ-123456 AC-1234-10\n", []m{ac("AC-1234-10", SourceMessage), req("REQ-0001", SourceMessage), req("REQ-123456", SourceMessage)}},
		{"a trailer value naming what the message does not", "REQ-200", "subject\n", []m{req("REQ-200", SourceTrailer)}},
		{"empty", "", "", nil},
		{"the same key twice", "", "REQ-001 and REQ-001\n", []m{req("REQ-001", SourceMessage)}},
	} {
		got := Mentions(c.trailers, c.message)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: Mentions = %v; want %v", c.name, got, c.want)
		}
	}
}
