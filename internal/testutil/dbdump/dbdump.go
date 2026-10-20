// Package dbdump reads one project's rows from a Specmetry database into a
// factgen.Dump, so that a test can hand the facts the commands wrote to
// the derivation (WP2.4 spec §12: "the written facts, as the derivation's
// input"), through factgen/toderive like any other dump. It only maps
// rows: JSON columns stay text and NULL is nil; it never selects a current
// record, filters by a cutoff or computes anything. A table that does not
// exist yet (check_runs and decisions before WP2.5 and WP2.6) has no rows.
package dbdump

import (
	"context"
	"database/sql"

	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
)

// Load reads project's rows inside tx.
func Load(ctx context.Context, tx *sql.Tx, project string) (factgen.Dump, error) {
	d := factgen.Dump{ProjectID: project}
	if err := tx.QueryRowContext(ctx, `SELECT seq_counter FROM meta WHERE id = 1`).Scan(&d.LatestSeq); err != nil {
		return d, err
	}
	each := func(query string, scan func(rs *sql.Rows) error) error {
		rs, err := tx.QueryContext(ctx, query, project)
		if err != nil {
			return err
		}
		defer func() { _ = rs.Close() }()
		for rs.Next() {
			if err := scan(rs); err != nil {
				return err
			}
		}
		return rs.Err()
	}
	steps := []func() error{
		func() error {
			return each(`SELECT id, name, ref, created_seq, archived_seq FROM delivery_targets WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.TargetRow
				err := rs.Scan(&r.ID, &r.Name, &r.Ref, &r.CreatedSeq, &r.ArchivedSeq)
				d.DeliveryTargets = append(d.DeliveryTargets, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq, number FROM baseline_versions WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.VersionRow
				err := rs.Scan(&r.ID, &r.Seq, &r.Number)
				d.BaselineVersions = append(d.BaselineVersions, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq, version_id, change_index, entity_type, entity_id, op, change_class, before_json, after_json,
    proposal_item_id, rationale FROM baseline_changes WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.ChangeRow
				err := rs.Scan(&r.ID, &r.Seq, &r.VersionID, &r.ChangeIndex, &r.EntityType, &r.EntityID, &r.Op, &r.ChangeClass, &r.BeforeJSON,
					&r.AfterJSON, &r.ProposalItemID, &r.Rationale)
				d.BaselineChanges = append(d.BaselineChanges, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, key, origin, status, revision, revision_seq FROM goals WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.EntityRow
				err := rs.Scan(&r.ID, &r.Key, &r.Origin, &r.Status, &r.Revision, &r.RevisionSeq)
				d.Goals = append(d.Goals, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, key, origin, status, revision, revision_seq FROM requirements WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.EntityRow
				err := rs.Scan(&r.ID, &r.Key, &r.Origin, &r.Status, &r.Revision, &r.RevisionSeq)
				d.Requirements = append(d.Requirements, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, key, origin, requirement_id, status, revision, revision_seq, requirement_revision, check_config_hash
FROM acceptance_criteria WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.EntityRow
				err := rs.Scan(&r.ID, &r.Key, &r.Origin, &r.RequirementID, &r.Status, &r.Revision, &r.RevisionSeq, &r.RequirementRevision, &r.CheckConfigHash)
				d.Criteria = append(d.Criteria, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq, target_id, baseline_version, source_set_hash FROM scope_confirmations WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.ScopeRow
				err := rs.Scan(&r.ID, &r.Seq, &r.TargetID, &r.BaselineVersion, &r.SourceSetHash)
				d.ScopeConfirmations = append(d.ScopeConfirmations, r)
				return err
			})
		},
		func() error {
			return each(`SELECT confirmation_id, requirement_id, requirement_revision FROM scope_items WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.ScopeItemRow
				err := rs.Scan(&r.ConfirmationID, &r.RequirementID, &r.RequirementRevision)
				d.ScopeItems = append(d.ScopeItems, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, path, kind, registered_seq, removed_seq, acked_sha256 FROM sources WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.SourceRow
				err := rs.Scan(&r.ID, &r.Path, &r.Kind, &r.RegisteredSeq, &r.RemovedSeq, &r.AckedSHA256)
				d.Sources = append(d.Sources, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq, source_id, previous_sha256, acked_sha256 FROM source_acks WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.SourceAckRow
				err := rs.Scan(&r.ID, &r.Seq, &r.SourceID, &r.PreviousSHA256, &r.AckedSHA256)
				d.SourceAcks = append(d.SourceAcks, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq, source_id, entity_type, entity_id, entity_revision, verification FROM source_refs WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.SourceRefRow
				err := rs.Scan(&r.ID, &r.Seq, &r.SourceID, &r.EntityType, &r.EntityID, &r.EntityRevision, &r.Verification)
				d.SourceRefs = append(d.SourceRefs, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq FROM proposals WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.ProposalRow
				err := rs.Scan(&r.ID, &r.Seq)
				d.Proposals = append(d.Proposals, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, proposal_id, item_index, review_state, reviewed_seq FROM proposal_items WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.ProposalItemRow
				err := rs.Scan(&r.ID, &r.ProposalID, &r.ItemIndex, &r.ReviewState, &r.ReviewedSeq)
				d.ProposalItems = append(d.ProposalItems, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, key, created_seq, answered_seq, status, blocking, related_type, related_id, proposal_item_id
FROM questions WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.QuestionRow
				err := rs.Scan(&r.ID, &r.Key, &r.CreatedSeq, &r.AnsweredSeq, &r.Status, &r.Blocking, &r.RelatedType, &r.RelatedID, &r.ProposalItemID)
				d.Questions = append(d.Questions, r)
				return err
			})
		},
		func() error {
			return each(`SELECT id, seq, kind, audit_scope_json FROM claim_batches WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.ClaimBatchRow
				err := rs.Scan(&r.ID, &r.Seq, &r.Kind, &r.AuditScopeJSON)
				d.ClaimBatches = append(d.ClaimBatches, r)
				return err
			})
		},
		func() error {
			// An unmapped item's NULL assertion is the dump's "" (factgen
			// writes unmapped work with the assertion its spec gives).
			return each(`SELECT id, seq, batch_id, target_type, criterion_id, criterion_revision, coalesce(assertion, '') FROM claims WHERE project_id = ?`,
				func(rs *sql.Rows) error {
					var r factgen.ClaimRow
					err := rs.Scan(&r.ID, &r.Seq, &r.BatchID, &r.TargetType, &r.CriterionID, &r.CriterionRevision, &r.Assertion)
					d.Claims = append(d.Claims, r)
					return err
				})
		},
		func() error {
			return each(`SELECT id, seq, sha FROM commits WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.CommitRow
				err := rs.Scan(&r.ID, &r.Seq, &r.SHA)
				d.Commits = append(d.Commits, r)
				return err
			})
		},
		func() error {
			return each(`SELECT commit_id, target_type, target_id, link_source FROM commit_links WHERE project_id = ?`, func(rs *sql.Rows) error {
				var r factgen.CommitLinkRow
				err := rs.Scan(&r.CommitID, &r.TargetType, &r.TargetID, &r.LinkSource)
				d.CommitLinks = append(d.CommitLinks, r)
				return err
			})
		},
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return d, err
		}
	}
	return d, nil
}
