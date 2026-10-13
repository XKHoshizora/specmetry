package orders

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

// R6: the export has four columns and only the orders of the date range.
func TestExportCSV(t *testing.T) {
	s := NewStore(clock())
	var created []*Order
	for i := 0; i < 3; i++ {
		o, err := s.Create("ana", "顾客", "000-0000-0000", oneLine)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, o)
	}
	var out strings.Builder
	from, to := created[1].CreatedAt, created[2].CreatedAt.Add(time.Second)
	if err := s.ExportCSV(&out, from, to); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || len(rows[0]) != 4 || rows[1][0] != created[1].ID || rows[2][3] != "3000" {
		t.Fatalf("rows = %v", rows)
	}
}
