package checkrun_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/contract"
)

var validator = sync.OnceValues(contract.New)

// envelope is err as error.v1 (D10: the existing fields only), checked
// against the schema, as JSON text.
func envelope(t *testing.T, err error) string {
	t.Helper()
	val, verr := validator()
	if verr != nil {
		t.Fatal(verr)
	}
	b, merr := json.Marshal(apperr.ToEnvelope(err))
	if merr != nil {
		t.Fatal(merr)
	}
	if verr := val.ValidateJSON("error.v1", b); verr != nil {
		t.Fatalf("not an error.v1 document: %v\n%s", verr, b)
	}
	return string(b)
}
