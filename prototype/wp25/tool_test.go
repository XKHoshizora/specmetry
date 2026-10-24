package wp25

import (
	"errors"
	"fmt"
	"sync"
)

// toolState is a helper's tool state (Review 086 B-N4-R). Every entry of
// the helper that can make its observation fail goes through it: each
// report operation, the output closes of a holder, the hard limit (its
// timer's callback, or the gate regressions' simulated limit point), and
// the holder's completion. One lock orders them:
//
//   - a failure, whatever entry meets it and whether or not its caller
//     looks at the error, is kept; a later success clears nothing;
//   - the hard limit, once its callback has entered, has failed the tool
//     before the callback does anything else, so no completion can be
//     published from then on;
//   - the completion takes its final state once, holding the lock for its
//     whole publication: it publishes only if no failure came and the
//     limit has not entered, and a failure of its own is kept. A callback
//     entering meanwhile waits for that final state;
//   - once the completion is published, the limit only ends the process,
//     and no report of the observation starts any more.
//
// So one state is never both a published completion and a limit's
// failure. The callback is not stopped by its timer: it decides under the
// lock itself, entered or not, so nothing assumes a Stop joined it.
type toolState struct {
	mu        sync.Mutex
	err       error // the first failure; never cleared
	limit     bool  // the hard limit's callback has entered
	final     bool  // the completion has taken its final state
	published bool  // and the completion was published
}

// tool is this helper process's tool state.
var tool toolState

var (
	errLimit           = errors.New("the helper reached its hard limit")
	errAfterCompletion = errors.New("the completion is published: nothing more belongs to the observation")
)

// failed keeps a failure; only the first is kept, and nothing clears it.
func (t *toolState) failed(err error) {
	if err == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err == nil {
		t.err = err
	}
}

// report runs one report operation of the observation, unless the
// completion is published, and keeps its failure. Only the limit's
// callback runs beside a helper's main path, and it reports only after it
// has won over the completion, so no publication can come between the
// check and the operation.
func (t *toolState) report(op func() error) error {
	t.mu.Lock()
	published := t.published
	t.mu.Unlock()
	if published {
		return errAfterCompletion
	}
	err := op()
	t.failed(err)
	return err
}

// limitEntered is the hard limit's first step: whether the limit wins. If
// the completion was published first, the limit only ends the process;
// otherwise the tool has failed, and no completion can follow.
func (t *toolState) limitEntered() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.published {
		return false
	}
	t.limit = true
	if t.err == nil {
		t.err = errLimit
	}
	return true
}

// complete takes the completion's final state, once: under the lock, it
// runs publish only if no failure came and the limit has not entered, and
// keeps a failure of publish.
func (t *toolState) complete(publish func() error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.final {
		return errors.New("the completion has its final state already")
	}
	t.final = true
	switch {
	case t.limit:
		return errors.New("not completed: the hard limit has entered")
	case t.err != nil:
		return fmt.Errorf("not completed: an earlier failure: %w", t.err)
	}
	if err := publish(); err != nil {
		t.err = err
		return err
	}
	t.published = true
	return nil
}
