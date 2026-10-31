package checkrun

// Observed is the evidence grade of every run the MVP records (data-model
// §4.11.1), Off its isolation mode.
const (
	Observed = "observed"
	Off      = "off"
)

// Qualifying is spec §7.4 (data-model §4.11.1) for a written row: the same
// non-NULL HEAD at both ends, clean at both ends, no context change, and
// (in the MVP) the observed grade. Neither the result nor the exit code
// decides it: a timeout, spawn_error, input_missing or nonzero_exit run
// qualifies when its context is consistent. It is the formula of the derive
// engine's qualifying, written again from the text; the tests compare the
// two.
func Qualifying(r Row) bool {
	return r.HeadStart != "" && r.HeadStart == r.HeadEnd &&
		r.TreeStateStart == Clean && r.TreeStateEnd == Clean &&
		r.FailureKind != ContextChanged && r.EvidenceGrade == Observed
}

// Qualifying is spec §7.4 for the row.
func (r Row) Qualifying() bool { return Qualifying(r) }

// TargetSnapshot is the batch summary is_target_snapshot (spec §11.2, D9):
// with runs, every run's head_start equals the target snapshot s; with none
// (D18), the HEAD at the start of the command does. No snapshot ("") and a
// NULL head ("") are never equal. It never decides whether a run counts.
func TargetSnapshot(s string, heads []string, commandHead string) bool {
	if s == "" {
		return false
	}
	if len(heads) == 0 {
		return commandHead == s
	}
	for _, h := range heads {
		if h != s {
			return false
		}
	}
	return true
}
