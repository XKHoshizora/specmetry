// Package e2waitid is P-5's entry E2 (docs/wp2.5-prototype.md §16.3): the
// macOS waitid of libSystem, imported dynamically and called through the
// runtime's syscall.syscall6, as golang.org/x/sys calls libSystem's other
// functions. It is test-only: only the prototype's tests import it, and
// only on darwin; elsewhere this file is all there is, so that ./... does
// not fail on a package without a file to build.
package e2waitid
