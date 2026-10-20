package cli

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/claim"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/dbdump"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// derived runs the derivation on the facts the commands wrote: every row
// of the project as dbdump reads it, through toderive, with the current
// values read where a command reads them (S from the target's ref, each
// active source's hash on disk, the stage). The engine is unchanged; this
// is its input, not a second implementation.
func derived(t *testing.T, home datadir.Dir, f *testutil.Fixture) *derive.Result {
	t.Helper()
	ctx := context.Background()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var d factgen.Dump
	var current factgen.Current
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		var project, ref string
		if err := tx.QueryRowContext(ctx, `SELECT p.id, p.stage, t.ref FROM projects p JOIN delivery_targets t ON t.project_id = p.id
WHERE t.archived_seq IS NULL`).Scan(&project, &current.Stage, &ref); err != nil {
			return err
		}
		current.S = f.Repo.Git("rev-parse", ref)
		var err error
		d, err = dbdump.Load(ctx, tx, project)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range d.Sources {
		if s.RemovedSeq != nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(f.Repo.Dir, filepath.FromSlash(s.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		current.Observations = append(current.Observations, factgen.Observation{SourceID: s.ID, SHA256: hex.EncodeToString(sum[:])})
	}
	res, err := derive.Derive(toderive.Input(d, current, d.LatestSeq))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func criterionView(t *testing.T, res *derive.Result, key string) derive.CriterionView {
	t.Helper()
	for _, r := range res.Requirements {
		for _, c := range r.Criteria {
			if c.Key == key {
				return c
			}
		}
	}
	t.Fatalf("no active criterion %s", key)
	return derive.CriterionView{}
}

// WP2.4 spec §12, the derivation of the facts claim writes (the engine and
// the reviewer are unchanged, WP2.1, WP2.2; the outputs are WP2.7's):
//   - MVP-CLM-01 and CX-12 (a): done on an automated criterion with a
//     check and no run is in progress, claimed_without_evidence, counted
//     in the metrics, and the delivery is not ready;
//   - CX-12 (e): done on a manual criterion is ready_for_review
//     (manual_claimed, not verification), the delivery not ready;
//   - CX-10: a partial audit leaves the inventory coverage below 1 and the
//     project not inventoried;
//   - MVP-INV-04: a full audit makes it 1, and inventoried.
//
// The project is in development (fx-in-dev): a docs-only project counts as
// inventoried whatever its audits (data-model §6.12).
func TestClaimFactsDerive(t *testing.T) {
	home, f, dir := claimsProjectOf(t, "fx-in-dev")
	if code, stdout, stderr := asHuman(t, dir, "v1", "scope", "confirm"); code != apperr.ExitOK {
		t.Fatalf("scope confirm: exit %d\n%s%s", code, stdout, stderr)
	}
	claimOK(t, home, dir, claimDoc(claim.Audit, 1, claimOn("AC-001-1", 1, "partial", nil)))
	res := derived(t, home, f)
	if res.InventoryCoverage == nil || *res.InventoryCoverage != (derive.Ratio{Num: 1, Den: 3}) || res.InitProgress == derive.Inventoried {
		t.Errorf("CX-10: coverage %+v, init progress %s", res.InventoryCoverage, res.InitProgress)
	}

	claimOK(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-002-1", 1, "done", nil), claimOn("AC-001-2", 1, "done", nil)))
	res = derived(t, home, f)
	auto := criterionView(t, res, "AC-002-1")
	if auto.Status != derive.CriterionInProgress || auto.Reason != "claimed_without_evidence" || !slices.Contains(auto.Flags, derive.FlagClaimedWithoutEvidence) ||
		res.Metrics.ClaimedWithoutEvidence != 1 {
		t.Errorf("CLM-01, CX-12 (a): %+v, metrics %+v", auto, res.Metrics)
	}
	manual := criterionView(t, res, "AC-001-2")
	if manual.Status != derive.CriterionReadyForReview || manual.Reason != "manual_claimed" {
		t.Errorf("CX-12 (e): %+v", manual)
	}
	if res.Delivery.Status == derive.TargetReady {
		t.Errorf("claims alone made the delivery ready: %+v", res.Delivery)
	}

	claimOK(t, home, dir, claimDoc(claim.Audit, 1, claimOn("AC-001-1", 1, "done", nil), claimOn("AC-001-2", 1, "done", nil),
		claimOn("AC-002-1", 1, "done", nil)))
	res = derived(t, home, f)
	if res.InventoryCoverage == nil || *res.InventoryCoverage != (derive.Ratio{Num: 3, Den: 3}) || res.InitProgress != derive.Inventoried {
		t.Errorf("INV-04: coverage %+v, init progress %s", res.InventoryCoverage, res.InitProgress)
	}
	if res.Delivery.Status == derive.TargetReady {
		t.Errorf("claims alone made the delivery ready: %+v", res.Delivery)
	}
}
