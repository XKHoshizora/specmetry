package toolctl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Lines renders a published record as canonical display lines. Error texts
// are not in them (only whether a real result is nil); the not-admitted and
// cleanup-unknown reasons are, since they are the protocol's own fixed
// texts. Nothing is filtered: every admission and skipped action, the Cmd,
// every descriptor with its release and held object, both reads, the reap,
// the pidfd (on Linux), the request, the expiry, the panic, lateness,
// terminality and the item 11 judgement each have a line.
//
// The display lines do not carry an error's source or kind; the structured
// fault set does (Snapshot.Faults, rendered by FaultLines), and a judgement
// must consume it too (Review 154 B-154-4).
func Lines(s Snapshot) []string {
	class := "class " + orDash(s.Class)
	if s.Phase != "" {
		class += " phase " + s.Phase
	}
	out := []string{class, "state " + s.State}
	for _, a := range s.Admissions {
		out = append(out, "admission "+a.Point+" "+admittedText(a.Admitted))
	}
	for _, k := range s.Skipped {
		out = append(out, "skipped "+k)
	}
	switch {
	case !s.StartCalled:
		out = append(out, "start not-initiated")
	case !s.StartReturned:
		out = append(out, "start not-returned")
	case s.StartErr != "":
		out = append(out, "start error")
	default:
		out = append(out, "start ok")
	}
	out = append(out, "cmd held "+yesNo(s.CmdHeld))
	for i, d := range s.Desc {
		l := DescNames[i] + " " + d.State
		if d.State == DescClosed {
			l += " " + okOrError(d.CloseErr)
		}
		obj := "none"
		if d.Held {
			obj = "held"
		}
		out = append(out, l+" release "+d.Release+" object "+obj)
		for range d.Injected {
			out = append(out, DescNames[i]+" injected")
		}
	}
	for i, o := range s.Out {
		var l string
		switch {
		case !o.Started:
			l = "not-started"
		case !o.Done:
			l = "running"
		case o.Overflow:
			l = "overflow"
		case o.ReadErr != "":
			l = "read-error"
		case o.EOF:
			l = "eof"
		default:
			l = "stopped"
		}
		if o.Done {
			sum := sha256.Sum256(o.Data)
			l += fmt.Sprintf(" kept %d sha256 %s", len(o.Data), hex.EncodeToString(sum[:]))
		}
		out = append(out, StreamNames[i]+" "+l)
	}
	switch {
	case !s.WaitCalled:
		out = append(out, "wait not-called")
	case !s.WaitReturned:
		out = append(out, "wait not-returned")
	case s.Exit == nil:
		out = append(out, "wait no-state")
	default:
		out = append(out, fmt.Sprintf("wait exit %d", s.Exit.Code))
	}
	if s.Mode == ModePidfd {
		l := "pidfd " + s.Pidfd.State
		if s.Pidfd.State == PidfdClosed {
			l += " " + okOrError(s.Pidfd.CloseErr)
		}
		out = append(out, fmt.Sprintf("%s closes %d", l, s.Pidfd.Closes))
	}
	l := fmt.Sprintf("request %s %s sends %d", s.Req.Mode, s.Req.State, s.Req.Sends)
	if s.Req.State == ReqDone {
		l += " " + okOrError(s.Req.Err)
	}
	out = append(out, l)
	for _, n := range s.NotAdmit {
		out = append(out, "not-admitted "+n)
	}
	for _, u := range s.Unknown {
		out = append(out, "cleanup-unknown "+u)
	}
	if s.Panic != "" {
		out = append(out, "panic yes")
	} else {
		out = append(out, "panic no")
	}
	for _, x := range []struct {
		name string
		v    bool
	}{{"expired", s.Expired}, {"late", hasLate(s.Events)}, {"terminal", s.Terminal}, {"success", s.Success}} {
		out = append(out, x.name+" "+yesNo(x.v))
	}
	return out
}

// FaultLines renders the structured fault set, one line per fault, in the
// record's order: "fault <site> <source> <kind>", kind given by the
// caller's classification of the error value (Review 154 B-154-4).
func FaultLines(s Snapshot, kind func(error) string) []string {
	out := make([]string, 0, len(s.Faults))
	for _, f := range s.Faults {
		out = append(out, "fault "+f.Site+" "+f.Source+" "+kind(f.Err))
	}
	return out
}

func hasLate(es []Event) bool {
	for _, e := range es {
		if e.Late {
			return true
		}
	}
	return false
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func okOrError(e string) string {
	if e == "" {
		return "ok"
	}
	return "error"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
