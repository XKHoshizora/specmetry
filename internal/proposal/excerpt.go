package proposal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// Verification is what checking an excerpt against its source showed
// (V-P07). verified only means that the excerpt appears in the source; it
// says nothing about whether the item is right or complete (data-model
// §4.5).
type Verification string

// Verification results.
const (
	Verified      Verification = "verified"
	NotFound      Verification = "not_found"
	SourceMissing Verification = "source_missing"
	NotCheckable  Verification = "not_checkable"
)

// nonText are the document types whose files are not plain text, so an
// excerpt cannot be found in them verbatim (data-model §4.3).
var nonText = map[string]bool{".pdf": true, ".docx": true, ".doc": true, ".odt": true, ".rtf": true}

// Normalize drops every whitespace character (Unicode White_Space: spaces,
// tabs, line breaks including CRLF, the ideographic space). Two texts that
// differ only in whitespace normalize to the same string, so a line wrapped
// differently, in Chinese text too, still matches.
func Normalize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

const chunkSize = 1 << 20

// CheckExcerpts reads file once, a chunk at a time, keeping none of it: it
// returns the file's sha256 and, for each excerpt, whether the normalized
// excerpt appears in the normalized text. A file that cannot be read is
// source_missing, with no hash. A file of a non-text type, or one that is
// not valid UTF-8 or holds a NUL byte, is not_checkable for every excerpt,
// with its hash. The caller refuses excerpts that normalize to nothing.
// Only cancellation is an error; a cancelled check returns no result.
func CheckExcerpts(ctx context.Context, file string, excerpts []string) (string, []Verification, error) {
	results := make([]Verification, len(excerpts))
	missing := func() (string, []Verification, error) {
		for i := range results {
			results[i] = SourceMissing
		}
		return "", results, nil
	}
	if err := ctx.Err(); err != nil {
		return "", nil, interrupted(err)
	}
	// A FIFO or a device is not a source and must not block the check:
	// it is refused from its open descriptor (codex-review-020 B5).
	f, err := source.OpenRegular(file)
	if err != nil {
		return missing()
	}
	defer func() { _ = f.Close() }()

	text := !nonText[strings.ToLower(filepath.Ext(file))]
	matchers := make([]*matcher, len(excerpts))
	for i, e := range excerpts {
		matchers[i] = newMatcher(Normalize(e))
	}
	h := sha256.New()
	buf := make([]byte, chunkSize)
	var pending []byte // the start of a character split across chunks
	for {
		if err := ctx.Err(); err != nil {
			return "", nil, interrupted(err)
		}
		n, rerr := f.Read(buf)
		_, _ = h.Write(buf[:n])
		if text && n > 0 {
			pending = append(pending, buf[:n]...)
			i := 0
			for i < len(pending) && utf8.FullRune(pending[i:]) {
				r, size := utf8.DecodeRune(pending[i:])
				if (r == utf8.RuneError && size == 1) || r == 0 {
					text = false
					break
				}
				i += size
				if !unicode.IsSpace(r) {
					for _, m := range matchers {
						m.feed(r)
					}
				}
			}
			pending = append(pending[:0], pending[i:]...)
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return missing()
		}
	}
	if len(pending) > 0 {
		text = false // the file ends inside a character
	}
	for i, m := range matchers {
		switch {
		case !text:
			results[i] = NotCheckable
		case m.found:
			results[i] = Verified
		default:
			results[i] = NotFound
		}
	}
	return hex.EncodeToString(h.Sum(nil)), results, nil
}

func interrupted(err error) error {
	return apperr.Wrap(apperr.Environment, err, "checking the excerpts was interrupted; nothing was written")
}

// matcher finds one pattern in a stream of characters (Knuth–Morris–Pratt),
// so that a source of any size is searched without keeping it.
type matcher struct {
	pattern []rune
	fail    []int
	state   int
	found   bool
}

func newMatcher(pattern string) *matcher {
	p := []rune(pattern)
	fail := make([]int, len(p))
	for i, k := 1, 0; i < len(p); i++ {
		for k > 0 && p[i] != p[k] {
			k = fail[k-1]
		}
		if p[i] == p[k] {
			k++
		}
		fail[i] = k
	}
	return &matcher{pattern: p, fail: fail, found: len(p) == 0}
}

func (m *matcher) feed(r rune) {
	if m.found {
		return
	}
	for m.state > 0 && m.pattern[m.state] != r {
		m.state = m.fail[m.state-1]
	}
	if m.pattern[m.state] == r {
		m.state++
	}
	if m.state == len(m.pattern) {
		m.found = true
	}
}
