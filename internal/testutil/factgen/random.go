package factgen

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/XKHoshizora/specmetry/internal/baseline"
)

// Options control a random history.
type Options struct {
	// Steps is the number of random operations.
	Steps int
	// Isolated allows synthetic isolated runs: the general legal fact
	// domain (data-model §6.14). Without it every run is observed, as in
	// the MVP.
	Isolated bool
	// Deliver is the probability of a delivery phase that drives the
	// history to a ready state, which a later perturbation may undo.
	Deliver float64
	// Cutoff is the probability that the cutoff is an earlier seq rather
	// than the latest one.
	Cutoff float64
}

// Case is a generated history and the current-value inputs of one command.
type Case struct {
	Seed    uint64
	History *History
	Dump    Dump
	Current Current
	AsOf    int64
}

// FeatureWorktree is the second worktree of the generated histories.
const FeatureWorktree = "01JAAAAAAAFEATVRE000000000"

// Random generates a legal fact history (data-model §6.14): every write
// satisfies its write preconditions (§7.1, §7.3, §8.3), so the history
// includes reviews that share a seq, person's edits, target changes,
// legal histories that cannot be ready and, with Deliver, ready ones. The
// same seed gives the same case.
func Random(seed uint64, o Options) Case {
	g := &gen{
		rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		h:   New("01JAAAAAAAPR0JECT000000000"),
	}
	g.isolated = o.Isolated
	for i := range 5 {
		g.commits = append(g.commits, Commit(fmt.Sprintf("commit %d", i)))
	}
	g.s = g.commits[0]
	g.h.SetTarget("default", "refs/heads/main")
	for range o.Steps {
		g.step()
	}
	if g.rng.Float64() < o.Deliver {
		g.deliver()
		for range g.rng.IntN(3) {
			if g.rng.IntN(2) == 0 {
				g.step()
			}
		}
	}
	c := Case{Seed: seed, History: g.h, Dump: g.h.Dump(), AsOf: g.h.Seq()}
	if g.rng.Float64() < o.Cutoff {
		c.AsOf = g.rng.Int64N(g.h.Seq() + 1)
	}
	c.Current = g.current(c.Dump, c.AsOf)
	return c
}

type gen struct {
	rng      *rand.Rand
	h        *History
	isolated bool
	commits  []string
	s        string // the commit the target's ref points to; "" unresolved
	pending  []string
}

func (g *gen) pick(n int) int { return g.rng.IntN(n) }

func (g *gen) chance(p float64) bool { return g.rng.Float64() < p }

func pickOne[T any](g *gen, s []T) T { return s[g.pick(len(s))] }

func (g *gen) active(typ string) []string {
	var out []string
	for _, id := range g.h.Entities(typ) {
		if g.h.Active(id) {
			out = append(out, id)
		}
	}
	return out
}

func (g *gen) step() {
	switch w := g.pick(100); {
	case w < 14:
		g.review(false)
	case w < 20:
		g.review(true)
	case w < 26:
		g.propose()
	case w < 30:
		g.confirmScope()
	case w < 34:
		g.source()
	case w < 37:
		g.answer()
	case w < 47:
		g.claims()
	case w < 67:
		g.run()
	case w < 82:
		g.decide()
	case w < 87:
		g.scan()
	case w < 95:
		g.moveRef()
	default:
		if g.chance(0.3) {
			g.h.SetTarget(fmt.Sprintf("target %d", g.h.Seq()), "refs/heads/release")
		}
	}
}

func (g *gen) moveRef() {
	if g.chance(0.1) {
		g.s = ""
		return
	}
	g.s = pickOne(g, g.commits)
}

func (g *gen) randomCheck() *baseline.Check {
	c := &baseline.Check{Command: pickOne(g, []string{"go test ./a", "go test ./b", "make check"}), Workdir: ".", TimeoutSec: 300, Rationale: "r"}
	if g.chance(0.3) {
		c.Inputs = []string{pickOne(g, []string{"a", "b/c.go", "tests"})}
	}
	return c
}

// review applies a review (or, with apply, a person's edit) in the order of
// data-model §5.5: goals and requirements first, then criteria. Pending
// items are accepted as the changes, or rejected.
func (g *gen) review(apply bool) {
	var items []string
	if !apply && len(g.pending) > 0 {
		n := 1 + g.pick(len(g.pending))
		items, g.pending = g.pending[:n], g.pending[n:]
	}
	write := g.h.Review
	if apply {
		write = g.h.Apply
	}
	write(func(r *Review) {
		next := func() {
			if len(items) > 0 && !apply {
				if g.chance(0.25) {
					r.RejectItem(items[0])
					items = items[1:]
					return
				}
				r.FromItem(items[0])
				items = items[1:]
			}
		}
		// Goals and requirements.
		for range 1 + g.pick(3) {
			goals, reqs := g.active("goal"), g.active("requirement")
			switch w := g.pick(10); {
			case w < 2 || len(goals) == 0:
				r.AddGoal("goal")
			case w < 6 || len(reqs) == 0:
				goal := ""
				if g.chance(0.7) {
					goal = pickOne(g, goals)
				}
				r.AddRequirement(goal, pickOne(g, []string{"must", "should", "could"}))
			case w < 8:
				material := g.chance(0.5)
				r.Update(pickOne(g, reqs), material, func(f *Fields) { f.Title = fmt.Sprintf("title %d", g.h.Seq()) })
			case w < 9:
				r.Retire(pickOne(g, reqs))
			default:
				r.Retire(pickOne(g, goals))
			}
			next()
		}
		// Criteria.
		for range g.pick(4) {
			reqs, acs := g.active("requirement"), g.active("criterion")
			if len(reqs) == 0 {
				break
			}
			switch w := g.pick(10); {
			case w < 5 || len(acs) == 0:
				if g.chance(0.7) {
					var c *baseline.Check
					if g.chance(0.85) {
						c = g.randomCheck()
					}
					r.AddCriterion(pickOne(g, reqs), "automated", c)
				} else {
					r.AddCriterion(pickOne(g, reqs), "manual", nil)
				}
			case w < 7:
				ac := pickOne(g, acs)
				if g.h.Verification(ac) == "automated" && g.chance(0.5) {
					r.Update(ac, true, func(f *Fields) { f.Check = g.randomCheck() })
				} else {
					r.Update(ac, g.chance(0.5), func(f *Fields) { f.Statement = fmt.Sprintf("s %d", g.h.Seq()) })
				}
			case w < 9:
				r.Reaffirm(pickOne(g, acs))
			default:
				r.Retire(pickOne(g, acs))
			}
			next()
		}
		for _, item := range items {
			r.RejectItem(item)
		}
		if srcs := g.h.SourceIDs(); len(srcs) > 0 {
			for _, e := range g.active("requirement") {
				if g.chance(0.2) {
					r.Ref(e, pickOne(g, srcs), pickOne(g, []string{"verified", "verified", "not_found"}))
				}
			}
		}
		if g.chance(0.3) && g.h.ActiveTarget() != "" {
			if reqs := g.scopeCandidates(); len(reqs) > 0 {
				r.Scope(reqs...)
			}
		}
	})
}

// scopeCandidates is a random non-empty subset of the active requirements.
func (g *gen) scopeCandidates() []string {
	reqs := g.active("requirement")
	var out []string
	for _, r := range reqs {
		if g.chance(0.6) {
			out = append(out, r)
		}
	}
	if len(out) == 0 && len(reqs) > 0 {
		out = append(out, pickOne(g, reqs))
	}
	return out
}

func (g *gen) propose() {
	n := g.pick(3)
	var qs []QuestionSpec
	for range g.pick(3) {
		q := QuestionSpec{Blocking: g.chance(0.5)}
		switch w := g.pick(3); {
		case w == 0 && n > 0:
			q.AboutItem = 1 + g.pick(n)
		case w == 1:
			if ents := slices.Concat(g.active("requirement"), g.active("criterion")); len(ents) > 0 {
				q.AboutEntity = pickOne(g, ents)
			}
		}
		qs = append(qs, q)
	}
	items, _ := g.h.Propose(n, qs...)
	g.pending = append(g.pending, items...)
}

func (g *gen) confirmScope() {
	if g.h.ActiveTarget() == "" || g.h.Version() == 0 {
		return
	}
	if reqs := g.scopeCandidates(); len(reqs) > 0 {
		g.h.ConfirmScope(reqs...)
	}
}

func (g *gen) source() {
	srcs := g.h.SourceIDs()
	switch w := g.pick(10); {
	case w < 5 || len(srcs) == 0:
		g.h.AddSource(fmt.Sprintf("docs/%d.md", g.h.Seq()), Hash(fmt.Sprintf("v%d", g.h.Seq())))
	case w < 8:
		g.h.AckSource(pickOne(g, srcs), Hash(fmt.Sprintf("v%d", g.h.Seq())))
	default:
		g.h.RemoveSource(pickOne(g, srcs))
	}
}

func (g *gen) answer() {
	if qs := g.h.Questions(); len(qs) > 0 {
		g.h.Answer(pickOne(g, qs))
	}
}

func (g *gen) claims() {
	acs := g.active("criterion")
	kind := "sync"
	if g.chance(0.3) {
		kind = "audit"
	}
	var claims []ClaimSpec
	for _, ac := range acs {
		if g.chance(0.4) {
			assertions := []string{"done", "partial", "not_done", "blocked"}
			if kind == "audit" {
				assertions = append(assertions, "unknown")
			}
			claims = append(claims, ClaimSpec{Criterion: ac, Assertion: pickOne(g, assertions)})
		}
	}
	if g.chance(0.2) {
		claims = append(claims, ClaimSpec{Assertion: "done"})
	}
	if len(claims) > 0 {
		g.h.Claims(kind, claims...)
	}
}

// treeEntry is the input record entry of a path at a commit: each commit
// has its own tree.
func treeEntry(commit, path string) InputEntry {
	typ, mode := "blob", "100644"
	if path == "tests" {
		typ, mode = "tree", "040000"
	}
	return InputEntry{Path: path, Mode: mode, Type: typ, OID: Commit(commit + ":" + path)}
}

func (g *gen) run() {
	var acs []string
	for _, ac := range g.active("criterion") {
		if g.h.HasCheck(ac) {
			acs = append(acs, ac)
		}
	}
	if len(acs) == 0 {
		return
	}
	ac := pickOne(g, acs)
	head := g.s
	if head == "" || g.chance(0.3) {
		head = pickOne(g, g.commits)
	}
	s := RunSpec{Head: head, Worktree: MainWorktree}
	if g.chance(0.3) {
		s.Worktree = FeatureWorktree
	}
	switch w := g.pick(10); {
	case w < 6:
		s.Result = "pass"
	case w < 9:
		s.Result, s.FailureKind = "fail", "nonzero_exit"
	default:
		s.Result, s.FailureKind = "error", pickOne(g, []string{"timeout", "not_found", "spawn_error"})
	}
	switch w := g.pick(20); {
	case w < 2:
		s.TreeStart, s.TreeEnd = "dirty", "dirty"
	case w < 3:
		s.HeadEnd, s.Result, s.FailureKind = pickOne(g, g.commits), "error", "context_changed"
	case w < 4:
		s.TreeEnd, s.Result, s.FailureKind = "dirty", "error", "context_changed"
	}
	if inputs := g.h.Inputs(ac); len(inputs) > 0 {
		if g.chance(0.1) {
			s.Result, s.FailureKind = "error", "input_missing"
		} else {
			for _, p := range inputs {
				s.Inputs = append(s.Inputs, treeEntry(head, p))
			}
		}
	}
	if g.isolated && head == g.s && s.FailureKind != "context_changed" && s.TreeStart == "" && g.chance(0.5) {
		s.Grade = "snapshot_isolated"
	}
	g.h.Run(ac, s)
}

// snapshotRuns are the criterion's qualifying runs on S of its current
// revision and configuration, in seq order: the write side's view for the
// decision preconditions (data-model §7.3).
func (g *gen) snapshotRuns(ac string) []CheckRunRow {
	var out []CheckRunRow
	for _, r := range g.h.Runs(ac) {
		if r.CriterionRevision == g.h.Revision(ac) && r.CheckConfigHash == g.h.ConfigHash(ac) &&
			r.HeadStart != nil && r.HeadEnd != nil && *r.HeadStart == *r.HeadEnd && *r.HeadStart == g.s &&
			r.TreeStateStart == "clean" && r.TreeStateEnd == "clean" && (r.FailureKind == nil || *r.FailureKind != "context_changed") {
			out = append(out, r)
		}
	}
	return out
}

// decide makes a decision that satisfies data-model §7.1 and §7.3.
func (g *gen) decide() {
	acs := g.active("criterion")
	if len(acs) == 0 || g.h.ActiveTarget() == "" {
		return
	}
	ac := pickOne(g, acs)
	var options []DecisionSpec
	parentOK := !g.h.ParentRevised(ac)
	if parentOK {
		options = append(options, DecisionSpec{Decision: "waived"})
	}
	if g.s != "" {
		runs := g.snapshotRuns(ac)
		reject := DecisionSpec{Decision: "rejected", Snapshot: g.s}
		if n := len(runs); n > 0 {
			reject.Evidence = runs[n-1].ID // the latest qualifying run the person saw
		}
		options = append(options, reject)
		switch {
		case !parentOK:
		case g.h.Verification(ac) == "manual":
			options = append(options, DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: g.s})
		case len(runs) == 0:
			options = append(options, DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: g.s})
		case runs[len(runs)-1].Result == "pass":
			latest := runs[len(runs)-1].ID
			normal := DecisionSpec{Decision: "accepted", Mode: "normal", Snapshot: g.s, Evidence: latest}
			options = append(options, normal, normal)
		}
	}
	if len(options) == 0 {
		return
	}
	g.h.Decide(ac, pickOne(g, options))
}

func (g *gen) scan() {
	ents := slices.Concat(g.active("requirement"), g.active("criterion"))
	if len(ents) == 0 {
		return
	}
	var commits []CommitSpec
	for range 1 + g.pick(2) {
		c := CommitSpec{SHA: Commit(fmt.Sprintf("found %d %d", g.h.Seq(), len(commits)))}
		for range 1 + g.pick(2) {
			c.Links = append(c.Links, pickOne(g, ents))
		}
		slices.Sort(c.Links)
		c.Links = slices.Compact(c.Links)
		commits = append(commits, c)
	}
	g.h.Scan(commits...)
}

// deliver drives the history to a ready state: a resolvable S, every
// criterion of a fresh scope accepted or waived (not all waived), no open
// blocking question and the observations at the acknowledged hashes.
func (g *gen) deliver() {
	if g.h.ActiveTarget() == "" {
		g.h.SetTarget("default", "refs/heads/main")
	}
	if g.s == "" {
		g.s = g.commits[0]
	}
	reqs := g.active("requirement")
	var scoped []string
	g.h.Review(func(r *Review) {
		if len(reqs) == 0 {
			reqs = []string{r.AddRequirement("", "must")}
		}
		for _, req := range reqs {
			if !g.chance(0.6) && len(scoped) > 0 {
				continue
			}
			scoped = append(scoped, req)
			has := false
			for _, ac := range g.active("criterion") {
				if g.h.Parent(ac) != req {
					continue
				}
				has = true
				if g.h.ParentRevised(ac) {
					r.Reaffirm(ac)
				}
			}
			if !has {
				r.AddCriterion(req, "automated", g.randomCheck())
			}
		}
		r.Scope(scoped...)
	})
	for _, q := range g.h.Questions() {
		g.h.Answer(q)
	}
	var inScope []string
	for _, ac := range g.active("criterion") {
		if slices.Contains(scoped, g.h.Parent(ac)) {
			inScope = append(inScope, ac)
		}
	}
	for i, ac := range inScope {
		if i > 0 && g.chance(0.15) {
			g.h.Decide(ac, DecisionSpec{Decision: "waived"})
			continue
		}
		switch {
		case g.h.Verification(ac) == "manual":
			g.h.Decide(ac, DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: g.s})
		case !g.h.HasCheck(ac):
			g.h.Decide(ac, DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: g.s})
		default:
			s := RunSpec{Head: g.s}
			for _, p := range g.h.Inputs(ac) {
				s.Inputs = append(s.Inputs, treeEntry(g.s, p))
			}
			if g.isolated && g.chance(0.5) {
				s.Grade = "snapshot_isolated"
			}
			run := g.h.Run(ac, s)
			g.h.Decide(ac, DecisionSpec{Decision: "accepted", Mode: "normal", Snapshot: g.s, Evidence: run})
		}
	}
}

// current is the command's current-value inputs at a cutoff: S, the
// observations of the sources in the source set at the cutoff (mostly at
// their acknowledged hash, sometimes changed or missing) and random
// settings.
func (g *gen) current(d Dump, asOf int64) Current {
	c := Current{
		S:                g.s,
		Stage:            pickOne(g, []string{"docs_only", "in_development", "in_development", "incomplete_docs"}),
		SourcesSkipped:   g.chance(0.3),
		InventorySkipped: g.chance(0.3),
	}
	for _, o := range ObservationsAt(d, asOf) {
		switch w := g.pick(20); {
		case w == 0:
			o.SHA256 = ""
		case w == 1:
			o.SHA256 = Hash("edited")
		}
		c.Observations = append(c.Observations, o)
	}
	if g.chance(0.5) {
		c.Workspace = &Workspace{WorktreeID: MainWorktree, Head: pickOne(g, g.commits), TreeState: "clean"}
	}
	return c
}

// ObservationsAt observes every source in the source set at a cutoff at its
// hash acknowledged at that cutoff: the observations of a command that
// finds no source changed.
func ObservationsAt(d Dump, asOf int64) []Observation {
	var out []Observation
	for _, s := range d.Sources {
		if s.RegisteredSeq > asOf || (s.RemovedSeq != nil && *s.RemovedSeq <= asOf) {
			continue
		}
		acked, first := s.AckedSHA256, true
		for _, a := range d.SourceAcks { // in seq order
			if a.SourceID != s.ID {
				continue
			}
			if first {
				acked, first = a.PreviousSHA256, false
			}
			if a.Seq <= asOf {
				acked = a.AckedSHA256
			}
		}
		out = append(out, Observation{SourceID: s.ID, SHA256: acked})
	}
	return out
}
