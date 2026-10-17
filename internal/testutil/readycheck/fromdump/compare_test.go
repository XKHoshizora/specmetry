package fromdump_test

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
	"github.com/XKHoshizora/specmetry/internal/testutil/readycheck"
	"github.com/XKHoshizora/specmetry/internal/testutil/readycheck/fromdump"
)

// The comparison is the connection layer of F5: it hands the same dump and
// the same current-value record to each implementation's own converter and
// compares only their conclusions. Nothing selected, filtered or computed
// by one side reaches the other.

const compareSeeds = 600

// legalCase is a random history package legality confirms is legal: the
// comparison runs on legal histories only (data-model §6.14, F5), and a
// generated one that is not fails with its operation trace.
func legalCase(t *testing.T, seed uint64, o factgen.Options) factgen.Case {
	t.Helper()
	c := factgen.Random(seed, o)
	if p := legality.Check(c.Dump, legality.Options{Isolated: o.Isolated}); len(p) > 0 {
		t.Fatalf("seed %d is not a legal history:\n%s", seed, legality.Report(p, c.History.Trace()))
	}
	return c
}

func engineReady(t *testing.T, c factgen.Case, cur factgen.Current) (bool, *derive.Result) {
	t.Helper()
	res, err := derive.Derive(toderive.Input(c.Dump, cur, c.AsOf))
	if err != nil {
		t.Fatalf("the engine refused a legal history: %v", err)
	}
	return res.Delivery.Status == derive.TargetReady, res
}

func checkerVerdict(c factgen.Case, cur factgen.Current) readycheck.Verdict {
	return readycheck.Check(fromdump.Facts(c.Dump), fromdump.Current(cur), c.AsOf)
}

// F5's two-way comparison over legal histories, in the MVP domain (observed
// runs only) and the general one (with synthetic isolated runs): the
// engine says ready if and only if every condition holds, and when ready
// the delivery grades agree (R10). Each history is also compared with other
// settings: R1 to R10 read none, so the engine's verdict must not change.
func TestEngineAndCheckerAgree(t *testing.T) {
	counts := map[string]int{}
	failures := map[string]map[readycheck.Condition]int{"MVP": {}, "general": {}}
	for seed := range uint64(compareSeeds) {
		general := seed%2 == 1
		domain := map[bool]string{false: "MVP", true: "general"}[general]
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			c := legalCase(t, seed, factgen.Options{Steps: 40, Deliver: 0.55, Cutoff: 0.15, Isolated: general})
			ready, res := engineReady(t, c, c.Current)
			v := checkerVerdict(c, c.Current)
			if ready != v.Ready() {
				t.Fatalf("engine ready = %v (delivery %s, blocking %v); checker failed %v",
					ready, res.Delivery.Status, res.Delivery.BlockingGaps, v.Failed)
			}
			if err := readycheck.Consistent(v, ready, string(res.Delivery.EvidenceGrade)); err != nil {
				t.Fatal(err)
			}
			// Review 046 N2: the grade of every output, ready or not.
			if err := readycheck.GradeConsistent(v, string(res.Delivery.EvidenceGrade)); err != nil {
				t.Fatalf("ready = %v: %v", ready, err)
			}
			if ready {
				counts[domain+": ready, "+string(res.Delivery.EvidenceGrade)]++
			} else {
				counts[domain+": not ready"]++
				for cond := range v.Failed {
					failures[domain][cond]++
				}
			}
			for _, settings := range [][3]any{{"docs_only", true, false}, {"incomplete_docs", false, true}} {
				cur := c.Current
				cur.Stage, cur.SourcesSkipped, cur.InventorySkipped = settings[0].(string), settings[1].(bool), settings[2].(bool)
				if other, _ := engineReady(t, c, cur); other != ready || checkerVerdict(c, cur).Ready() != ready {
					t.Errorf("settings %v changed the verdict", settings)
				}
			}
		})
	}
	t.Logf("legal histories: %v", counts)
	t.Logf("failed conditions of those not ready: MVP %v; general %v", failures["MVP"], failures["general"])
	// The legal histories that cannot be ready fail every condition, in
	// each domain (R1 at cutoffs before the first baseline version).
	for _, domain := range []string{"MVP", "general"} {
		for _, cond := range []readycheck.Condition{readycheck.R1, readycheck.R2, readycheck.R3, readycheck.R4, readycheck.R5,
			readycheck.R6, readycheck.R7, readycheck.R8, readycheck.R9} {
			if failures[domain][cond] == 0 {
				t.Errorf("%s: no history failed %s: %v", domain, cond, failures[domain])
			}
		}
	}
	// Both verdicts in each domain, and the grades each one can have; no
	// isolated grade in the MVP domain.
	for _, want := range []string{"MVP: not ready", "MVP: ready, observed", "MVP: ready, none",
		"general: not ready", "general: ready, snapshot_isolated"} {
		if counts[want] < 10 {
			t.Errorf("the comparison would run idle: too few %q in %v", want, counts)
		}
	}
	if counts["MVP: ready, snapshot_isolated"] > 0 {
		t.Errorf("an isolated grade in the MVP domain: %v", counts)
	}
}

// Malformed input, option (a): neither implementation crashes and each is
// deterministic; the checker gives its own verdict, and the two are not
// compared (data-model §6.14).
func TestMalformedInputsDoNotCrashEither(t *testing.T) {
	for seed := range uint64(200) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			// A legal history broken on purpose; not counted in the legal
			// domain.
			c := legalCase(t, seed, factgen.Options{Steps: 40, Deliver: 0.5, Isolated: seed%2 == 1})
			rng := rand.New(rand.NewPCG(seed, 17))
			corruptDump(&c.Dump, rng)
			r1, e1 := derive.Derive(toderive.Input(c.Dump, c.Current, c.AsOf))
			r2, e2 := derive.Derive(toderive.Input(c.Dump, c.Current, c.AsOf))
			if (e1 == nil) != (e2 == nil) || !reflect.DeepEqual(r1, r2) {
				t.Error("the engine is not deterministic")
			}
			f := fromdump.Facts(c.Dump)
			cur := fromdump.Current(c.Current)
			v1 := readycheck.Check(f, cur, c.AsOf)
			shuffleFacts(&f, rng)
			if v2 := readycheck.Check(f, cur, c.AsOf); !reflect.DeepEqual(v1, v2) {
				t.Errorf("the checker depends on the row order: %v / %v", v1, v2)
			}
		})
	}
}

// corruptDump breaks write preconditions and fact invariants in the rows.
func corruptDump(d *factgen.Dump, rng *rand.Rand) {
	pick := func(n int) int { return rng.IntN(n) }
	str := func(s string) *string { return &s }
	for range 1 + pick(5) {
		switch pick(8) {
		case 0:
			if n := len(d.Decisions); n > 1 {
				d.Decisions[pick(n)].Seq = d.Decisions[pick(n)].Seq
			}
		case 1:
			if n, m := len(d.Decisions), len(d.CheckRuns); n > 0 && m > 0 {
				x := &d.Decisions[pick(n)]
				x.Decision, x.Mode, x.EvidenceID = "accepted", str("normal"), str(d.CheckRuns[pick(m)].ID)
			}
		case 2:
			if n := len(d.Decisions); n > 0 {
				d.Decisions[pick(n)].CriterionID = "01JAAAAAAADANGL1NG00000000"
			}
		case 3:
			if n := len(d.CheckRuns); n > 0 {
				d.CheckRuns[pick(n)].Result = "maybe"
			}
		case 4:
			if n := len(d.ScopeItems); n > 0 {
				d.ScopeItems[pick(n)].RequirementID = "01JAAAAAAAN0REQ00000000000"
			}
		case 5:
			d.DeliveryTargets = append(d.DeliveryTargets, factgen.TargetRow{ID: "01JAAAAAAASEC0ND00000000T0", Name: "x", Ref: "refs/heads/x", CreatedSeq: 1})
		case 6:
			if n := len(d.Questions); n > 0 {
				d.Questions[pick(n)].ProposalItemID = str("01JAAAAAAAN0ITEM0000000000")
			}
		default:
			if n := len(d.BaselineChanges); n > 0 {
				d.BaselineChanges[pick(n)].Seq = 0
			}
		}
	}
}

func shuffleFacts(f *readycheck.Facts, rng *rand.Rand) {
	sh := func(n int, swap func(i, j int)) { rng.Shuffle(n, swap) }
	sh(len(f.Targets), func(i, j int) { f.Targets[i], f.Targets[j] = f.Targets[j], f.Targets[i] })
	sh(len(f.Versions), func(i, j int) { f.Versions[i], f.Versions[j] = f.Versions[j], f.Versions[i] })
	sh(len(f.Entities), func(i, j int) { f.Entities[i], f.Entities[j] = f.Entities[j], f.Entities[i] })
	sh(len(f.Changes), func(i, j int) { f.Changes[i], f.Changes[j] = f.Changes[j], f.Changes[i] })
	sh(len(f.Scopes), func(i, j int) { f.Scopes[i], f.Scopes[j] = f.Scopes[j], f.Scopes[i] })
	sh(len(f.ScopeItems), func(i, j int) { f.ScopeItems[i], f.ScopeItems[j] = f.ScopeItems[j], f.ScopeItems[i] })
	sh(len(f.Sources), func(i, j int) { f.Sources[i], f.Sources[j] = f.Sources[j], f.Sources[i] })
	sh(len(f.Acks), func(i, j int) { f.Acks[i], f.Acks[j] = f.Acks[j], f.Acks[i] })
	sh(len(f.Items), func(i, j int) { f.Items[i], f.Items[j] = f.Items[j], f.Items[i] })
	sh(len(f.Questions), func(i, j int) { f.Questions[i], f.Questions[j] = f.Questions[j], f.Questions[i] })
	sh(len(f.Runs), func(i, j int) { f.Runs[i], f.Runs[j] = f.Runs[j], f.Runs[i] })
	sh(len(f.Decisions), func(i, j int) { f.Decisions[i], f.Decisions[j] = f.Decisions[j], f.Decisions[i] })
}
