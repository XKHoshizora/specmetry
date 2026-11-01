package checkrun_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
	"github.com/XKHoshizora/specmetry/internal/contract"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
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

// closeBatch closes b and fails the test when a close did not confirm its
// release (Review 123 B-123-2: a test's own cleanup checks it too).
func closeBatch(t *testing.T, b *checkrun.Batch) {
	t.Helper()
	if err := b.Close(); err != nil {
		t.Errorf("Batch.Close: %v", err)
	}
}

// closeDB closes a test's own database handle and checks the result.
func closeDB(t *testing.T, db *sqlite.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Errorf("closing the test's database: %v", err)
	}
}
