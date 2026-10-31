package checkrun_test

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// row is a written row with the fields §7.4 reads; the rest stay zero.
func row(headStart, headEnd, treeStart, treeEnd, result, kind, grade string) checkrun.Row {
	return checkrun.Row{
		Run: checkrun.Run{
			HeadStart: headStart, HeadEnd: headEnd, TreeStateStart: treeStart, TreeStateEnd: treeEnd, Result: result, FailureKind: kind,
		},
		EvidenceGrade: grade,
	}
}

// WP2.5 spec §7.4 (data-model §4.11.1): head_start = head_end ≠ NULL, clean
// at both ends, failure_kind ≠ context_changed, and (in the MVP) the
// observed grade. The result and the exit code do not decide it: timeout,
// spawn_error, input_missing and nonzero_exit runs qualify when their
// context is consistent.
func TestQualifying(t *testing.T) {
	const observed = checkrun.Observed
	for _, tc := range []struct {
		name string
		r    checkrun.Row
		want bool
	}{
		{"pass, one HEAD, clean at both ends", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Pass, "", observed), true},
		{"nonzero_exit qualifies", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Fail, checkrun.NonzeroExit, observed), true},
		{"timeout qualifies", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Error, checkrun.Timeout, observed), true},
		{"spawn_error qualifies", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Error, checkrun.SpawnError, observed), true},
		{"input_missing qualifies", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Error, checkrun.InputMissing, observed), true},
		{"context_changed never qualifies", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Error, checkrun.ContextChanged, observed), false},
		{"another HEAD at the end", row(headA, headB, checkrun.Clean, checkrun.Clean, checkrun.Pass, "", observed), false},
		{"no HEAD at the end", row(headA, "", checkrun.Clean, checkrun.Clean, checkrun.Pass, "", observed), false},
		{"no HEAD at either end", row("", "", checkrun.Clean, checkrun.Clean, checkrun.Pass, "", observed), false},
		{"dirty at the start", row(headA, headA, checkrun.Dirty, checkrun.Clean, checkrun.Pass, "", observed), false},
		{"unknown at the start", row(headA, headA, checkrun.Unknown, checkrun.Clean, checkrun.Pass, "", observed), false},
		{"dirty at the end", row(headA, headA, checkrun.Clean, checkrun.Dirty, checkrun.Pass, "", observed), false},
		{"unknown at the end", row(headA, headA, checkrun.Clean, checkrun.Unknown, checkrun.Pass, "", observed), false},
		{"not observed (only observed in the MVP)", row(headA, headA, checkrun.Clean, checkrun.Clean, checkrun.Pass, "", "snapshot_isolated"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkrun.Qualifying(tc.r); got != tc.want {
				t.Errorf("Qualifying = %v; want %v", got, tc.want)
			}
			if got := tc.r.Qualifying(); got != tc.want {
				t.Errorf("Row.Qualifying = %v; want %v", got, tc.want)
			}
		})
	}
}

// WP2.5 spec §11.2 (D9): is_target_snapshot is a batch summary. With runs
// it is true when every run's head_start equals the target snapshot S;
// with none (D18), when the HEAD at the start of the command does. No S, or
// a NULL head_start, is never equal. The mixed example of §11.2 is false,
// though its first run still counts for its criterion.
func TestTargetSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name        string
		s           string
		heads       []string
		commandHead string
		want        bool
	}{
		{"every run on S", headA, []string{headA, headA}, headB, true},
		{"one run elsewhere (the mixed example)", headA, []string{headA, headB}, headA, false},
		{"one run with a NULL head_start", headA, []string{headA, ""}, headA, false},
		{"no S", "", []string{headA}, headA, false},
		{"no S and NULL heads", "", []string{""}, "", false},
		{"an empty batch on S", headA, nil, headA, true},
		{"an empty batch elsewhere", headA, nil, headB, false},
		{"an empty batch with no commit", headA, nil, "", false},
		{"an empty batch with no S", "", nil, headA, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkrun.TargetSnapshot(tc.s, tc.heads, tc.commandHead); got != tc.want {
				t.Errorf("TargetSnapshot = %v; want %v", got, tc.want)
			}
		})
	}
}
