package toolctl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Lines renders a published record as the canonical lines the LC-9
// controls compare with their complete expectations. Error texts are not
// in the lines (only whether a real result is nil); the not-admitted and
// cleanup-unknown reasons are, since they are the protocol's own fixed
// texts. Nothing is filtered: every descriptor, both reads, the reap, the
// pidfd (on Linux), the request, the expiry, lateness, terminality and the
// item 11 judgement each have a line.
func Lines(s Snapshot) []string {
	class := "class " + orDash(s.Class)
	if s.Phase != "" {
		class += " phase " + s.Phase
	}
	out := []string{class, "state " + s.State}
	switch {
	case s.StartErr != "":
		out = append(out, "start error")
	case s.State == StStarted || s.State == StWaited:
		out = append(out, "start ok")
	default:
		out = append(out, "start none")
	}
	for i, d := range s.Desc {
		l := DescNames[i] + " " + d.State
		if d.State == DescClosed {
			l += " " + okOrError(d.CloseErr)
		}
		out = append(out, l)
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
	for _, x := range []struct {
		name string
		v    bool
	}{{"expired", s.Expired}, {"late", hasLate(s.Events)}, {"terminal", s.Terminal}, {"success", s.Success}} {
		out = append(out, x.name+" "+yesNo(x.v))
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
