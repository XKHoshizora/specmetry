package judge

import (
	"strings"
	"testing"
)

func TestChecksRules(t *testing.T) {
	c := NewChecks([]string{"A4", "B3"})
	c.Enter("A4", "sha")
	c.Hold("A4")
	c.Hold("A4")                     // a second result is a defect; the first stays
	c.Fail("B3")                     // a failure without failures is a defect
	c.Hold("B3")                     // evaluated without an entry record: a defect, kept
	c.Unevaluated("B'1", "not in E") // a result outside E (the not-covered set N) is a defect
	c.Enter("C", "")                 // an entry outside E too
	if r, _ := c.Get("A4"); r.Status != Hold {
		t.Fatalf("A4 %v", r.Status)
	}
	got := strings.Join(c.Defects(), "|")
	for _, want := range []string{"second result for check A4", "fail without failures for check B3",
		"evaluated without entry record: B3", "result for unknown check B'1", "entry for unknown check C"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing defect %q in %q", want, got)
		}
	}
	if !c.Complete() {
		t.Fatal("both checks have results")
	}
}

func TestAccAndSetIfUnset(t *testing.T) {
	var a Acc
	if a.Result().Status != Hold {
		t.Fatal("an empty accumulator holds")
	}
	a.Add(Real(J2, "GOWORK", "B1", "mismatch", ""))
	a.Add(Real(J3, "ToolTags", "B2", "mismatch", ""))
	if r := a.Result(); r.Status != Fail || len(r.Failures) != 2 {
		t.Fatalf("%+v", r)
	}
	c := NewChecks([]string{"A1-a", "A1-b"})
	c.Enter("A1-a", "")
	c.Fail("A1-a", Real(J1, "analyzer", "A1-a", "version", ""))
	c.SetIfUnset("A1-a failed")
	if c.Status("A1-a") != Fail || c.Status("A1-b") != Unevaluated {
		t.Fatal("SetIfUnset must not rewrite a result")
	}
	if len(c.Defects()) != 0 {
		t.Fatalf("defects %v", c.Defects())
	}
}
