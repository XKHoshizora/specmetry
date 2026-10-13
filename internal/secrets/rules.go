package secrets

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// rule is one detection rule. accept, when set, is the rule's condition
// (architecture §10.8): it receives the submatches of one match and decides
// whether that match is a hit.
type rule struct {
	id     string
	re     *regexp.Regexp
	accept func(groups []string) bool
}

// builtin is the rule table of architecture §10.8, in order. Changing a
// pattern or condition changes the approved TD-21 rule set.
var builtin = []rule{
	{id: "private_key", re: regexp.MustCompile(`-----BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----`)},
	{id: "aws_access_key_id", re: regexp.MustCompile(`\b(?:AKIA|ASIA|ABIA|ACCA)[A-Z0-9]{16}\b`)},
	{id: "github_token", re: regexp.MustCompile(`\b(?:(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,255}|github_pat_[A-Za-z0-9_]{22,255})\b`)},
	{id: "gitlab_token", re: regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`)},
	{id: "slack_token", re: regexp.MustCompile(`\bxox[abposr]-[0-9A-Za-z-]{10,}`)},
	{id: "stripe_secret_key", re: regexp.MustCompile(`\b(?:sk|rk)_(?:live|test)_[0-9A-Za-z]{16,}`)},
	{id: "google_api_key", re: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`)},
	{id: "anthropic_api_key", re: regexp.MustCompile(`\bsk-ant-(?:api|admin)[0-9]{2}-[A-Za-z0-9_-]{20,}`)},
	{id: "openai_api_key", re: regexp.MustCompile(`\bsk-(?:proj-|svcacct-|admin-)?[A-Za-z0-9_-]{20,}`)},
	{id: "npm_token", re: regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}\b`)},
	{id: "jwt", re: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
	{
		id:     "url_credentials",
		re:     regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9+.-]{1,20}://[^\s/:@]{1,256}:([^\s/@]{1,256})@[^\s/]+`),
		accept: func(g []string) bool { return !isEnvReference(g[1]) },
	},
	{
		id: "bearer_token",
		re: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/-]+=*)`),
		accept: func(g []string) bool {
			return utf8.RuneCountInString(g[1]) >= 20 && hasLetter(g[1]) && hasDigit(g[1])
		},
	},
	{
		id: "generic_assignment",
		re: regexp.MustCompile(`(?i)(password|passwd|pwd|secret|api[_-]?key|access[_-]?key|auth[_-]?token|access[_-]?token|private[_-]?key)["']?\s*[:=]\s*["']?([^\s"',;]+)`),
		accept: func(g []string) bool {
			v := g[2]
			if isEnvReference(v) || isPlaceholder(v) {
				return false
			}
			n := utf8.RuneCountInString(v)
			return n >= 8 && (hasLetter(v) && hasDigit(v) || n >= 20)
		},
	},
}

// envReference matches a value that is entirely one environment variable
// reference: $NAME, ${NAME}, %NAME% or $env:NAME. A command substitution
// such as $(...) is not a reference: it and any literal next to it are
// scanned like other text (codex-review-010 B1).
var envReference = regexp.MustCompile(`^(?:\$[A-Za-z_][A-Za-z0-9_]*|\$\{[A-Za-z_][A-Za-z0-9_]*\}|%[A-Za-z_][A-Za-z0-9_]*%|\$(?i:env):[A-Za-z_][A-Za-z0-9_]*)$`)

func isEnvReference(v string) bool { return envReference.MatchString(v) }

// isPlaceholder reports <...>, {{...}}, and values made only of *, x, X or '.'.
func isPlaceholder(v string) bool {
	if strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">") ||
		strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}") {
		return true
	}
	return v != "" && strings.Trim(v, "*xX.") == ""
}

func hasLetter(v string) bool { return strings.IndexFunc(v, unicode.IsLetter) >= 0 }
func hasDigit(v string) bool  { return strings.IndexFunc(v, unicode.IsDigit) >= 0 }

// hit reports whether r has a hit in v. A rule without a condition hits on
// any match. Otherwise the condition is applied to each match on its own,
// so an exempt match (an environment reference, a placeholder) never hides
// a later one in the same text.
func (r rule) hit(v string) bool {
	if r.accept == nil {
		return r.re.MatchString(v)
	}
	for _, g := range r.re.FindAllStringSubmatch(v, -1) {
		if r.accept(g) {
			return true
		}
	}
	return false
}
