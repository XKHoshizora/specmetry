// A test-only collector for the C-9 limited evidence plan
// (docs/wp2.5-c9-limited-evidence-plan.md; implementation record
// docs/wp2.5-c9-evidence-tool.md). It is its own module, so Specmetry's
// module does not contain it, cannot import it, and the root's ./... (the
// regular CI's go test, go vet and golangci-lint) does not reach it.
module github.com/XKHoshizora/specmetry/prototype/c9evidence

go 1.27

require golang.org/x/mod v0.42.0
