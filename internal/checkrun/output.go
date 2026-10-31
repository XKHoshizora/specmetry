package checkrun

import (
	"io"
	"os"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// streams is step 9 (spec §9.1, §19.9; D20 (i), C-7): the objects a
// check's output goes to, and whether this batch can carry it. Without
// --json the check's stdout and stderr go to the raw stdout and stderr;
// with --json both go to the raw stderr, and stdout carries Specmetry's
// own document only. A destination that is an *os.File is inherited by
// the check. The controlled in-memory destination is a later batch's
// (plan §11.11 item 2); until then any other Writer is refused before a
// check starts (internal, exit 1).
func streams(out Output, asJSON bool) (stdout, stderr *os.File, err error) {
	type dest struct {
		name string
		w    io.Writer
	}
	dests := []dest{{"stdout", out.Stdout}, {"stderr", out.Stderr}}
	if asJSON {
		dests = dests[1:]
	}
	var files []*os.File
	var problems []apperr.Detail
	for _, d := range dests {
		f, ok := d.w.(*os.File)
		if !ok || f == nil {
			problems = append(problems, apperr.Detail{Pointer: d.name, Problem: "is not a file a check can inherit"})
			continue
		}
		files = append(files, f)
	}
	if len(problems) > 0 {
		return nil, nil, apperr.New(apperr.Internal, "the output cannot carry check output; no check was run").WithDetails(problems...)
	}
	if asJSON {
		return files[0], files[0], nil
	}
	return files[0], files[1], nil
}
