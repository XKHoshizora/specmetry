// Package run is the formal entry of the C-9 limited evidence run (plan
// §4.1). Its TestMain is built only with the build tag c9evidence_run,
// which only the evidence workflow (.github/workflows/c9-evidence.yml) on a
// dedicated run branch passes; a plain `go test ./...` of this module never
// runs it. In phase 1 (review-resolution §190) the real invoker is stopped:
// E-10 (ii) is not confirmed, so no process is started.
package run
