package gitx

import (
	"bytes"
	"context"
	"io"
	"os"
	"slices"
)

// call is one git command of the WP2.3 reads (spec §4.3, §5): its
// arguments after globalArgs, what it may read, its input and the
// variables it adds to the controlled environment. An error names label,
// a fixed word for the command, never one of its arguments, which may hold
// paths such as --git-dir=<path> (codex-review-052 N2).
type call struct {
	label string
	dir   string
	args  []string
	stdin []byte
	env   []string
	// stdout receives git's output as it comes; nil keeps it in the
	// result.
	stdout io.Writer
	// objects: the command may read a git object, so it runs only with a
	// git shown never to fetch one (PD-26). list: it lists an index, so
	// that git must also know ls-files --sparse.
	objects, list bool
	private       *privateDir
}

// result is what a call showed: its output (unless it went to stdout), its
// exit status, and whether git wrote anything to stderr. What git wrote
// there is never kept: it may quote configuration values or paths.
type result struct {
	out    []byte
	code   int
	stderr bool
}

func (c call) run(ctx context.Context) (result, error) {
	var git string
	var err error
	switch {
	case c.list:
		git, err = listGit(ctx)
	case c.objects:
		git, err = objectGit(ctx)
	default:
		git, err = gitToRun(ctx)
	}
	if err != nil {
		return result{}, err
	}
	return c.runWithGit(ctx, git)
}

// runWithGit pins a probed executable for this call; it never rediscovers PATH.
func (c call) runWithGit(ctx context.Context, git string) (result, error) {
	if c.private != nil {
		if err := c.private.beginConsumer(); err != nil {
			return result{}, err
		}
	}
	var out bytes.Buffer
	stdout := c.stdout
	if stdout == nil {
		stdout = &out
	}
	var stdin io.Reader
	if c.stdin != nil {
		stdin = bytes.NewReader(c.stdin)
	}
	var stderr presence
	env := append(filterEnv(os.Environ()), c.env...)
	code, err := commandIO(ctx, git, c.dir, env, stdin, stdout, &stderr, append(slices.Clone(globalArgs), c.args...)...)
	// commandIO waits for the actual Git process even on cancellation; a failed
	// Start has no consumer. Only this live caller can persist that completion.
	if c.private != nil {
		completionErr := c.private.finishConsumer()
		if err == nil {
			err = completionErr
		}
	}
	if err != nil {
		return result{}, err
	}
	return result{out: out.Bytes(), code: code, stderr: stderr.seen}, nil
}

// must runs c and requires exit status 0.
func (c call) must(ctx context.Context) ([]byte, error) {
	r, err := c.run(ctx)
	if err != nil {
		return nil, err
	}
	if r.code != 0 {
		return nil, failed(c.label, r.code)
	}
	return r.out, nil
}

// presence records whether anything was written to it, and keeps nothing.
type presence struct{ seen bool }

func (p *presence) Write(b []byte) (int, error) {
	if len(b) > 0 {
		p.seen = true
	}
	return len(b), nil
}
