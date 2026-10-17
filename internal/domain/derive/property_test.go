package derive_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// The property tests run over fixed seeds, so every run checks the same
// histories; a failure names its seed, and the subtest reruns it alone:
//
//	go test ./internal/domain/derive -run 'TestGapsAndDelivery/seed=17'
const (
	propertySeeds = 400
	shuffleSeeds  = 4
	shuffles      = 1000
)

func randomOptions(seed uint64) factgen.Options {
	return factgen.Options{Steps: 40, Deliver: 0.5, Cutoff: 0.2, Isolated: seed%3 == 0}
}

// legalCase is a random history the legality checker confirms is legal
// (data-model §6.14): the property tests run only on legal histories, and
// a generated one that is not fails with its operation trace.
func legalCase(t *testing.T, seed uint64, o factgen.Options) factgen.Case {
	t.Helper()
	c := factgen.Random(seed, o)
	if p := legality.Check(c.Dump, legality.Options{Isolated: o.Isolated}); len(p) > 0 {
		t.Fatalf("seed %d is not a legal history:\n%s", seed, legality.Report(p, c.History.Trace()))
	}
	return c
}

// domainOf names a case's legal domain.
func domainOf(o factgen.Options) string {
	if o.Isolated {
		return "general"
	}
	return "MVP"
}

// shuffled is the input with the rows of every fact table and the
// observations in another physical order, including the order of the rows
// that share a seq.
func shuffled(in derive.Input, rng *rand.Rand) derive.Input {
	out := deepCopyFacts(in)
	f := &out.Facts
	shuffle(rng, f.Targets)
	shuffle(rng, f.BaselineVersions)
	shuffle(rng, f.BaselineChanges)
	shuffle(rng, f.Entities)
	shuffle(rng, f.ScopeConfirmations)
	shuffle(rng, f.ScopeItems)
	shuffle(rng, f.Sources)
	shuffle(rng, f.SourceAcks)
	shuffle(rng, f.SourceRefs)
	shuffle(rng, f.Proposals)
	shuffle(rng, f.ProposalItems)
	shuffle(rng, f.Questions)
	shuffle(rng, f.ClaimBatches)
	shuffle(rng, f.Claims)
	shuffle(rng, f.CheckRuns)
	shuffle(rng, f.Commits)
	shuffle(rng, f.CommitLinks)
	shuffle(rng, f.Decisions)
	for i := range f.CheckRuns {
		shuffle(rng, f.CheckRuns[i].Inputs)
	}
	for i := range f.Decisions {
		shuffle(rng, f.Decisions[i].Inputs)
	}
	shuffle(rng, out.Current.Observations)
	return out
}

func shuffle[T any](rng *rand.Rand, s []T) {
	rng.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

func deepCopyFacts(in derive.Input) derive.Input {
	out := in
	f := in.Facts
	out.Facts = derive.Facts{
		ProjectID: f.ProjectID, LatestSeq: f.LatestSeq,
		Targets: clone(f.Targets), BaselineVersions: clone(f.BaselineVersions), BaselineChanges: clone(f.BaselineChanges),
		Entities: clone(f.Entities), ScopeConfirmations: clone(f.ScopeConfirmations), ScopeItems: clone(f.ScopeItems),
		Sources: clone(f.Sources), SourceAcks: clone(f.SourceAcks), SourceRefs: clone(f.SourceRefs), Proposals: clone(f.Proposals),
		ProposalItems: clone(f.ProposalItems), Questions: clone(f.Questions), ClaimBatches: clone(f.ClaimBatches),
		Claims: clone(f.Claims), Commits: clone(f.Commits), CommitLinks: clone(f.CommitLinks),
	}
	for _, r := range f.CheckRuns {
		r.Inputs = clone(r.Inputs)
		out.Facts.CheckRuns = append(out.Facts.CheckRuns, r)
	}
	for _, d := range f.Decisions {
		d.Inputs = clone(d.Inputs)
		out.Facts.Decisions = append(out.Facts.Decisions, d)
	}
	out.Current.Observations = clone(in.Current.Observations)
	return out
}

// MVP-ST-05, invariant #7: shuffling the physical order of the rows 1000
// times, including rows that share a seq, never changes the derivation or
// the --diff-since projection.
func TestShuffledFactsGiveTheSameResult(t *testing.T) {
	for seed := range uint64(shuffleSeeds) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			c := legalCase(t, seed, factgen.Options{Steps: 60, Deliver: 0.5, Isolated: seed%2 == 0})
			in := toderive.Input(c.Dump, c.Current, c.AsOf)
			want := mustDerive(t, in)
			since := c.AsOf / 2
			wantChanges, err := derive.ChangesSince(in, since)
			if err != nil {
				t.Fatal(err)
			}
			rng := rand.New(rand.NewPCG(seed, 7))
			for i := range shuffles {
				s := shuffled(in, rng)
				if got := mustDerive(t, s); !reflect.DeepEqual(got, want) {
					t.Fatalf("shuffle %d changed the result", i)
				}
				if i%10 == 0 {
					got, err := derive.ChangesSince(s, since)
					if err != nil || !reflect.DeepEqual(got, wantChanges) {
						t.Fatalf("shuffle %d changed the changes: %v", i, err)
					}
				}
			}
		})
	}
}

// MVP-ST-13 and the derivation's own invariants, over random legal
// histories: ready if and only if no gap blocks; at most one state gap per
// criterion; with a scope, no gap outside it; accepted and waived criteria
// have no state gap; warnings never block; the blocking gaps of the
// delivery are the blocking gaps, in order.
func TestGapsAndDelivery(t *testing.T) {
	counts := map[string]int{}
	for seed := range uint64(propertySeeds) {
		o := randomOptions(seed)
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			c := legalCase(t, seed, o)
			res := mustDerive(t, toderive.Input(c.Dump, c.Current, c.AsOf))
			verdict := "not ready"
			if res.Delivery.Status == derive.TargetReady {
				verdict = "ready, " + string(res.Delivery.EvidenceGrade)
			}
			counts[domainOf(o)+": "+verdict]++
			checkGapInvariants(t, res)
		})
	}
	t.Logf("legal histories: %v", counts)
	// Each domain has both verdicts, and the grades it can have.
	for _, want := range []string{"MVP: not ready", "MVP: ready, observed", "MVP: ready, none", "general: not ready", "general: ready, snapshot_isolated"} {
		if counts[want] < 5 {
			t.Errorf("too few %q: %v", want, counts)
		}
	}
	if counts["MVP: ready, snapshot_isolated"] > 0 {
		t.Errorf("an isolated grade in the MVP domain: %v", counts)
	}
}

var stateGaps = map[derive.GapType]bool{
	derive.GapNeedsCriteria: true, derive.GapNoCheck: true, derive.GapClaimedWithoutEvidence: true,
	derive.GapNotVerified: true, derive.GapFailing: true, derive.GapRejected: true, derive.GapParentRevised: true,
	derive.GapNeedsReverification: true, derive.GapAwaitingReview: true,
}

var warnings = map[derive.GapType]bool{
	derive.GapConflict: true, derive.GapExceptionAcceptance: true, derive.GapInconsistentResults: true,
	derive.GapSourceRemoved: true, derive.GapPendingReview: true, derive.GapInventoryIncomplete: true,
}

func checkGapInvariants(t *testing.T, res *derive.Result) {
	t.Helper()
	var blocking []derive.Gap
	for _, g := range res.Gaps {
		if g.Blocking {
			blocking = append(blocking, g)
		}
		if warnings[g.Type] && g.Blocking {
			t.Errorf("warning %+v blocks", g)
		}
	}
	if !reflect.DeepEqual(blocking, res.Delivery.BlockingGaps) {
		t.Errorf("delivery blocking gaps %v, gaps %v", res.Delivery.BlockingGaps, res.Gaps)
	}
	if (res.Delivery.Status == derive.TargetReady) != (len(blocking) == 0) {
		t.Errorf("delivery %s with blocking gaps %v", res.Delivery.Status, gapTypes(blocking))
	}
	// Blocking gaps come first.
	for i := 1; i < len(res.Gaps); i++ {
		if !res.Gaps[i-1].Blocking && res.Gaps[i].Blocking {
			t.Errorf("gap %d blocks after a warning: %v", i, gapTypes(res.Gaps))
		}
	}
	inScope := map[string]bool{}
	for _, r := range res.Requirements {
		if r.InScope {
			inScope[r.Key] = true
		}
	}
	state := map[string]int{}
	for _, r := range res.Requirements {
		for _, cr := range r.Criteria {
			if r.InScope {
				inScope[cr.Key] = true
			}
		}
	}
	for _, g := range res.Gaps {
		if stateGaps[g.Type] {
			state[g.TargetKey]++
		}
		isEntity := strings.HasPrefix(g.TargetKey, "REQ-") || strings.HasPrefix(g.TargetKey, "AC-")
		if res.Scope != nil && isEntity && !inScope[g.TargetKey] {
			t.Errorf("gap %+v outside the scope", g)
		}
		if res.Scope == nil && stateGaps[g.Type] && g.Blocking {
			t.Errorf("gap %+v blocks without a scope", g)
		}
	}
	for _, r := range res.Requirements {
		for _, cr := range r.Criteria {
			if state[cr.Key] > 1 {
				t.Errorf("%s has %d state gaps", cr.Key, state[cr.Key])
			}
			done := cr.Status == derive.CriterionAccepted || cr.Status == derive.CriterionWaived || cr.Status == derive.CriterionNotAssessed
			if done && state[cr.Key] > 0 {
				t.Errorf("%s is %s and has a state gap", cr.Key, cr.Status)
			}
		}
	}
}

// MVP-ST-14 over random histories: a replay with the current values the
// result records gives the same result.
func TestReplayOfRandomResults(t *testing.T) {
	for seed := range uint64(propertySeeds / 4) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			c := legalCase(t, seed, randomOptions(seed))
			in := toderive.Input(c.Dump, c.Current, c.AsOf)
			res := mustDerive(t, in)
			replay := derive.Input{
				Facts: toderive.Facts(c.Dump),
				Current: derive.Current{
					Snapshot: &res.Snapshot, Settings: &res.Settings, Observations: res.Observations,
					Workspace: in.Current.Workspace,
				},
				AsOfSeq: res.AsOfSeq,
			}
			if again := mustDerive(t, replay); !reflect.DeepEqual(again, res) {
				t.Error("the replay differs")
			}
		})
	}
}

// MVP-ST-11 over random histories: the --diff-since projection agrees with
// the full derivations at both cutoffs, with the same S and settings (the
// start with the observations of its own source set).
func TestChangesSinceAgreesWithTheDerivations(t *testing.T) {
	for seed := range uint64(propertySeeds / 4) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			c := legalCase(t, seed, randomOptions(seed))
			rng := rand.New(rand.NewPCG(seed, 11))
			since := rng.Int64N(c.AsOf + 1)
			in := toderive.Input(c.Dump, c.Current, c.AsOf)
			changes, err := derive.ChangesSince(in, since)
			if err != nil {
				t.Fatal(err)
			}
			now := mustDerive(t, in)
			startCur := c.Current
			startCur.Observations = factgen.ObservationsAt(c.Dump, since)
			start := mustDerive(t, toderive.Input(c.Dump, startCur, since))
			type sp struct{ status, progress string }
			before := map[string]sp{}
			for _, r := range start.Requirements {
				for _, cr := range r.Criteria {
					before[cr.Key] = sp{string(cr.Status), string(cr.Progress)}
				}
			}
			var want []derive.Change
			for _, r := range now.Requirements {
				for _, cr := range r.Criteria {
					b := before[cr.Key]
					if b.status != string(cr.Status) {
						want = append(want, derive.Change{Criterion: cr.Key, Field: "delivery_status", From: b.status, To: string(cr.Status)})
					}
					if b.progress != string(cr.Progress) {
						want = append(want, derive.Change{Criterion: cr.Key, Field: "progress", From: b.progress, To: string(cr.Progress)})
					}
				}
			}
			// Requirements and criteria are listed in key order, and a
			// criterion's key starts with its requirement's number, so want
			// is in the documented order already.
			if !reflect.DeepEqual(changes, want) {
				t.Errorf("changes since %d = %+v\nwant %+v", since, changes, want)
			}
		})
	}
}

// F5 input domain, data-model §6.13: settings never change a delivery
// status or whether a blocking gap exists, its type or whether it blocks
// (they may change reason codes, so gap objects are not compared).
func TestSettingsNeverChangeDelivery(t *testing.T) {
	for seed := range uint64(propertySeeds / 4) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			c := legalCase(t, seed, randomOptions(seed))
			base := mustDerive(t, toderive.Input(c.Dump, c.Current, c.AsOf))
			for _, stage := range []string{"docs_only", "in_development", "incomplete_docs"} {
				for _, skips := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
					cur := c.Current
					cur.Stage, cur.SourcesSkipped, cur.InventorySkipped = stage, skips[0], skips[1]
					res := mustDerive(t, toderive.Input(c.Dump, cur, c.AsOf))
					if res.Delivery.Status != base.Delivery.Status {
						t.Errorf("%s %v: delivery %s, was %s", stage, skips, res.Delivery.Status, base.Delivery.Status)
					}
					if a, b := blockingKinds(res), blockingKinds(base); !reflect.DeepEqual(a, b) {
						t.Errorf("%s %v: blocking %v, was %v", stage, skips, a, b)
					}
					for i, r := range res.Requirements {
						for j, cr := range r.Criteria {
							if cr.Status != base.Requirements[i].Criteria[j].Status {
								t.Errorf("%s %v: %s is %s, was %s", stage, skips, cr.Key, cr.Status, base.Requirements[i].Criteria[j].Status)
							}
						}
					}
				}
			}
		})
	}
}

func blockingKinds(res *derive.Result) []string {
	var out []string
	for _, g := range res.Delivery.BlockingGaps {
		out = append(out, string(g.Type)+" "+g.TargetKey)
	}
	return out
}

// Malformed input, option (a) (data-model §6.14): no business result is
// promised, but the derivation neither panics nor varies: the same input
// gives the same result, in any row order.
func TestMalformedInputsAreDeterministic(t *testing.T) {
	for seed := range uint64(propertySeeds / 2) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			// Malformed input is derived from a legal history broken on
			// purpose; it is not counted in the legal domain.
			c := legalCase(t, seed, randomOptions(seed))
			rng := rand.New(rand.NewPCG(seed, 13))
			in := corrupt(toderive.Input(c.Dump, c.Current, c.AsOf), rng)
			first, err1 := derive.Derive(in)
			second, err2 := derive.Derive(shuffled(in, rng))
			if (err1 == nil) != (err2 == nil) || !reflect.DeepEqual(first, second) {
				t.Errorf("the results differ: %v / %v", err1, err2)
			}
			ch1, e1 := derive.ChangesSince(in, in.AsOfSeq/2)
			ch2, e2 := derive.ChangesSince(shuffled(in, rng), in.AsOfSeq/2)
			if (e1 == nil) != (e2 == nil) || !reflect.DeepEqual(ch1, ch2) {
				t.Errorf("the changes differ: %v / %v", e1, e2)
			}
		})
	}
}

// corrupt breaks write preconditions and fact invariants the production
// writes guarantee: a duplicated seq in a column unique per table,
// decisions on evidence of another criterion or snapshot, dangling links,
// a normal acceptance without a pass, unknown enum values.
func corrupt(in derive.Input, rng *rand.Rand) derive.Input {
	out := deepCopyFacts(in)
	f := &out.Facts
	pick := func(n int) int { return rng.IntN(n) }
	for range 1 + pick(6) {
		switch pick(10) {
		case 0:
			if n := len(f.Decisions); n > 1 {
				f.Decisions[pick(n)].Seq = f.Decisions[pick(n)].Seq
			}
		case 1:
			if n, m := len(f.Decisions), len(f.CheckRuns); n > 0 && m > 0 {
				d := &f.Decisions[pick(n)]
				d.Decision, d.Mode, d.EvidenceID = "accepted", "normal", f.CheckRuns[pick(m)].ID
			}
		case 2:
			if n := len(f.Decisions); n > 0 {
				f.Decisions[pick(n)].CriterionID = "01JAAAAAAADANGL1NG00000000"
			}
		case 3:
			if n := len(f.BaselineChanges); n > 0 {
				c := &f.BaselineChanges[pick(n)]
				c.After.Status, c.After.Revision = "unknown", -1
			}
		case 4:
			if n := len(f.Entities); n > 0 {
				f.Entities[pick(n)].RequirementID = "01JAAAAAAAN0PARENT00000000"
			}
		case 5:
			if n := len(f.CheckRuns); n > 0 {
				r := &f.CheckRuns[pick(n)]
				r.Result, r.EvidenceGrade = "maybe", "unverified"
			}
		case 6:
			if n := len(f.ScopeConfirmations); n > 1 {
				f.ScopeConfirmations[pick(n)].Seq = f.ScopeConfirmations[pick(n)].Seq
			}
		case 7:
			f.Targets = append(f.Targets, derive.Target{ID: "01JAAAAAAASEC0NDTARGET0000", Name: "second", Ref: "refs/heads/x", CreatedSeq: 1})
		case 8:
			if n := len(f.Questions); n > 0 {
				q := &f.Questions[pick(n)]
				q.ProposalItemID, q.RelatedID = "01JAAAAAAAN0ITEM0000000000", "01JAAAAAAAN0ENTITY00000000"
			}
		default:
			if n := len(f.Claims); n > 0 {
				f.Claims[pick(n)].Assertion = "certainly"
			}
		}
	}
	return out
}
