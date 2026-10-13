//go:build !windows && !darwin

package datadir

// extendedACLProblems: outside macOS, ACLs cannot grant more than the mode
// bits allow (see checkPermissions).
func extendedACLProblems([]string) ([]string, error) { return nil, nil }
