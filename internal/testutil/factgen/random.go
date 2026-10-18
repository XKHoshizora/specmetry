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
	// Trees are the trees of the generated commits, which the runs read
	// their inputs from: the generator's witness, for a test to check the
	// input records against (legality.CheckTrees).
	Trees Trees
}

// Trees are commit trees: for each commit, the entry of each lookup path
// it has (data-model §4.11.2); a path a commit does not have is missing
// there. A commit's tree never changes, whatever the worktree does.
type Trees map[string]map[string]InputEntry

// inputPaths are the paths the generated checks declare.
var inputPaths = []string{"a", "b/c.go", "tests"}

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

		pending: map[string][]string{}, asked: map[string]bool{}, named: map[string]map[string]bool{},
		eligible: map[string]map[string]bool{},
	}
	g.isolated = o.Isolated
	g.trees = Trees{}
	for i := range 5 {
		c := Commit(fmt.Sprintf("commit %d", i))
		g.commits = append(g.commits, c)
		// Each commit has most of the input paths: a run of a check whose
		// input its HEAD lacks is input_missing (§7.4 step 2).
		g.trees[c] = map[string]InputEntry{}
		for _, p := range inputPaths {
			if g.chance(0.85) {
				g.trees[c][p] = treeEntry(c, p)
			}
		}
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
	c := Case{Seed: seed, History: g.h, Dump: g.h.Dump(), AsOf: g.h.Seq(), Trees: g.trees}
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
	trees    Trees
	s        string // the commit the target's ref points to; "" unresolved

	proposals []string                   // in submission order
	pending   map[string][]string        // proposal -> its pending items
	asked     map[string]bool            // items a question is about
	named     map[string]map[string]bool // proposal -> the targets its items named
	// eligible holds, for each proposal, the entities active when it was
	// submitted: the only keys its items can name (V-P03; a target is a
	// key, and a link is a key or a ref of the proposal).
	eligible map[string]map[string]bool
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

func (g *gen) randomCheck() *baseline.Check { return g.checkOf(inputPaths) }

// checkOf is a random check that declares one of paths as its input, or
// none.
func (g *gen) checkOf(paths []string) *baseline.Check {
	c := &baseline.Check{Command: pickOne(g, []string{"go test ./a", "go test ./b", "make check"}), Workdir: ".", TimeoutSec: 300, Rationale: "r"}
	if len(paths) > 0 && g.chance(0.3) {
		c.Inputs = []string{pickOne(g, paths)}
	}
	return c
}

// missingAt reports whether a criterion's check declares an input the
// commit's tree does not have.
func (g *gen) missingAt(ac, commit string) bool {
	return slices.ContainsFunc(g.h.Inputs(ac), func(p string) bool {
		_, ok := g.trees[commit][p]
		return !ok
	})
}

// pathsAt are the input paths a commit's tree has.
func (g *gen) pathsAt(commit string) []string {
	var out []string
	for _, p := range inputPaths {
		if _, ok := g.trees[commit][p]; ok {
			out = append(out, p)
		}
	}
	return out
}

// review is a review of one proposal's pending items, or a person's edit.
// The plan is made phase by phase in the order of data-model §5.5, so that
// each change is valid when its step applies: a proposal names each target
// once (V-P03); no change of an entity the write retires; a requirement
// under a goal the write retires can still be updated or retired, after
// the cascade that loses it the goal; criteria added to requirements that
// are active then; an item a question is about becomes a requirement or
// criterion add; follow-ups only for the criteria of a requirement the
// review revises materially. The planned changes are handed to History in
// a shuffled order: History applies them in the order of §5.5 by step and
// item_index.
func (g *gen) review(apply bool) {
	var proposal string
	var decided []string
	if !apply {
		var open []string
		for _, p := range g.proposals {
			if len(g.pending[p]) > 0 {
				open = append(open, p)
			}
		}
		if len(open) == 0 {
			apply = true
		} else {
			proposal = pickOne(g, open)
			items := g.pending[proposal]
			chosen := g.rng.Perm(len(items))[:1+g.pick(len(items))]
			var left []string
			for i, it := range items {
				if slices.Contains(chosen, i) {
					decided = append(decided, it)
				} else {
					left = append(left, it)
				}
			}
			g.pending[proposal] = left
		}
	}
	write := g.h.Review
	if apply {
		write = g.h.Apply
	}
	write(func(r *Review) {
		named := map[string]bool{} // a new edit's proposal
		if !apply {
			if g.named[proposal] == nil {
				g.named[proposal] = map[string]bool{}
			}
			named = g.named[proposal]
		}
		p := &plan{g: g, r: r, apply: apply, named: named, eligible: g.eligible[proposal], retiredGoals: map[string]bool{},
			retiredReqs: map[string]bool{}, material: map[string]bool{}, here: map[string]bool{}}
		var slots []slot
		if apply {
			for range 1 + g.pick(4) {
				slots = append(slots, slot{})
			}
		}
		for _, it := range decided {
			if g.chance(0.25) {
				r.RejectItem(it)
				continue
			}
			slots = append(slots, slot{item: it, addOnly: g.asked[it]})
		}
		// Each slot gets a step; the steps are planned in order. A slot a
		// question is about is a requirement or criterion add: a question's
		// local about names one (proposal.CheckStatic).
		phases := make([][]slot, 3)
		for _, s := range slots {
			n := g.pick(3)
			if s.addOnly {
				n = 1 + g.pick(2)
			}
			phases[n] = append(phases[n], s)
		}
		for _, s := range phases[0] {
			p.goal(s)
		}
		for _, s := range phases[1] {
			p.requirement(s)
		}
		for _, s := range phases[2] {
			p.criterion(s)
		}
		if !apply {
			p.followups()
		}
		g.rng.Shuffle(len(p.calls), func(i, j int) { p.calls[i], p.calls[j] = p.calls[j], p.calls[i] })
		for _, call := range p.calls {
			call()
		}
		if srcs := g.h.SourceIDs(); len(srcs) > 0 {
			for _, e := range p.changedReqs {
				if g.chance(0.3) {
					r.Ref(e, pickOne(g, srcs), pickOne(g, []string{"verified", "verified", "not_found"}))
				}
			}
		}
		if g.chance(0.3) && g.h.ActiveTarget() != "" {
			var after []string
			for _, req := range g.active("requirement") {
				if !p.retiredReqs[req] {
					after = append(after, req)
				}
			}
			after = append(after, p.newReqs...)
			var scope []string
			for _, req := range after {
				if g.chance(0.6) {
					scope = append(scope, req)
				}
			}
			if len(scope) == 0 && len(after) > 0 {
				scope = append(scope, pickOne(g, after))
			}
			if len(scope) > 0 {
				r.Scope(scope...)
			}
		}
	})
}

// slot is one item of a write: a pending item of the reviewed proposal
// (addOnly when a question is about it: a requirement or criterion add),
// or a new item of an edit.
type slot struct {
	item    string
	addOnly bool
}

// plan is a write being planned, step by step.
type plan struct {
	g            *gen
	r            *Review
	apply        bool
	named        map[string]bool // the targets the proposal names
	eligible     map[string]bool // the keys the proposal can name; nil: an edit's, every active one
	retiredGoals map[string]bool
	retiredReqs  map[string]bool
	material     map[string]bool // requirements revised materially
	here         map[string]bool // entities an item of this write names
	newGoals     []string
	newReqs      []string
	changedReqs  []string
	calls        []func()
}

// do records a planned change and binds it to the slot's item.
func (p *plan) do(s slot, change func()) {
	p.calls = append(p.calls, func() {
		change()
		if !p.apply {
			p.r.FromItem(s.item)
		}
	})
}

func (p *plan) name(id string) {
	p.named[id], p.here[id] = true, true
}

// existing are the active entities of a type the proposal can name.
func (p *plan) existing(typ string) []string {
	var out []string
	for _, id := range p.g.active(typ) {
		if p.eligible == nil || p.eligible[id] {
			out = append(out, id)
		}
	}
	return out
}

// free are the active entities of a type the proposal can name and no item
// of it names yet.
func (p *plan) free(typ string, ok func(id string) bool) []string {
	var out []string
	for _, id := range p.existing(typ) {
		if !p.named[id] && (ok == nil || ok(id)) {
			out = append(out, id)
		}
	}
	return out
}

func (p *plan) goal(s slot) {
	goals := p.free("goal", nil)
	if len(goals) == 0 || p.g.chance(0.6) {
		p.do(s, func() { p.newGoals = append(p.newGoals, p.r.AddGoal("goal")) })
		return
	}
	g := pickOne(p.g, goals)
	p.name(g)
	p.retiredGoals[g] = true
	p.do(s, func() { p.r.Retire(g) })
}

func (p *plan) requirement(s slot) {
	g := p.g
	reqs := p.free("requirement", nil)
	w := g.pick(10)
	if s.addOnly || len(reqs) == 0 || w < 5 {
		var goals []string
		for _, id := range p.existing("goal") {
			if !p.retiredGoals[id] {
				goals = append(goals, id)
			}
		}
		goals = append(goals, p.newGoals...)
		goal := ""
		if len(goals) > 0 && g.chance(0.7) {
			goal = pickOne(g, goals)
		}
		priority := pickOne(g, []string{"must", "should", "could"})
		p.do(s, func() {
			id := p.r.AddRequirement(goal, priority)
			p.newReqs = append(p.newReqs, id)
			p.changedReqs = append(p.changedReqs, id)
		})
		return
	}
	req := pickOne(g, reqs)
	p.name(req)
	if w < 9 {
		material := g.chance(0.5)
		if material {
			p.material[req] = true
		}
		title := fmt.Sprintf("title %d", g.h.Seq())
		p.do(s, func() { p.r.Update(req, material, func(f *Fields) { f.Title = title }) })
		p.changedReqs = append(p.changedReqs, req)
		return
	}
	p.retiredReqs[req] = true
	p.do(s, func() { p.r.Retire(req) })
}

func (p *plan) criterion(s slot) {
	g := p.g
	var parents []string
	for _, id := range p.existing("requirement") {
		if !p.retiredReqs[id] {
			parents = append(parents, id)
		}
	}
	parents = append(parents, p.newReqs...)
	acs := p.free("criterion", func(id string) bool { return !p.retiredReqs[g.h.Parent(id)] })
	w := g.pick(10)
	if len(parents) == 0 && (s.addOnly || len(acs) == 0) {
		p.requirement(s) // nothing to add a criterion to
		return
	}
	if s.addOnly || len(acs) == 0 || w < 5 {
		parent := pickOne(g, parents)
		verification, check := "manual", (*baseline.Check)(nil)
		if g.chance(0.7) {
			verification = "automated"
			if g.chance(0.85) {
				check = g.randomCheck()
			}
		}
		p.do(s, func() { p.r.AddCriterion(parent, verification, check) })
		return
	}
	ac := pickOne(g, acs)
	p.name(ac)
	switch {
	case w < 7:
		if g.h.Verification(ac) == "automated" && g.chance(0.5) {
			check := g.randomCheck()
			p.do(s, func() { p.r.Update(ac, true, func(f *Fields) { f.Check = check }) })
		} else {
			material, statement := g.chance(0.5), fmt.Sprintf("s %d", g.h.Seq())
			p.do(s, func() { p.r.Update(ac, material, func(f *Fields) { f.Statement = statement }) })
		}
	case w < 9:
		p.do(s, func() { p.r.Reaffirm(ac) })
	default:
		p.do(s, func() { p.r.Retire(ac) })
	}
}

// followups decides, for each active criterion of a requirement the review
// revises materially that no item names, to reaffirm, retire or defer it
// (step 8).
func (p *plan) followups() {
	g := p.g
	for _, ac := range g.active("criterion") {
		if !p.material[g.h.Parent(ac)] || p.here[ac] {
			continue
		}
		switch g.pick(5) {
		case 0, 1:
			p.calls = append(p.calls, func() { p.r.FollowUp(ac, "reaffirm") })
		case 2:
			p.calls = append(p.calls, func() { p.r.FollowUp(ac, "retire") })
		}
	}
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
	if len(items) > 0 {
		active := map[string]bool{}
		for _, typ := range []string{"goal", "requirement", "criterion"} {
			for _, id := range g.active(typ) {
				active[id] = true
			}
		}
		g.eligible[g.h.itemProposal[items[0]]] = active
	}
	for _, q := range qs {
		if q.AboutItem > 0 {
			g.asked[items[q.AboutItem-1]] = true
		}
	}
	if len(items) > 0 {
		pid := g.h.itemProposal[items[0]]
		g.proposals = append(g.proposals, pid)
		g.pending[pid] = items
	}
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
	// A declared input the tree of HEAD lacks fails the run before the
	// command runs (§7.4 step 2): no context change, no input record.
	if g.missingAt(ac, head) {
		s.Result, s.FailureKind = "error", "input_missing"
		g.h.Run(ac, s)
		return
	}
	for _, p := range g.h.Inputs(ac) {
		s.Inputs = append(s.Inputs, g.trees[head][p])
	}
	// §7.4 step 4: another HEAD at the end, or clean at the start and not
	// at the end, is error / context_changed; a run dirty from the start
	// is only not qualifying.
	switch w := g.pick(20); {
	case w < 2:
		s.TreeStart, s.TreeEnd = "dirty", "dirty"
	case w < 3:
		var others []string
		for _, c := range g.commits {
			if c != head {
				others = append(others, c)
			}
		}
		s.HeadEnd, s.Result, s.FailureKind = pickOne(g, others), "error", "context_changed"
	case w < 4:
		s.TreeEnd, s.Result, s.FailureKind = "dirty", "error", "context_changed"
	}
	// A synthetic isolated run of the general domain is a qualifying one on
	// S.
	qualifying := s.TreeStart == "" && s.FailureKind != "context_changed"
	if g.isolated && head == g.s && qualifying && g.chance(0.5) {
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
	// A person's edit: reaffirm the criteria whose requirement was revised,
	// give the criteria whose check declares an input S lacks a check S
	// can run (an update, which confirms the requirement's revision too),
	// give every requirement of the scope a criterion, attach the scope.
	g.h.Apply(func(r *Review) {
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
				switch {
				case g.h.HasCheck(ac) && g.missingAt(ac, g.s):
					check := g.checkOf(g.pathsAt(g.s))
					r.Update(ac, true, func(f *Fields) { f.Check = check })
				case g.h.ParentRevised(ac):
					r.Reaffirm(ac)
				}
			}
			if !has {
				r.AddCriterion(req, "automated", g.checkOf(g.pathsAt(g.s)))
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
		case !g.h.HasCheck(ac) || (len(g.snapshotRuns(ac)) == 0 && g.chance(0.3)):
			// No qualifying run on S (V = none): an exception (§7.3).
			g.h.Decide(ac, DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: g.s})
		default:
			s := RunSpec{Head: g.s}
			for _, p := range g.h.Inputs(ac) {
				s.Inputs = append(s.Inputs, g.trees[g.s][p])
			}
			if g.isolated && g.chance(0.8) {
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
		switch g.pick(20) {
		case 0:
			o.SHA256 = ""
		case 1:
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
