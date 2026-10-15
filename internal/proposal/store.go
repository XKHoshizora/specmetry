package proposal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/source"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// CheckedSource is a source reference as checked and stored: under
// hash_only its locator and the excerpt's hash, never the excerpt (data-
// model §4.5).
type CheckedSource struct {
	SourceID      string       `json:"source_id"`
	Locator       *string      `json:"locator"`
	ExcerptSHA256 string       `json:"excerpt_sha256"`
	Verification  Verification `json:"verification"`
	// SourceSHA256 is the hash of the source file the excerpt was checked
	// against; null when it could not be read.
	SourceSHA256 *string `json:"source_sha256"`
}

// Checked holds the checked references of each item and question.
type Checked struct {
	Items     [][]CheckedSource
	Questions [][]CheckedSource
	CodeRefs  [][]CheckedCodeRef
}

// ResolveSources finds the registered source each reference names, by its
// stored path or its ID (V-P06). A removed or unknown source is refused;
// all such references are reported.
func ResolveSources(ctx context.Context, tx *sql.Tx, projectID string, p *Proposal) (map[string]source.Source, error) {
	found := map[string]source.Source{}
	var ps problems
	resolve := func(refs []SourceRef, pointer string) error {
		for k, r := range refs {
			if _, ok := found[r.Source]; ok {
				continue
			}
			s, ok, err := source.FindActive(ctx, tx, projectID, r.Source)
			if err != nil {
				return err
			}
			if ok {
				found[r.Source] = s
				continue
			}
			removed, err := source.FindRemoved(ctx, tx, projectID, r.Source)
			if err != nil {
				return err
			}
			at := pointer + "/" + strconv.Itoa(k) + "/source"
			if removed {
				ps.add(at, "names a removed source; only registered, active sources can be cited")
			} else {
				ps.add(at, "names no registered source of this project (give its path as registered, or its ID)")
			}
		}
		return nil
	}
	for i, it := range p.Items {
		if err := resolve(it.Sources, itemPointer(i, "sources")); err != nil {
			return nil, err
		}
	}
	for i, q := range p.Questions {
		if err := resolve(q.Sources, "/questions/"+strconv.Itoa(i)+"/sources"); err != nil {
			return nil, err
		}
	}
	return found, ps.err()
}

// CheckSources checks every excerpt against its source, reading each
// source file once (V-P07). root is the project root, for in-repository
// sources.
func CheckSources(ctx context.Context, root string, p *Proposal, sources map[string]source.Source) (Checked, error) {
	type key struct{ id, excerpt string }
	wanted := map[string][]string{}
	seen := map[key]bool{}
	collect := func(refs []SourceRef) {
		for _, r := range refs {
			s := sources[r.Source]
			if k := (key{s.ID, r.Excerpt}); !seen[k] {
				seen[k] = true
				wanted[s.ID] = append(wanted[s.ID], r.Excerpt)
			}
		}
	}
	for _, it := range p.Items {
		collect(it.Sources)
	}
	for _, q := range p.Questions {
		collect(q.Sources)
	}
	type result struct {
		verification Verification
		sha          *string
	}
	results := map[key]result{}
	for id, excerpts := range wanted {
		var s source.Source
		for _, src := range sources {
			if src.ID == id {
				s = src
				break
			}
		}
		sum, verifications, err := CheckExcerpts(ctx, s.File(root), excerpts)
		if err != nil {
			return Checked{}, err
		}
		var sha *string
		if sum != "" {
			sha = &sum
		}
		for i, e := range excerpts {
			results[key{id, e}] = result{verifications[i], sha}
		}
	}
	checked := func(refs []SourceRef) []CheckedSource {
		out := make([]CheckedSource, len(refs))
		for k, r := range refs {
			s := sources[r.Source]
			res := results[key{s.ID, r.Excerpt}]
			out[k] = CheckedSource{SourceID: s.ID, Locator: r.Locator, ExcerptSHA256: digest(r.Excerpt),
				Verification: res.verification, SourceSHA256: res.sha}
		}
		return out
	}
	var c Checked
	for _, it := range p.Items {
		c.Items = append(c.Items, checked(it.Sources))
	}
	for _, q := range p.Questions {
		c.Questions = append(c.Questions, checked(q.Sources))
	}
	return c, nil
}

// digest is the excerpt's sha256: of its text exactly as submitted.
func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Submission is a proposal checked outside the database, ready to store.
type Submission struct {
	Proposal *Proposal
	Checked  Checked
	Where    project.Where
	Actor    string
	Channel  string
}

// Receipt is the proposal-receipt.v1 document.
type Receipt struct {
	Schema         string            `json:"schema"`
	ProposalID     string            `json:"proposal_id"`
	Seq            int64             `json:"seq"`
	Origin         string            `json:"origin"`
	BaseVersion    int               `json:"base_version"`
	AppliedVersion *int              `json:"applied_version"`
	Items          []ReceiptItem     `json:"items"`
	Questions      []ReceiptQuestion `json:"questions"`
}

// ReceiptItem is one stored item.
type ReceiptItem struct {
	ItemID      string           `json:"item_id"`
	ItemIndex   int              `json:"item_index"`
	Op          string           `json:"op"`
	Type        string           `json:"type"`
	Ref         *string          `json:"ref"`
	Target      *string          `json:"target"`
	ReviewState string           `json:"review_state"`
	AssignedKey *string          `json:"assigned_key"`
	Sources     []ReceiptSource  `json:"sources"`
	CodeRefs    []ReceiptCodeRef `json:"code_refs"`
}

// ReceiptSource is what checking an excerpt showed; never the excerpt.
type ReceiptSource struct {
	SourceID     string       `json:"source_id"`
	Verification Verification `json:"verification"`
}

// ReceiptCodeRef is whether a code reference's file is there.
type ReceiptCodeRef struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// ReceiptQuestion is one opened question.
type ReceiptQuestion struct {
	Key      string  `json:"key"`
	About    *string `json:"about"`
	Blocking bool    `json:"blocking"`
}

// Store checks the proposal against the project's baseline and stores it,
// in the caller's write transaction. A stale base_version or
// target_revision is stale_precondition (exit code 5, V-P02) with the
// current values; a key that names no active entity of the right type, or
// a check the target criterion cannot take, is invalid input (exit code 2,
// V-P03, V-P04). The proposal is then stored with one seq: its items
// pending, its questions open with their keys, and the questions' source
// references (hash_only: no excerpt text). The baseline does not change.
func Store(ctx context.Context, tx *sql.Tx, projectID string, s Submission) (Receipt, error) {
	p := s.Proposal
	current, err := baseline.CurrentVersion(ctx, tx, projectID)
	if err != nil {
		return Receipt{}, err
	}
	if current != p.BaseVersion {
		e := apperr.Newf(apperr.StalePrecondition,
			"the baseline is at version %d, not %d; read it again (specmetry status --json) and propose against it", current, p.BaseVersion).
			WithDetails(apperr.Detail{Pointer: "/base_version", Problem: fmt.Sprintf("is stale: the current baseline version is %d", current)})
		e.Current = map[string]any{"base_version": current}
		return Receipt{}, e
	}

	var ps problems
	entities := map[string]baseline.Entity{}
	find := func(pointer, key string, types ...baseline.Type) (baseline.Entity, bool, error) {
		e, ok, err := baseline.Find(ctx, tx, projectID, key)
		if err != nil {
			return e, false, err
		}
		allowed := false
		for _, t := range types {
			allowed = allowed || e.Type == t
		}
		switch {
		case !ok:
			ps.add(pointer, "names no %s of the baseline", joinTypes(types))
		case !allowed:
			ps.add(pointer, "names a %s; it must name a %s", e.Type, joinTypes(types))
		case !e.Active:
			ps.add(pointer, "names a retired %s; only active entities can be changed or referred to", e.Type)
		default:
			entities[key] = e
			return e, true, nil
		}
		return e, false, nil
	}
	type staleTarget struct {
		Pointer  string `json:"pointer"`
		Target   string `json:"target"`
		Revision int    `json:"revision"`
	}
	var stale []staleTarget
	for i, it := range p.Items {
		if it.Op != "add" {
			e, ok, err := find(itemPointer(i, "target"), it.Target, baseline.Type(it.Type))
			if err != nil {
				return Receipt{}, err
			}
			if ok && e.Revision != it.TargetRevision {
				stale = append(stale, staleTarget{itemPointer(i, "target_revision"), it.Target, e.Revision})
			}
			if ok && it.Op == "update" && it.Type == "criterion" {
				criterionCheck(i, it, e, &ps)
			}
		}
		if goal, ok := stringData(it.Data, "goal"); ok && !IsLocalRef(goal) {
			if _, _, err := find(itemPointer(i, "data", "goal"), goal, baseline.Goal); err != nil {
				return Receipt{}, err
			}
		}
		if req, ok := stringData(it.Data, "requirement"); ok && !IsLocalRef(req) {
			if _, _, err := find(itemPointer(i, "data", "requirement"), req, baseline.Requirement); err != nil {
				return Receipt{}, err
			}
		}
	}
	for i, q := range p.Questions {
		if q.About != "" && !IsLocalRef(q.About) {
			if _, _, err := find("/questions/"+strconv.Itoa(i)+"/about", q.About, baseline.Requirement, baseline.Criterion); err != nil {
				return Receipt{}, err
			}
		}
	}
	if err := ps.err(); err != nil {
		return Receipt{}, err
	}
	if len(stale) > 0 {
		e := apperr.New(apperr.StalePrecondition, "a target has changed since it was read; read the baseline again and propose against it")
		for _, st := range stale {
			e.Details = append(e.Details, apperr.Detail{Pointer: st.Pointer,
				Problem: fmt.Sprintf("is stale: %s is at revision %d", st.Target, st.Revision)})
		}
		e.Current = map[string]any{"base_version": current, "targets": stale}
		return Receipt{}, e
	}
	// A source may have been removed since it was resolved.
	for _, refs := range append(append([][]CheckedSource{}, s.Checked.Items...), s.Checked.Questions...) {
		for _, r := range refs {
			var active int
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sources WHERE id = ? AND removed_seq IS NULL`, r.SourceID).Scan(&active); err != nil {
				return Receipt{}, err
			}
			if active == 0 {
				return Receipt{}, apperr.New(apperr.InvalidInput, "a cited source was removed meanwhile; nothing was written")
			}
		}
	}

	worktree, err := project.EnsureWorktree(ctx, tx, projectID, s.Where, time.Now())
	if err != nil {
		return Receipt{}, err
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return Receipt{}, err
	}
	r := Receipt{Schema: "specmetry.proposal-receipt/v1", ProposalID: ulid.New(), Seq: seq, Origin: p.Origin,
		BaseVersion: p.BaseVersion, Items: []ReceiptItem{}, Questions: []ReceiptQuestion{}}
	if _, err := tx.ExecContext(ctx, `INSERT INTO proposals (id, seq, project_id, origin, actor, channel, worktree_id, summary, base_version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, r.ProposalID, seq, projectID, p.Origin, s.Actor, s.Channel, worktree, p.Summary, p.BaseVersion); err != nil {
		return Receipt{}, err
	}
	itemIDs := map[string]string{}
	for i, it := range p.Items {
		ri, err := storeItem(ctx, tx, projectID, r.ProposalID, i, it, s.Checked.Items[i], s.Checked.CodeRefs[i], entities)
		if err != nil {
			return Receipt{}, err
		}
		if it.Op == "add" {
			itemIDs[it.Ref] = ri.ItemID
		}
		r.Items = append(r.Items, ri)
	}
	for i, q := range p.Questions {
		rq, err := storeQuestion(ctx, tx, projectID, r.ProposalID, seq, p.Origin, q, s.Checked.Questions[i], entities, itemIDs)
		if err != nil {
			return Receipt{}, err
		}
		r.Questions = append(r.Questions, rq)
	}
	return r, nil
}

func joinTypes(types []baseline.Type) string {
	out := ""
	for i, t := range types {
		if i > 0 {
			out += " or "
		}
		out += string(t)
	}
	return out
}

// criterionCheck refuses an update that would leave a criterion with a
// check but not automated, or automated by data only in name (V-P04;
// data-model §4.4): a check needs the criterion to be automated, and
// making it manual needs its check removed.
func criterionCheck(i int, it Item, e baseline.Entity, ps *problems) {
	c, hasCheckField := checkData(it.Data)
	verification, setsVerification := stringData(it.Data, "verification")
	if !setsVerification {
		verification = e.Verification
	}
	withCheck := e.HasCheck
	if hasCheckField {
		withCheck = c != nil
	}
	if withCheck && verification != "automated" {
		pointer := itemPointer(i, "data", "check")
		if !hasCheckField {
			pointer = itemPointer(i, "data", "verification")
		}
		ps.add(pointer, "a check needs an automated criterion; %s is %s", it.Target, verification)
	}
}

func storeItem(ctx context.Context, tx *sql.Tx, projectID, proposalID string, index int, it Item,
	sources []CheckedSource, codeRefs []CheckedCodeRef, entities map[string]baseline.Entity) (ReceiptItem, error) {
	ri := ReceiptItem{ItemID: ulid.New(), ItemIndex: index, Op: it.Op, Type: it.Type, ReviewState: "pending",
		Sources: []ReceiptSource{}, CodeRefs: []ReceiptCodeRef{}}
	payload := []byte("null")
	if it.Data != nil {
		var err error
		if payload, err = json.Marshal(it.Data); err != nil {
			return ri, err
		}
	}
	sourcesJSON, err := json.Marshal(sources)
	if err != nil {
		return ri, err
	}
	codeRefsJSON, err := json.Marshal(codeRefs)
	if err != nil {
		return ri, err
	}
	var target, targetRevision, ref, suggested any
	if it.Op == "add" {
		ref = it.Ref
		r := it.Ref
		ri.Ref = &r
	} else {
		target, targetRevision = entities[it.Target].ID, it.TargetRevision
		t := it.Target
		ri.Target = &t
	}
	if it.Op == "update" && it.ChangeClass != "" {
		suggested = it.ChangeClass
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO proposal_items (id, proposal_id, project_id, item_index, op, entity_type, target_id,
    target_revision, ref, payload_json, suggested_change_class, sources_json, code_refs_json, rationale, origin, confidence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ri.ItemID, proposalID, projectID, index, it.Op, it.Type, target, targetRevision, ref, string(payload), suggested,
		string(sourcesJSON), string(codeRefsJSON), nullable(it.Rationale), nullable(it.Origin), nullable(it.Confidence)); err != nil {
		return ri, err
	}
	for _, s := range sources {
		ri.Sources = append(ri.Sources, ReceiptSource{SourceID: s.SourceID, Verification: s.Verification})
	}
	for _, c := range codeRefs {
		ri.CodeRefs = append(ri.CodeRefs, ReceiptCodeRef{Path: c.Path, Exists: c.Exists})
	}
	return ri, nil
}

func storeQuestion(ctx context.Context, tx *sql.Tx, projectID, proposalID string, seq int64, origin string, q Question,
	sources []CheckedSource, entities map[string]baseline.Entity, itemIDs map[string]string) (ReceiptQuestion, error) {
	key, err := baseline.NextKey(ctx, tx, projectID, baseline.Question, nil)
	if err != nil {
		return ReceiptQuestion{}, err
	}
	rq := ReceiptQuestion{Key: key, Blocking: q.Blocking}
	var relatedType, relatedID, itemID any
	if q.About != "" {
		about := q.About
		rq.About = &about
		if IsLocalRef(q.About) {
			itemID = itemIDs[q.About]
		} else {
			e := entities[q.About]
			relatedType, relatedID = string(e.Type), e.ID
		}
	}
	id := ulid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO questions (id, project_id, key, text, context, related_type, related_id, proposal_item_id,
    blocking, origin, proposal_id, created_seq)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, projectID, key, q.Text, nullable(q.Context), relatedType, relatedID, itemID,
		q.Blocking, origin, proposalID, seq); err != nil {
		return rq, err
	}
	// hash_only: the reference keeps the excerpt's hash, never its text.
	for _, s := range sources {
		if _, err := tx.ExecContext(ctx, `INSERT INTO source_refs (id, seq, project_id, source_id, entity_type, entity_id, entity_revision,
    locator, excerpt, excerpt_sha256, verification, source_sha256)
VALUES (?, ?, ?, ?, 'question', ?, 1, ?, NULL, ?, ?, ?)`, ulid.New(), seq, projectID, s.SourceID, id, s.Locator, s.ExcerptSHA256,
			string(s.Verification), s.SourceSHA256); err != nil {
			return rq, err
		}
	}
	return rq, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
