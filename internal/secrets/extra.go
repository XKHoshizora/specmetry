package secrets

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"unicode/utf8"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

const (
	maxExtraRules  = 50
	maxPatternLen  = 500
	extraIDPattern = `^custom_[a-z0-9_]{1,40}$`
)

var extraID = regexp.MustCompile(extraIDPattern)

// ValidateExtraRules checks the [secrets] extra_rules of config.toml
// (architecture §10.8). An invalid rule is an environment error (exit code
// 6): Specmetry refuses to run rather than ignore it.
func ValidateExtraRules(rules []ExtraRule) error {
	_, err := compileExtra(rules)
	return err
}

// compileExtra validates and compiles extra rules. Errors name the rule by
// position and say what is wrong; they never contain a pattern or an
// invalid id, because doctor and the CLI print them.
func compileExtra(rules []ExtraRule) ([]rule, error) {
	out := make([]rule, 0, len(rules))
	seen := make(map[string]bool, len(rules))
	for i, r := range rules {
		if i == maxExtraRules {
			return nil, invalidExtra(i, fmt.Sprintf("at most %d rules are allowed", maxExtraRules))
		}
		if !extraID.MatchString(r.ID) {
			return nil, invalidExtra(i, "id must match "+extraIDPattern)
		}
		if seen[r.ID] {
			return nil, invalidExtra(i, "duplicate id "+r.ID)
		}
		seen[r.ID] = true
		if n := utf8.RuneCountInString(r.Pattern); n == 0 || n > maxPatternLen {
			return nil, invalidExtra(i, fmt.Sprintf("pattern must be 1 to %d characters long", maxPatternLen))
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, invalidExtra(i, "pattern is not a valid regular expression ("+syntaxErrorKind(err)+")")
		}
		if re.MatchString("") {
			return nil, invalidExtra(i, "pattern matches the empty string")
		}
		out = append(out, rule{id: r.ID, re: re})
	}
	return out, nil
}

// syntaxErrorKind returns the fixed description of a regexp parse error
// (e.g. "missing closing )") without the offending part of the pattern,
// which the error's text would include.
func syntaxErrorKind(err error) string {
	var se *syntax.Error
	if errors.As(err, &se) {
		return string(se.Code)
	}
	return "unknown error"
}

func invalidExtra(i int, problem string) error {
	return apperr.New(apperr.Environment, fmt.Sprintf("[secrets] extra_rules[%d]: %s", i, problem))
}
