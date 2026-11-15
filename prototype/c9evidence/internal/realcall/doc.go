// Package realcall holds the real-call tests on fixture modules (LE-5 ②;
// phase 2, review-resolution §200; record docs/wp2.5-c9-evidence-phase2.md
// §6.3): the three reviewed calls, with their fixed arguments and the
// tool's environment, through the real control protocol, on a temporary
// fixture module, never on the repository R.
//
// The tests are built only with the tag c9evidence_realcall, and their
// TestMain refuses to run unless C9EVIDENCE_REALCALL=1, GITHUB_ACTIONS=true
// and RUNNER_ENVIRONMENT=github-hosted are all set: only the dedicated test
// workflow (.github/workflows/c9-tool-test.yml), on a GitHub-hosted runner
// that is destroyed after the job, runs them. A plain `go test ./...` of
// this module compiles only this file.
package realcall
