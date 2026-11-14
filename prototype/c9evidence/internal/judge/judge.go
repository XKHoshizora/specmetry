// Package judge holds the result model every check of the tool shares: the
// three results of a check (method record §6.1), the J classes it uses
// (§6.3: J-1 to J-4; J-5 to J-10 belong to B′ and C, which this tool does
// not implement, plan §5.3), and the per-instance registry of checks with
// their entry records (plan §5.1).
package judge

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Class is a J class of the method record §6.3. This tool records only
// J-1 to J-4; anything else is outside its check set E.
type Class int

// The classes this tool records.
const (
	J1 Class = 1 // tool or read error, budget, guard
	J2 Class = 2 // source or environment anomaly
	J3 Class = 3 // incomplete collection
	J4 Class = 4 // content changed
)

func (c Class) String() string { return fmt.Sprintf("J-%d", int(c)) }

// Source says where a failure came from (plan §5.5 item 1).
type Source string

// The sources.
const (
	SourceReal     Source = "real"
	SourceInjected Source = "injected"
	SourceBudget   Source = "budget"
)

// Failure is one recorded failure. Item is the part of the check it is
// about (a key, a field, a module, a package, a file); Site names the
// operation; Kind is the error kind ("close error", "budget insufficient",
// "decode", "mismatch", ...). Inject is the injection serial when Source is
// SourceInjected, otherwise 0.
type Failure struct {
	Class  Class
	Item   string
	Site   string
	Kind   string
	Source Source
	Inject int
	Detail string
}

func (f Failure) String() string {
	s := fmt.Sprintf("%s %s [%s/%s] at %s", f.Class, f.Item, f.Kind, f.Source, f.Site)
	if f.Inject != 0 {
		s += fmt.Sprintf(" #%d", f.Inject)
	}
	if f.Detail != "" {
		s += ": " + f.Detail
	}
	return s
}

// Real builds a failure whose source is a real result.
func Real(c Class, item, site, kind, detail string) Failure {
	return Failure{Class: c, Item: item, Site: site, Kind: kind, Source: SourceReal, Detail: detail}
}

// Status is the result of a check.
type Status int

// The statuses. Unset means the check has no result yet.
const (
	Unset Status = iota
	Hold
	Fail
	Unevaluated
)

func (s Status) String() string {
	switch s {
	case Hold:
		return "hold"
	case Fail:
		return "fail"
	case Unevaluated:
		return "unevaluated"
	}
	return "unset"
}

// Result is a check's result: Hold, Fail with at least one failure, or
// Unevaluated with the dependency or admission that kept it from running.
type Result struct {
	Status   Status
	Failures []Failure
	Reason   string
}

// Acc accumulates the failures of a check that has several items; every
// item is evaluated and every failure kept (method record §6.2: no stop at
// the first error).
type Acc struct{ fs []Failure }

// Add records one failure.
func (a *Acc) Add(f Failure) { a.fs = append(a.fs, f) }

// Failed reports whether any failure was recorded.
func (a *Acc) Failed() bool { return len(a.fs) > 0 }

// Failures returns a copy of the failures.
func (a *Acc) Failures() []Failure { return append([]Failure(nil), a.fs...) }

// Result returns Hold when nothing failed, otherwise Fail with the failures.
func (a *Acc) Result() Result {
	if len(a.fs) == 0 {
		return Result{Status: Hold}
	}
	return Result{Status: Fail, Failures: a.Failures()}
}

// Entry is an entry record: the check was entered with this input.
type Entry struct {
	Seq         int
	Check       string
	InputSHA256 string
}

// Item is one check in a snapshot.
type Item struct {
	ID      string
	Result  Result
	Entered bool
}

// Checks is the registry of one instance: an R configuration, or a tested
// fixture instance of a control. It accepts only the IDs it was created
// with; recording any other ID (for example one of the not-covered set N)
// or setting a result twice is a tool defect, kept and reported, and the
// first result stays (a result is never rewritten, method record §6.1).
type Checks struct {
	mu      sync.Mutex
	ids     []string
	known   map[string]bool
	results map[string]Result
	entered map[string]bool
	entries []Entry
	defects []string
	seq     int
}

// NewChecks makes a registry for exactly these check IDs.
func NewChecks(ids []string) *Checks {
	c := &Checks{
		ids:     append([]string(nil), ids...),
		known:   map[string]bool{},
		results: map[string]Result{},
		entered: map[string]bool{},
	}
	for _, id := range ids {
		if c.known[id] {
			panic("judge: duplicate check id " + id)
		}
		c.known[id] = true
	}
	return c
}

// Enter writes the entry record of a check: it was entered with an input
// whose SHA-256 is inputSHA ("" when the check has no byte input).
func (c *Checks) Enter(id, inputSHA string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.known[id] {
		c.defects = append(c.defects, "entry for unknown check "+id)
		return
	}
	c.seq++
	c.entered[id] = true
	c.entries = append(c.entries, Entry{Seq: c.seq, Check: id, InputSHA256: inputSHA})
}

// Set records the result of a check, once. Hold and Fail need an entry
// record first; Fail needs at least one failure.
func (c *Checks) Set(id string, r Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case !c.known[id]:
		c.defects = append(c.defects, "result for unknown check "+id)
		return
	case c.results[id].Status != Unset:
		c.defects = append(c.defects, "second result for check "+id)
		return
	case r.Status == Unset:
		c.defects = append(c.defects, "unset result for check "+id)
		return
	case r.Status == Fail && len(r.Failures) == 0:
		c.defects = append(c.defects, "fail without failures for check "+id)
		return
	case (r.Status == Hold || r.Status == Fail) && !c.entered[id]:
		c.defects = append(c.defects, "evaluated without entry record: "+id)
	}
	r.Failures = append([]Failure(nil), r.Failures...)
	c.results[id] = r
}

// Hold records that the check holds.
func (c *Checks) Hold(id string) { c.Set(id, Result{Status: Hold}) }

// Fail records that the check failed with these failures.
func (c *Checks) Fail(id string, fs ...Failure) { c.Set(id, Result{Status: Fail, Failures: fs}) }

// Unevaluated records that the check was not evaluated, and why.
func (c *Checks) Unevaluated(id, reason string) {
	c.Set(id, Result{Status: Unevaluated, Reason: reason})
}

// SetIfUnset gives every check that has no result yet this unevaluated
// reason; it never touches a result already set.
func (c *Checks) SetIfUnset(reason string) {
	c.mu.Lock()
	ids := append([]string(nil), c.ids...)
	c.mu.Unlock()
	for _, id := range ids {
		if r, _ := c.Get(id); r.Status == Unset {
			c.Unevaluated(id, reason)
		}
	}
}

// Get returns a check's result.
func (c *Checks) Get(id string) (Result, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r, ok := c.results[id]
	if ok {
		r.Failures = append([]Failure(nil), r.Failures...)
	}
	return r, ok
}

// Status returns a check's status (Unset when it has none).
func (c *Checks) Status(id string) Status {
	r, _ := c.Get(id)
	return r.Status
}

// AllHold reports whether every given check holds.
func (c *Checks) AllHold(ids ...string) bool {
	for _, id := range ids {
		if c.Status(id) != Hold {
			return false
		}
	}
	return true
}

// Snapshot returns every check in creation order with its result.
func (c *Checks) Snapshot() []Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Item, 0, len(c.ids))
	for _, id := range c.ids {
		r := c.results[id]
		r.Failures = append([]Failure(nil), r.Failures...)
		out = append(out, Item{ID: id, Result: r, Entered: c.entered[id]})
	}
	return out
}

// Entries returns the entry records.
func (c *Checks) Entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Entry(nil), c.entries...)
}

// Defects returns the tool defects the registry saw.
func (c *Checks) Defects() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.defects...)
}

// Complete reports whether every check has a result.
func (c *Checks) Complete() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range c.ids {
		if c.results[id].Status == Unset {
			return false
		}
	}
	return true
}

// Failures returns every failure of every check, in check order.
func (c *Checks) Failures() []Failure {
	var out []Failure
	for _, it := range c.Snapshot() {
		out = append(out, it.Result.Failures...)
	}
	return out
}

// Summary is a stable one-line-per-check rendering, for records and tests.
func Summary(items []Item) string {
	var b strings.Builder
	for _, it := range items {
		fmt.Fprintf(&b, "%s %s", it.ID, it.Result.Status)
		if it.Result.Status == Fail {
			fs := make([]string, 0, len(it.Result.Failures))
			for _, f := range it.Result.Failures {
				fs = append(fs, f.Class.String()+":"+f.Item)
			}
			sort.Strings(fs)
			fmt.Fprintf(&b, " %s", strings.Join(fs, ","))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
