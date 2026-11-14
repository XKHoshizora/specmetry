package job

import (
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
)

// The tests' closeout of a job is one request (Review 151 B-151-1). It is
// registered before its goroutine starts and stays registered until the
// test process ends, so its state is never lost when a wait times out:
//
//   - no request: Finish was never issued;
//   - a request whose done channel is open: issued, not returned;
//   - a request whose done channel is closed: returned, with its original
//     Final.
//
// A closed done shows only that this Finish call returned and its Final
// was published. The record writer's and the diagnostic task's terminal
// states (tasks_test.go) are separate facts, and neither is an observed
// exit of any goroutine (Review 152 N-152-1).
//
// Every caller (finish, finishWant, the asynchronous waits, endTasks and
// the cleanup) uses the same request; none of them issues a second Finish,
// and reading the result consumes nothing: done is closed once and final is
// written once before that.
type finishReq struct {
	done  chan struct{} // closed when Finish returned
	final chain.Final   // the original Final; set before done is closed
}

var finishSeam = struct {
	sync.Mutex
	reqs   map[*Job]*finishReq
	calls  map[*Job]int
	before map[*Job]func()
}{reqs: map[*Job]*finishReq{}, calls: map[*Job]int{}, before: map[*Job]func(){}}

// setFinishHook installs f to run in the request's goroutine right before
// it calls Finish (before the call is counted): a test seam for the Review
// 151 regressions.
func setFinishHook(j *Job, f func()) {
	finishSeam.Lock()
	defer finishSeam.Unlock()
	finishSeam.before[j] = f
}

// finishCalls is the number of Finish calls the tests made for j.
func finishCalls(j *Job) int {
	finishSeam.Lock()
	defer finishSeam.Unlock()
	return finishSeam.calls[j]
}

// callFinish runs the hook, counts the call and calls j.Finish. Only a
// request's goroutine calls it.
func callFinish(j *Job) chain.Final {
	finishSeam.Lock()
	f := finishSeam.before[j]
	finishSeam.Unlock()
	if f != nil {
		f()
	}
	finishSeam.Lock()
	finishSeam.calls[j]++
	finishSeam.Unlock()
	return j.Finish(0)
}

// requestOf returns j's request, or nil when Finish was never issued.
func requestOf(j *Job) *finishReq {
	finishSeam.Lock()
	defer finishSeam.Unlock()
	return finishSeam.reqs[j]
}

// requestFinish returns j's request, registering it and starting its one
// goroutine when there is none. The registration comes before the goroutine
// starts.
func requestFinish(j *Job) *finishReq {
	finishSeam.Lock()
	if r := finishSeam.reqs[j]; r != nil {
		finishSeam.Unlock()
		return r
	}
	r := &finishReq{done: make(chan struct{})}
	finishSeam.reqs[j] = r
	finishSeam.Unlock()
	go func() {
		r.final = callFinish(j)
		close(r.done)
	}()
	return r
}

// wait waits at most d for the request to return. A timeout is only this
// wait's failure: the request stays registered and may still return.
func (r *finishReq) wait(d time.Duration) (chain.Final, bool) {
	select {
	case <-r.done:
		return r.final, true
	case <-time.After(d):
		return chain.Final{}, false
	}
}

// tryFinish issues j's closeout if it was never issued and waits at most d
// for that one request.
func tryFinish(j *Job, d time.Duration) (chain.Final, bool) {
	return requestFinish(j).wait(d)
}
