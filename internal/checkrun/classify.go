// Package checkrun is the observed Check Runner of WP2.5
// (docs/wp2.5-check-runner.md, frozen on 2026-10-31). The first batch of
// its production plan (docs/wp2.5-implementation-plan.md §4.1) holds what
// needs no process: the classification of a run (spec §6.2), the context
// change (§7.2), the time and exit code fields (§10.1), the qualifying
// formula (§7.4), the batch summary (§11.2, D9) and the write of one
// check_runs row (§10.3). Nothing here starts a process, and no command
// uses it yet.
package checkrun

import (
	"errors"
	"fmt"
)

// Tree states (data-model §4.11; spec §7.1).
const (
	Clean   = "clean"
	Dirty   = "dirty"
	Unknown = "unknown"
)

// Results and failure kinds (spec §6.2). The observed runner never
// produces not_found (D5).
const (
	Pass  = "pass"
	Fail  = "fail"
	Error = "error"

	NonzeroExit    = "nonzero_exit"
	Timeout        = "timeout"
	SpawnError     = "spawn_error"
	InputMissing   = "input_missing"
	ContextChanged = "context_changed"
)

// Context is a code context (spec §7.1): the HEAD, "" when it is NULL, and
// the tree state.
type Context struct {
	Head      string
	TreeState string
}

// Changed is the context change of spec §7.2 (data-model §7.4 step 4):
// another HEAD at the end, one of the two NULL included, or clean at the
// start and not clean at the end. clean to unknown counts: it does not
// prove a change, only that no change can be ruled out.
func Changed(start, end Context) bool {
	return start.Head != end.Head || (start.TreeState == Clean && end.TreeState != Clean)
}

// Launch is how a run's process started (spec §10.1, the event
// dictionary). It is decided at the start and never changes.
type Launch int

const (
	// NotStarted: no process; the input was missing (L2) or the process
	// could not be created.
	NotStarted Launch = iota
	// Started is a normal start: Unix Start returned, Windows ResumeThread
	// returned 1 (event 1, t_start).
	Started
	// ResumeAnomaly is a Windows resume with an abnormal result:
	// ResumeThread returned -1, 0 or more than 1, or the thread snapshot
	// showed more than one thread (event 2, t_anom; spec §19.7). The user
	// code may have run.
	ResumeAnomaly
)

// Exit is the main process's exit status when the Runner read it reliably
// (event 4). The zero value is "not obtained".
type Exit struct {
	Obtained bool
	// Signaled is a Unix process ended by a signal: it has no exit code
	// (D12).
	Signaled bool
	// Code is the exit code read: on Windows as it is (D17), on macOS's
	// waitid path the low 8 bits of si_status (Q-3 (a)).
	Code int64
}

// Observation is what the Runner knows of one run when it decides what to
// record.
type Observation struct {
	Launch Launch
	// NotStarted is InputMissing or SpawnError for a run that started no
	// process, "" otherwise.
	NotStarted string
	// TreeNotConfirmed: S5 did not confirm within T_kill that the process
	// tree ended (F1, D19). Only a started process has a tree.
	TreeNotConfirmed bool
	// Canceled: a cancellation took effect before the write transaction
	// began (D8).
	Canceled bool
	// TimedOut: the timeout was the event S2 woke for first.
	TimedOut bool
	Exit     Exit
	Start    Context
	// End is the end context as read; nil when it is not read: the tree
	// was not confirmed ended (D19), or the resume was abnormal (A12).
	End *Context
	// EndUnreadable: reading the end context failed (D6). End keeps what
	// was read, a NULL HEAD and unknown for the rest.
	EndUnreadable bool
}

// Case is the row of spec §6.2 (or §8.6 A12) a run falls in.
type Case string

// The cases, by the priority of §6.2.
const (
	CaseTreeNotConfirmed Case = "0a" // D19, A6 included
	CaseCanceled         Case = "0b" // D8: not recorded
	CaseContextChanged   Case = "1"
	CaseEndUnreadable    Case = "1/D6"
	CaseResumeAnomaly    Case = "A12" // Q-2 (a)
	CaseInputMissing     Case = "2"
	CaseSpawnError       Case = "3"
	CaseTimeout          Case = "4"
	CasePass             Case = "5"
	CaseFail             Case = "6"
)

// Outcome is what is recorded for a run.
type Outcome struct {
	Case Case
	// Record is false only for a cancellation before the write (0b).
	Record      bool
	Result      string
	FailureKind string // "" is NULL
	ExitCode    *int64 // nil is NULL
	// End is the end context recorded: head_end ("" NULL) and
	// tree_state_end; NULL and unknown when it was not read.
	End Context
}

// ErrObservation reports an observation or a reading that no lawful
// sequence of the Runner's events produces. It is the caller's bug, never
// a check result.
var ErrObservation = errors.New("checkrun: inconsistent observation")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrObservation, fmt.Sprintf(format, args...))
}

// Classify decides what is recorded for a run, by the rows of spec §6.2 in
// their priority, with the exit code of §10.1 for the row hit, and the end
// context of §10.1 and §7.1.
func Classify(o Observation) (Outcome, error) {
	if err := o.check(); err != nil {
		return Outcome{}, err
	}
	notRead := Context{TreeState: Unknown}
	switch {
	case o.TreeNotConfirmed:
		return Outcome{Case: CaseTreeNotConfirmed, Record: true, Result: Error, FailureKind: ContextChanged, ExitCode: o.Exit.code(), End: notRead}, nil
	case o.Canceled:
		return Outcome{Case: CaseCanceled}, nil
	case o.Launch == ResumeAnomaly:
		return Outcome{Case: CaseResumeAnomaly, Record: true, Result: Error, FailureKind: ContextChanged, ExitCode: o.Exit.code(), End: notRead}, nil
	}
	end := *o.End
	switch {
	case o.EndUnreadable:
		return Outcome{Case: CaseEndUnreadable, Record: true, Result: Error, FailureKind: ContextChanged, ExitCode: o.Exit.code(), End: end}, nil
	case Changed(o.Start, end):
		return Outcome{Case: CaseContextChanged, Record: true, Result: Error, FailureKind: ContextChanged, ExitCode: o.Exit.code(), End: end}, nil
	case o.NotStarted == InputMissing:
		return Outcome{Case: CaseInputMissing, Record: true, Result: Error, FailureKind: InputMissing, End: end}, nil
	case o.NotStarted == SpawnError:
		return Outcome{Case: CaseSpawnError, Record: true, Result: Error, FailureKind: SpawnError, End: end}, nil
	case o.TimedOut:
		return Outcome{Case: CaseTimeout, Record: true, Result: Error, FailureKind: Timeout, End: end}, nil
	case o.Exit.Code == 0 && !o.Exit.Signaled:
		zero := int64(0)
		return Outcome{Case: CasePass, Record: true, Result: Pass, ExitCode: &zero, End: end}, nil
	default:
		return Outcome{Case: CaseFail, Record: true, Result: Fail, FailureKind: NonzeroExit, ExitCode: o.Exit.code(), End: end}, nil
	}
}

// code is the exit code a row keeps (§10.1): the code reliably read, NULL
// for a signal or when none was read.
func (e Exit) code() *int64 {
	if !e.Obtained || e.Signaled {
		return nil
	}
	c := e.Code
	return &c
}

func (o Observation) check() error {
	switch e := o.Exit; {
	case e.Signaled && !e.Obtained:
		return invalid("a signal with no exit status read")
	case e.Signaled && e.Code != 0, !e.Obtained && e.Code != 0:
		return invalid("an exit code with no exit read")
	}
	if !validTree(o.Start.TreeState) {
		return invalid("start tree state %q", o.Start.TreeState)
	}
	switch o.Launch {
	case NotStarted:
		switch {
		case o.NotStarted != InputMissing && o.NotStarted != SpawnError:
			return invalid("no process, and no reason %q", o.NotStarted)
		case o.TreeNotConfirmed, o.TimedOut, o.Exit.Obtained:
			return invalid("no process, but a process tree, a timeout or an exit")
		}
	case Started, ResumeAnomaly:
		if o.NotStarted != "" {
			return invalid("a started process with no-process reason %q", o.NotStarted)
		}
		// S2 woke for the main process's exit, so its status was read.
		if o.Launch == Started && !o.TreeNotConfirmed && !o.TimedOut && !o.Canceled && !o.Exit.Obtained {
			return invalid("S2 woke for neither an exit, a timeout nor a cancellation")
		}
	default:
		return invalid("launch %d", o.Launch)
	}
	read := !o.TreeNotConfirmed && o.Launch != ResumeAnomaly
	switch {
	case o.Canceled && !o.TreeNotConfirmed:
		// Not recorded: whether the end was read does not matter.
	case read && o.End == nil:
		return invalid("the end context is read once the tree is confirmed ended")
	case !read && (o.End != nil || o.EndUnreadable):
		return invalid("the end context is not read (D19, A12)")
	}
	if o.End != nil && !validTree(o.End.TreeState) {
		return invalid("end tree state %q", o.End.TreeState)
	}
	if o.EndUnreadable && o.End == nil {
		return invalid("an unreadable end context keeps what was read")
	}
	return nil
}

func validTree(s string) bool { return s == Clean || s == Dirty || s == Unknown }
