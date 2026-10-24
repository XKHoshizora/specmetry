package wp25

import (
	"os"
	"slices"
	"testing"
)

// TestF1Classify: the F1 scenario's classification (Review 084 B-N4) on
// fixed evidence. The real holder's paths are the gate regressions
// (control.f1-*, report.*); these rows add mixes and orders on their own.
// Every row is also classified with its records in the reverse order,
// which must not change the finding.
func TestF1Classify(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	const h = 100 // the holder
	ep, ok := reportLine{"epipe", h}, reportLine{"ok", h}
	answered := func(rs ...reportLine) f1Evidence {
		return f1Evidence{holder: h, asked: true, kept: 2, streams: 2, ended: true, late: []int{5, 7}, records: rs}
	}
	failedTo := func(ev f1Evidence) f1Evidence {
		ev.failures = []string{"report: injected"}
		return ev
	}
	rows := []struct {
		name string
		ev   f1Evidence
		want f1Finding
	}{
		{"epipe", f1Evidence{holder: h, records: f1Records{ep}}, f1Failed},
		{"epipe while waiting for an answer", f1Evidence{holder: h, asked: true, records: f1Records{ep}}, f1Failed},
		{"final answer, ended, late bytes on both streams", answered(ok), f1Wrote},
		{"answer, then epipe", answered(ok, ep), f1Invalid},
		{"epipe, then answer", answered(ep, ok), f1Invalid},
		{"epipe and an answer nobody asked for", f1Evidence{holder: h, records: f1Records{ok, ep}}, f1Invalid},
		{"epipe with a failure", failedTo(f1Evidence{holder: h, records: f1Records{ep}}), f1Invalid},
		{"final answer with a failure", failedTo(answered(ok)), f1Invalid},
		{"both with a failure", failedTo(answered(ok, ep)), f1Invalid},
		{"a failure alone", failedTo(answered()), f1Invalid},
		{"neither record", answered(), f1Invalid},
		{"epipe of another process", f1Evidence{holder: h, records: f1Records{{"epipe", 7}}}, f1Invalid},
		{"answer of another process", answered(reportLine{"ok", 7}), f1Invalid},
		{"two epipe records", f1Evidence{holder: h, records: f1Records{ep, ep}}, f1Invalid},
		{"two answers", answered(ok, ok), f1Invalid},
		{"answer, the observation not ended", func() f1Evidence { ev := answered(ok); ev.ended = false; return ev }(), f1Invalid},
		{"answer, one stream kept", func() f1Evidence { ev := answered(ok); ev.kept, ev.late = 1, []int{5}; return ev }(), f1Invalid},
		{"answer, no late bytes on one stream", func() f1Evidence { ev := answered(ok); ev.late = []int{5, 0}; return ev }(), f1Invalid},
		{"answer nobody asked for, else complete", func() f1Evidence { ev := answered(ok); ev.asked = false; return ev }(), f1Invalid},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			got, why := classifyF1(r.ev)
			if got != r.want || (got == f1Invalid) != (len(why) > 0) {
				t.Errorf("finding %d, want %d (why %q)", got, r.want, why)
			}
			rev := r.ev
			rev.records = slices.Clone(r.ev.records)
			slices.Reverse(rev.records)
			if back, _ := classifyF1(rev); back != got {
				t.Errorf("reversed records: finding %d, in order %d", back, got)
			}
		})
	}
}
