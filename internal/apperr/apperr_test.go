package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, ExitOK},
		{New(InvalidInput, "x"), ExitInvalidInput},
		{New(NotFound, "x"), ExitInvalidInput},
		{New(HumanRequired, "x"), ExitHumanRequired},
		{New(NotInitialized, "x"), ExitNotInitialized},
		{New(StalePrecondition, "x"), ExitStalePrecondition},
		{New(Environment, "x"), ExitEnvironment},
		{New(Internal, "x"), ExitInternal},
		{errors.New("plain"), ExitInternal},
		{fmt.Errorf("wrapped: %w", New(Environment, "x")), ExitEnvironment},
	}
	for _, c := range cases {
		if got := ExitCode(c.err); got != c.want {
			t.Errorf("ExitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestHTTPStatus(t *testing.T) {
	want := map[Code]int{
		InvalidInput:      http.StatusBadRequest,
		Unauthorized:      http.StatusUnauthorized,
		Forbidden:         http.StatusForbidden,
		HumanRequired:     http.StatusForbidden,
		NotFound:          http.StatusNotFound,
		NotInitialized:    http.StatusNotFound,
		StalePrecondition: http.StatusConflict,
		Environment:       http.StatusServiceUnavailable,
		Internal:          http.StatusInternalServerError,
	}
	for code, status := range want {
		if got := HTTPStatus(code); got != status {
			t.Errorf("HTTPStatus(%s) = %d, want %d", code, got, status)
		}
	}
}

func TestFromWrapsPlainErrorsAsInternal(t *testing.T) {
	cause := errors.New("disk on fire")
	e := From(cause)
	if e.Code != Internal || !errors.Is(e, cause) {
		t.Fatalf("From(plain) = %+v", e)
	}
	if From(nil) != nil {
		t.Fatal("From(nil) must be nil")
	}
}

func TestEnvelopeTruncatesMessage(t *testing.T) {
	env := ToEnvelope(New(InvalidInput, strings.Repeat("界", 3000)))
	if n := len([]rune(env.Error.Message)); n != maxMessageLen {
		t.Fatalf("message has %d runes, want %d", n, maxMessageLen)
	}
	if env.Schema != SchemaID {
		t.Fatalf("schema = %q", env.Schema)
	}
}
