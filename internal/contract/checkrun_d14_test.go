package contract

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// WP2.5 D14 (spec §19.10, frozen 2026-10-31): the conditional constraints
// check-run.v1 puts on each run, one fixed expectation per constraint and
// per allowed edge, written from the frozen text. A refused run must be
// refused at the field the constraint is about; an allowed edge must pass.
// The edges are part of D14: context_changed with exit code 0, a signal's
// fail with NULL, not_found still read, an error run that qualifies, and
// no exit code bound (no global maximum of 255).
func TestCheckRunD14(t *testing.T) {
	v := validator(t)
	run := func(edit func(r map[string]any)) []byte {
		r := map[string]any{
			"criterion": "AC-001-1", "criterion_revision": 1, "evidence_id": "01J9Z3K5Q8R2T4V6W8X0Y2Z4B1", "seq": 405,
			"result": "pass", "failure_kind": nil, "exit_code": 0, "duration_ms": 8421,
			"head_start": "8f3c2a1b9d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a", "tree_state_start": "clean",
			"evidence_grade": "observed", "qualifying": true,
		}
		edit(r)
		b, err := json.Marshal(map[string]any{
			"schema": "specmetry.check-run/v1", "worktree_id": "01J9Z3K5Q8R2T4V6W8X0Y2Z4D1", "is_target_snapshot": true,
			"isolation": "off", "runs": []any{r},
		})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	set := func(kv ...any) func(r map[string]any) {
		return func(r map[string]any) {
			for i := 0; i < len(kv); i += 2 {
				r[kv[i].(string)] = kv[i+1]
			}
		}
	}
	for _, tc := range []struct {
		name string
		edit func(r map[string]any)
		// at is the field the run is refused at; "" means it is valid.
		at string
	}{
		// pass ⇒ failure_kind NULL and exit_code 0.
		{"pass, NULL, 0", set(), ""},
		{"pass with a failure kind", set("failure_kind", "nonzero_exit"), "/runs/0/failure_kind"},
		{"pass with exit code 1", set("exit_code", 1), "/runs/0/exit_code"},
		{"pass with a NULL exit code", set("exit_code", nil), "/runs/0/exit_code"},
		// Review 089: pass + timeout + exit code 0.
		{"pass, timeout, 0", set("failure_kind", "timeout"), "/runs/0/failure_kind"},
		// fail ⇒ nonzero_exit, and an exit code that is not 0 (NULL allowed).
		{"fail, nonzero_exit, 1", set("result", "fail", "failure_kind", "nonzero_exit", "exit_code", 1), ""},
		{"fail, nonzero_exit, NULL (a Unix signal, D12)", set("result", "fail", "failure_kind", "nonzero_exit", "exit_code", nil), ""},
		{"fail with a Windows code above 255 (D17)", set("result", "fail", "failure_kind", "nonzero_exit", "exit_code", int64(3221225477)), ""},
		{"fail with exit code 0", set("result", "fail", "failure_kind", "nonzero_exit", "exit_code", 0), "/runs/0/exit_code"},
		{"fail with no failure kind", set("result", "fail", "exit_code", 1), "/runs/0/failure_kind"},
		{"fail as a timeout", set("result", "fail", "failure_kind", "timeout", "exit_code", nil), "/runs/0/failure_kind"},
		// error ⇒ one of timeout, spawn_error, input_missing, context_changed,
		// not_found (read only).
		{"error with no failure kind", set("result", "error", "exit_code", nil), "/runs/0/failure_kind"},
		{"error as nonzero_exit", set("result", "error", "failure_kind", "nonzero_exit", "exit_code", 1), "/runs/0/failure_kind"},
		{"error, not_found, read compatibility (D5)", set("result", "error", "failure_kind", "not_found", "exit_code", nil), ""},
		// timeout, spawn_error, input_missing ⇒ exit_code NULL.
		{"timeout, NULL, qualifying", set("result", "error", "failure_kind", "timeout", "exit_code", nil), ""},
		{"timeout with exit code 0", set("result", "error", "failure_kind", "timeout", "exit_code", 0), "/runs/0/exit_code"},
		{"spawn_error with exit code 1", set("result", "error", "failure_kind", "spawn_error", "exit_code", 1, "duration_ms", 0), "/runs/0/exit_code"},
		{"input_missing with exit code 0", set("result", "error", "failure_kind", "input_missing", "exit_code", 0, "duration_ms", 0), "/runs/0/exit_code"},
		// spawn_error, input_missing ⇒ duration_ms 0.
		{"spawn_error, NULL, 0 ms, qualifying", set("result", "error", "failure_kind", "spawn_error", "exit_code", nil, "duration_ms", 0), ""},
		{"input_missing, NULL, 0 ms, qualifying", set("result", "error", "failure_kind", "input_missing", "exit_code", nil, "duration_ms", 0), ""},
		{"spawn_error that lasted", set("result", "error", "failure_kind", "spawn_error", "exit_code", nil, "duration_ms", 5), "/runs/0/duration_ms"},
		{"input_missing that lasted", set("result", "error", "failure_kind", "input_missing", "exit_code", nil, "duration_ms", 1), "/runs/0/duration_ms"},
		// context_changed: any exit code, 0 included; never qualifying.
		{"context_changed with exit code 0", set("result", "error", "failure_kind", "context_changed", "exit_code", 0, "qualifying", false), ""},
		{"context_changed with NULL", set("result", "error", "failure_kind", "context_changed", "exit_code", nil, "qualifying", false), ""},
		{"context_changed with exit code 256", set("result", "error", "failure_kind", "context_changed", "exit_code", 256, "qualifying", false), ""},
		{"context_changed that qualifies", set("result", "error", "failure_kind", "context_changed", "exit_code", 0), "/runs/0/qualifying"},
		// qualifying ⇒ a head_start and a clean start.
		{"qualifying with a NULL head_start", set("head_start", nil), "/runs/0/head_start"},
		{"qualifying from a dirty start", set("tree_state_start", "dirty"), "/runs/0/tree_state_start"},
		{"qualifying from an unknown start", set("tree_state_start", "unknown"), "/runs/0/tree_state_start"},
		{"not qualifying with a NULL head_start", set("head_start", nil, "qualifying", false), ""},
		{"not qualifying from a dirty start", set("tree_state_start", "dirty", "qualifying", false), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateJSON("check-run.v1", run(tc.edit))
			if tc.at == "" {
				if err != nil {
					t.Fatalf("refused: %v %+v", err, apperr.From(err).Details)
				}
				return
			}
			if err == nil {
				t.Fatalf("accepted; want refused at %s", tc.at)
			}
			var at []string
			for _, d := range apperr.From(err).Details {
				at = append(at, d.Pointer)
			}
			if !slices.Contains(at, tc.at) {
				t.Errorf("refused at %v; want %s", at, tc.at)
			}
		})
	}
}
