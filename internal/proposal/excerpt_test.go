package proposal_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/XKHoshizora/specmetry/internal/proposal"
)

var ctx = context.Background()

func writeSource(t *testing.T, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// MVP-BL-04: differences in whitespace or line breaks only (CRLF too) still
// verify; one changed character does not; a PDF cannot be checked; a file
// that is gone is missing. The expected results are written out.
func TestCheckExcerpts(t *testing.T) {
	brief := "## R1 账号登录\r\n\r\n店员使用公司分配的账号和密码登录系统。连续 5 次输错密码后，\r\n账号锁定 15 分钟。\r\n" +
		"Orders are listed\tnewest   first,\nten per page.\n"
	file := writeSource(t, "brief.md", []byte(brief))
	cases := []struct {
		excerpt string
		want    proposal.Verification
	}{
		{"连续 5 次输错密码后，账号锁定 15 分钟。", proposal.Verified},                       // a line break (CRLF) in the source
		{"连续5次输错密码后，账号锁定15分钟。", proposal.Verified},                           // spaces left out
		{"连续 5 次输错密码后，\n账号锁定 15 分钟。", proposal.Verified},                     // LF where the source has CRLF
		{"连续　5　次输错密码后，账号锁定 15 分钟。", proposal.Verified},                       // full-width spaces
		{"Orders are listed newest first, ten per page.", proposal.Verified}, // tab, runs of spaces, line break
		{"连续 6 次输错密码后，账号锁定 15 分钟。", proposal.NotFound},                       // one character changed
		{"Orders are listed newest first, ten per pages.", proposal.NotFound},
		{"orders are listed newest first", proposal.NotFound}, // letter case is not whitespace
		{"店员使用公司分配的账号和密码登录系统。", proposal.Verified},
	}
	excerpts := make([]string, len(cases))
	for i, c := range cases {
		excerpts[i] = c.excerpt
	}
	gotSHA, got, err := proposal.CheckExcerpts(ctx, file, excerpts)
	if err != nil {
		t.Fatal(err)
	}
	if gotSHA != sha([]byte(brief)) {
		t.Errorf("source sha256 = %s, want the file's", gotSHA)
	}
	for i, c := range cases {
		if got[i] != c.want {
			t.Errorf("%q: %s, want %s", c.excerpt, got[i], c.want)
		}
	}

	pdf := writeSource(t, "brief.PDF", []byte("%PDF-1.7\n账号锁定 15 分钟。\n"))
	for _, c := range []struct {
		name, file string
		want       proposal.Verification
		wantSHA    bool
	}{
		{"a PDF (any letter case)", pdf, proposal.NotCheckable, true},
		{"a .docx", writeSource(t, "brief.docx", []byte("账号锁定 15 分钟。")), proposal.NotCheckable, true},
		{"not UTF-8", writeSource(t, "gbk.txt", []byte{0xd5, 0xcb, 0xba, 0xc5, '\n'}), proposal.NotCheckable, true},
		{"a NUL byte", writeSource(t, "nul.txt", []byte("账号锁定 15 分钟。\x00")), proposal.NotCheckable, true},
		{"a file that is gone", filepath.Join(t.TempDir(), "gone.md"), proposal.SourceMissing, false},
		{"a directory", t.TempDir(), proposal.SourceMissing, false},
	} {
		gotSHA, got, err := proposal.CheckExcerpts(ctx, c.file, []string{"账号锁定 15 分钟。"})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got[0] != c.want || (gotSHA != "") != c.wantSHA {
			t.Errorf("%s: %s with sha256 %q, want %s (hash: %v)", c.name, got[0], gotSHA, c.want, c.wantSHA)
		}
	}
}

// An excerpt that is nothing but whitespace would be found anywhere; the
// caller refuses it, and Normalize shows why.
func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"a b\tc\r\nd　e f": "abcdef",
		" \r\n\t　":        "",
		"账号 锁定":           "账号锁定",
	} {
		if got := proposal.Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// oracle is an independent check: the whole file in memory, every
// whitespace rune dropped, and a plain substring search.
func oracle(content []byte, excerpt string) proposal.Verification {
	drop := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}
	if strings.Contains(drop(string(content)), drop(excerpt)) {
		return proposal.Verified
	}
	return proposal.NotFound
}

// The streaming check agrees with the oracle on random text in files large
// enough to be read in several chunks, with multi-byte characters and line
// breaks falling across chunk boundaries, for excerpts taken from the text
// (with their whitespace changed) and excerpts changed in one character.
func TestCheckExcerptsAgreesWithTheOracle(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	alphabet := []rune("ab订单账号锁定，。 \t\r\n　xyz数量")
	for round := range 30 {
		n := 1 + r.IntN(300_000)
		text := make([]rune, n)
		for i := range text {
			text[i] = alphabet[r.IntN(len(alphabet))]
		}
		content := []byte(string(text))
		file := writeSource(t, "doc.md", content)
		var excerpts []string
		for range 20 {
			start := r.IntN(len(text))
			end := min(len(text), start+1+r.IntN(200))
			piece := []rune(string(text[start:end]))
			// Change whitespace only, or one character.
			switch r.IntN(3) {
			case 0:
				piece = []rune(strings.NewReplacer("\r\n", "\n", "\t", " ", "　", "").Replace(string(piece)))
			case 1:
				piece[r.IntN(len(piece))] = 'Q'
			}
			excerpts = append(excerpts, string(piece))
		}
		_, got, err := proposal.CheckExcerpts(ctx, file, excerpts)
		if err != nil {
			t.Fatal(err)
		}
		for i, e := range excerpts {
			if proposal.Normalize(e) == "" {
				continue
			}
			if want := oracle(content, e); got[i] != want {
				t.Fatalf("round %d: excerpt %q: %s, oracle %s", round, e, got[i], want)
			}
		}
	}
}

// Reading stops when the context is cancelled; no partial result.
func TestCheckExcerptsStopsWhenCanceled(t *testing.T) {
	file := writeSource(t, "doc.md", []byte(strings.Repeat("订单 ", 1<<20)))
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := proposal.CheckExcerpts(canceled, file, []string{"订单"}); err == nil {
		t.Fatal("a canceled check returned a result")
	}
}
