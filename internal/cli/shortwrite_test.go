package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
)

// halfWriter takes half of each write and reports that, with no error, as
// a writer to a terminal that was cut off may.
type halfWriter struct{ got bytes.Buffer }

func (w *halfWriter) Write(p []byte) (int, error) {
	n := len(p) / 2
	w.got.Write(p[:n])
	return n, nil
}

// readCounter counts the reads made from it.
type readCounter struct {
	r     io.Reader
	reads int
}

func (c *readCounter) Read(p []byte) (int, error) {
	c.reads++
	return c.r.Read(p)
}

// codex-review-031 B1, through the real command entry: a person's
// confirmation that cannot be shown whole is no confirmation (architecture
// §5.1). With stdout taking half of each write and no error, and the right
// confirmation already waiting, question answer is exit 6, reads nothing,
// and writes nothing: Q-001 stays open, no seq is taken, no other fact
// changes. Written whole, the same command answers it.
func TestConfirmationNotShownWholeWritesNothing(t *testing.T) {
	home, dir, _ := questionsProject(t)
	args := []string{"question", "answer", "Q-001", "-m", "review031-confirmation"}
	before := closeState(t, home)
	t.Chdir(dir)
	out := &halfWriter{}
	var errOut bytes.Buffer
	root := newRoot(out, &errOut)
	in := &readCounter{r: strings.NewReader("Q-001\n")}
	root.SetIn(in)
	code := executeAs(context.Background(), channel.Detection{Channel: channel.HumanCLI}, root, args, &errOut)
	if code != apperr.ExitEnvironment || in.reads != 0 || !strings.Contains(errOut.String(), "cannot show the confirmation") {
		t.Fatalf("exit %d after %d confirmation reads, want 6 and none\n%s", code, in.reads, errOut.String())
	}
	if out.got.Len() == 0 {
		t.Fatal("nothing reached stdout, so the fault did not happen")
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
	got := rows(t, home, `SELECT status, coalesce(answered_seq, '-') FROM questions WHERE key = 'Q-001'`)
	if want := [][]string{{"open", "-"}}; !equalRows(got, want) {
		t.Fatalf("Q-001 = %v, want %v", got, want)
	}

	var whole bytes.Buffer
	root = newRoot(&whole, &errOut)
	in = &readCounter{r: strings.NewReader("Q-001\n")}
	root.SetIn(in)
	if code := executeAs(context.Background(), channel.Detection{Channel: channel.HumanCLI}, root, args, &errOut); code != apperr.ExitOK {
		t.Fatalf("written whole: exit %d\n%s%s", code, whole.String(), errOut.String())
	}
	if got := rows(t, home, `SELECT status FROM questions WHERE key = 'Q-001'`); !equalRows(got, [][]string{{"answered"}}) {
		t.Fatalf("Q-001 = %v after a whole confirmation", got)
	}
}
