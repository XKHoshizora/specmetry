// Package secrets rejects text that looks like a secret before Specmetry
// persists it (docs/architecture.md §10.8, TD-21).
//
// The rules lower the risk of storing a secret; they cannot detect every
// secret. Formats outside the rule set, secrets split or encoded across
// text, and short alphabetic passwords are known to pass.
package secrets

import "slices"

// ExtraRule is a rule added in config.toml under [secrets].
type ExtraRule struct {
	ID      string `toml:"id"`
	Pattern string `toml:"pattern"`
}

// Scanner matches text against the built-in rules and the extra rules.
type Scanner struct {
	rules []rule
}

// New returns a scanner with the built-in rules followed by extra, which
// is validated as by ValidateExtraRules.
func New(extra []ExtraRule) (*Scanner, error) {
	compiled, err := compileExtra(extra)
	if err != nil {
		return nil, err
	}
	return &Scanner{rules: append(slices.Clip(builtin), compiled...)}, nil
}

// Match reports the first rule, in table order, that has a hit in v.
func (s *Scanner) Match(v string) (ruleID string, ok bool) {
	for _, r := range s.rules {
		if r.hit(v) {
			return r.id, true
		}
	}
	return "", false
}

// BuiltinRules returns the built-in rule IDs in table order.
func BuiltinRules() []string {
	ids := make([]string, len(builtin))
	for i, r := range builtin {
		ids[i] = r.id
	}
	return ids
}
