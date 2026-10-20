package claim

// Receipt is the claim-receipt.v1 document (WP2.4 spec §9).
type Receipt struct {
	Schema       string            `json:"schema"`
	BatchID      string            `json:"batch_id"`
	Seq          int64             `json:"seq"`
	Kind         string            `json:"kind"`
	BaseVersion  int               `json:"base_version"`
	Context      CodeContext       `json:"context"`
	CompareFrom  *CompareFrom      `json:"compare_from,omitempty"`
	Claims       []ReceiptClaim    `json:"claims"`
	UnmappedWork []ReceiptUnmapped `json:"unmapped_work"`
}

// CodeContext is the code context the CLI captured (common.v1
// codeContext); Head is null outside git or before the first commit.
type CodeContext struct {
	WorktreeID string  `json:"worktree_id"`
	Head       *string `json:"head"`
	TreeState  string  `json:"tree_state"`
}

// CompareFrom is a sync batch's comparison start: a commit, or null with
// source none.
type CompareFrom struct {
	Commit *string `json:"commit"`
	Source string  `json:"source"`
}

// ReceiptClaim is one stored claim, at its index in the input.
type ReceiptClaim struct {
	Index          int         `json:"index"`
	ClaimID        string      `json:"claim_id"`
	Target         string      `json:"target"`
	TargetRevision int         `json:"target_revision"`
	Assertion      string      `json:"assertion"`
	Refs           ReceiptRefs `json:"refs"`
}

// ReceiptUnmapped is one stored unmapped work item, at its index.
type ReceiptUnmapped struct {
	Index   int         `json:"index"`
	ClaimID string      `json:"claim_id"`
	Refs    ReceiptRefs `json:"refs"`
}

// ReceiptRefs are the stored reference results without checked_head (the
// context's head), and the number of test names.
type ReceiptRefs struct {
	Files   []ReceiptFile  `json:"files"`
	Commits []CommitResult `json:"commits"`
	Tests   int            `json:"tests"`
}

// ReceiptFile is a file reference's result.
type ReceiptFile struct {
	Path   string  `json:"path"`
	Lines  *string `json:"lines"`
	Exists bool    `json:"exists"`
}

// CommitResult is a commit reference's result: resolved is false for the
// unknown_commit mark (MVP-INV-02, E4).
type CommitResult struct {
	Commit   string `json:"commit"`
	Resolved bool   `json:"resolved"`
}
