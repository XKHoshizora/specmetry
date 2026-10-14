package orders

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"time"
)

// ExportCSV writes the orders created in [from, to) as CSV with the order
// ID, creation time, customer and amount (R6).
func (s *Store) ExportCSV(w io.Writer, from, to time.Time) error {
	var rows []*Order
	for _, o := range s.orders {
		if !o.CreatedAt.Before(from) && o.CreatedAt.Before(to) {
			rows = append(rows, o)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"order_id", "created_at", "customer", "amount"}); err != nil {
		return err
	}
	for _, o := range rows {
		amount := 0
		for _, l := range o.Lines {
			amount += l.Quantity * l.UnitPrice
		}
		record := []string{o.ID, o.CreatedAt.UTC().Format(time.RFC3339), o.Customer, strconv.Itoa(amount)}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
