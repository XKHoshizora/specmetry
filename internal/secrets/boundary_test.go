package secrets_test

import "testing"

// codex-review-009: an environment reference exempts only its own match.
// A field that also carries a recognisable secret is rejected, whether the
// secret falls under another rule or under the same rule as the reference.
func TestEnvironmentReferenceDoesNotExemptTheWholeField(t *testing.T) {
	s := scanner(t)
	cases := map[string]struct {
		value string
		rule  string
	}{
		"other rule": {
			"API_KEY=$API_KEY ./deploy.sh --token " + "ghp_" + rep("A1b2", 9),
			"github_token",
		},
		"same rule, assignment": {
			"DB_PASSWORD=$DB_PASSWORD ADMIN_PASSWORD=" + "s3cr3t" + "Pass9",
			"generic_assignment",
		},
		"same rule, URL": {
			"psql postgres://app:${DB_PASSWORD}@db1/app && psql postgres://admin:" + "s3cr3tPass" + "@db2/app",
			"url_credentials",
		},
	}
	for name, c := range cases {
		if got, ok := s.Match(c.value); !ok || got != c.rule {
			t.Errorf("%s: Match = %q, %v; want %q", name, got, ok, c.rule)
		}
	}
}

// codex-review-009: with several candidate matches in one field, an
// exempt earlier match must not hide a later hit.
func TestLaterCandidateIsCheckedAfterAnExemptOne(t *testing.T) {
	s := scanner(t)
	cases := map[string]struct {
		value string
		rule  string
	}{
		"placeholder then value": {
			"password=<your-password>; password=" + "hunter2" + "Pass",
			"generic_assignment",
		},
		"short token then token": {
			"Bearer short Bearer " + rep("Z9y8", 6),
			"bearer_token",
		},
	}
	for name, c := range cases {
		if got, ok := s.Match(c.value); !ok || got != c.rule {
			t.Errorf("%s: Match = %q, %v; want %q", name, got, ok, c.rule)
		}
	}
}

// Only a value that is entirely an environment reference is exempt
// (architecture §10.8).
func TestEnvironmentReferenceForms(t *testing.T) {
	s := scanner(t)
	for _, v := range []string{
		"password=$DB_PASSWORD",
		"password=${DB_PASSWORD}",
		"password=%DB_PASSWORD%",
		"password=$env:DB_PASSWORD",
		"password=$(vault read -field=password secret/db)",
		"postgres://app:%DB_PASSWORD%@db/app",
	} {
		if got, ok := s.Match(v); ok {
			t.Errorf("%q matched %q, but the value is an environment reference", v, got)
		}
	}
	for _, v := range []string{
		"password=${DB_PASSWORD}" + "x9" + "Q",
		"password=prefix$DB_PASSWORD" + "9",
		"postgres://app:${DB_PASSWORD}" + "s3cr3t9" + "@db/app",
	} {
		if _, ok := s.Match(v); !ok {
			t.Errorf("%q: a value that only contains a reference must still be checked", v)
		}
	}
}
