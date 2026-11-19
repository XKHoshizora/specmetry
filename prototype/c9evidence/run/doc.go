// Package run is the formal entry of the C-9 limited evidence run (plan
// §4.1). Its TestMain is built only with the build tag c9evidence_run,
// which only the evidence workflow (.github/workflows/c9-evidence.yml) on a
// dedicated run branch passes; a plain `go test ./...` of this module never
// runs it. In phase 1 (review-resolution §190) the real invoker was
// stopped. In phase 2 (review-resolution §200) it is the real caller
// (internal/toolctl), but this entry is only built and vetted: the formal
// run is not authorized, and no run branch exists.
package run
