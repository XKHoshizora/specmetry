package sqlite

// SetReadOnlyHook gives the external test package (doctor through the CLI)
// access to readOnlyHook. Compiled only into this package's tests.
func SetReadOnlyHook(f func()) { readOnlyHook = f }
