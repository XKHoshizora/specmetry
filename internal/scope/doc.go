package scope

// Doc is the target-scope.v1 document: what target show and scope show
// print with --json.
type Doc struct {
	Schema        string        `json:"schema"`
	Target        *TargetDoc    `json:"target"`
	Scope         *ScopeDoc     `json:"scope"`
	ConfirmInputs ConfirmInputs `json:"confirm_inputs"`
}

// TargetDoc is the active target, with the commit its ref resolves to now.
type TargetDoc struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Ref        string  `json:"ref"`
	Snapshot   *string `json:"snapshot"`
	CreatedSeq int64   `json:"created_seq"`
}

// ScopeDoc is the target's latest scope confirmation.
type ScopeDoc struct {
	ConfirmationID      string                 `json:"confirmation_id"`
	Seq                 int64                  `json:"seq"`
	BaselineVersion     int                    `json:"baseline_version"`
	SourceSetHash       string                 `json:"source_set_hash"`
	Requirements        []ConfirmedRequirement `json:"requirements"`
	Attestations        Attestations           `json:"attestations"`
	AcknowledgedWaivers []string               `json:"acknowledged_waivers"`
	Stale               bool                   `json:"stale"`
}

// ConfirmInputs are the current values a new confirmation must carry.
type ConfirmInputs struct {
	TargetID                *string  `json:"target_id"`
	SeenScopeConfirmationID *string  `json:"seen_scope_confirmation_id"`
	BaselineVersion         int      `json:"baseline_version"`
	SourceSetHash           string   `json:"source_set_hash"`
	PendingItems            int      `json:"pending_items"`
	MustRequirements        []string `json:"must_requirements"`
	EffectiveWaivers        []Waiver `json:"effective_waivers"`
}

// Document is st as target-scope.v1, with snapshot the commit the target's
// ref resolves to now ("" when it resolves to none).
func Document(st State, snapshot string) Doc {
	doc := Doc{Schema: "specmetry.target-scope/v1", ConfirmInputs: ConfirmInputs{
		BaselineVersion: st.BaselineVersion, SourceSetHash: st.SourceSetHash, PendingItems: st.PendingItems,
		MustRequirements: st.MustRequirements(), EffectiveWaivers: st.WaiversUnder(allKeys(st)),
	}}
	if doc.ConfirmInputs.MustRequirements == nil {
		doc.ConfirmInputs.MustRequirements = []string{}
	}
	if t := st.Target; t != nil {
		doc.Target = &TargetDoc{ID: t.ID, Name: t.Name, Ref: t.Ref, CreatedSeq: t.CreatedSeq}
		if snapshot != "" {
			doc.Target.Snapshot = &snapshot
		}
		id := t.ID
		doc.ConfirmInputs.TargetID = &id
	}
	if c := st.Latest; c != nil {
		waivers := c.AcknowledgedWaivers
		if waivers == nil {
			waivers = []string{}
		}
		doc.Scope = &ScopeDoc{ConfirmationID: c.ID, Seq: c.Seq, BaselineVersion: c.BaselineVersion, SourceSetHash: c.SourceSetHash,
			Requirements: c.Requirements, Attestations: c.Attestations, AcknowledgedWaivers: waivers, Stale: st.Stale()}
		id := c.ID
		doc.ConfirmInputs.SeenScopeConfirmationID = &id
	}
	return doc
}

// allKeys are the keys of every active requirement: confirm_inputs lists
// every waiver in effect on an active criterion.
func allKeys(st State) []string {
	keys := make([]string, 0, len(st.Requirements))
	for k := range st.Requirements {
		keys = append(keys, k)
	}
	return keys
}
