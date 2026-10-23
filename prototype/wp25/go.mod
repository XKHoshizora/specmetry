// A test-only prototype of WP2.5's teardown mechanisms
// (docs/wp2.5-prototype.md). It is its own module, so Specmetry's module
// does not contain it, cannot import it, and its ./... does not test it.
module github.com/XKHoshizora/specmetry/prototype/wp25

go 1.27

require golang.org/x/sys v0.49.0
