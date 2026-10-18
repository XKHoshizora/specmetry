// Package gitscan is the commit scan (M2 WP2.3; WP2.3 spec §5, §6): it
// reads every local branch and every worktree's detached HEAD through
// internal/gitx, finds the requirement and criterion keys commits mention,
// and stores the commits that mention at least one of the project's, with
// their links and the scan's cursors, in one write transaction. Only SHAs,
// times and links are kept: a commit's message is split into words in
// memory and dropped (architecture §10.2 item 6). It neither prints nor
// throttles, nor decides which commands scan (WP2.7).
package gitscan

import (
	"regexp"
	"slices"
	"strings"
)

// Target types and link sources, as commit_links stores them.
const (
	TargetRequirement = "requirement"
	TargetCriterion   = "criterion"

	SourceTrailer = "trailer"
	SourceMessage = "message"
)

// Mention is a key a commit mentions, of a requirement or a criterion, in
// a Specmetry-Ref trailer or else in its message.
type Mention struct {
	Key, Type, Source string
}

var (
	// The key formats of schemas/common.v1 (TD-14), exactly: not loosened.
	requirementKey = regexp.MustCompile(`^REQ-[0-9]{3,}$`)
	criterionKey   = regexp.MustCompile(`^AC-[0-9]{3,}-[1-9][0-9]*$`)
	// word is a longest run of ASCII letters, digits, "_" and "-";
	// anything else (white space, punctuation, CR, other scripts) ends one.
	word = regexp.MustCompile(`[A-Za-z0-9_-]+`)
)

// Mentions finds the keys that trailers (the Specmetry-Ref trailer values
// git gave) and message (the whole message, %B, whose trailer lines it
// holds too) mention, each once, sorted by key: a trailer mention when the
// trailer values have it, else a message mention (WP2.3 spec §5.3).
// Whether a key is one of the project's is for the write to decide.
func Mentions(trailers, message string) []Mention {
	found := map[string]Mention{}
	add := func(text, source string) {
		for _, w := range word.FindAllString(text, -1) {
			t := ""
			switch {
			case requirementKey.MatchString(w):
				t = TargetRequirement
			case criterionKey.MatchString(w):
				t = TargetCriterion
			default:
				continue
			}
			if _, seen := found[w]; !seen {
				found[w] = Mention{Key: w, Type: t, Source: source}
			}
		}
	}
	add(trailers, SourceTrailer)
	add(message, SourceMessage)
	out := make([]Mention, 0, len(found))
	for _, m := range found {
		out = append(out, m)
	}
	slices.SortFunc(out, func(a, b Mention) int { return strings.Compare(a.Key, b.Key) })
	if len(out) == 0 {
		return nil
	}
	return out
}
