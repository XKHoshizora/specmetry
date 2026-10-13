package orders

import (
	"errors"
	"testing"
	"time"
)

// clock returns a time source that moves one minute per call.
func clock() func() time.Time {
	now := time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(time.Minute)
		return now
	}
}

var oneLine = []Line{{SKU: "A-1", Quantity: 2, UnitPrice: 1500}}

// R2: a valid order gets an ID and is pending; a non-positive quantity is
// refused.
func TestCreateOrder(t *testing.T) {
	s := NewStore(clock())
	o, err := s.Create("ana", "顾客甲", "000-0000-0001", oneLine)
	if err != nil || o.ID == "" || o.Status != Pending {
		t.Fatalf("Create = %+v, %v", o, err)
	}
	for _, q := range []int{0, -1} {
		lines := []Line{{SKU: "A-1", Quantity: q, UnitPrice: 1500}}
		if _, err := s.Create("ana", "顾客乙", "000-0000-0002", lines); !errors.Is(err, ErrInvalid) {
			t.Errorf("quantity %d: %v", q, err)
		}
	}
}

// R3: newest first; paging neither repeats nor skips an order.
func TestListPages(t *testing.T) {
	s := NewStore(clock())
	for i := 0; i < 5; i++ {
		if _, err := s.Create("ana", "顾客", "000-0000-0000", oneLine); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Create("bo", "顾客", "000-0000-0000", oneLine); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var last time.Time
	for page := 1; page <= 3; page++ {
		for _, o := range s.List("ana", page, 2) {
			if seen[o.ID] {
				t.Fatalf("%s listed twice", o.ID)
			}
			seen[o.ID] = true
			if !last.IsZero() && o.CreatedAt.After(last) {
				t.Fatalf("%s is out of order", o.ID)
			}
			last = o.CreatedAt
		}
	}
	if len(seen) != 5 {
		t.Fatalf("listed %d of 5 orders", len(seen))
	}
}

// R4: another clerk's order looks exactly like a missing one.
func TestOrdersAreIsolated(t *testing.T) {
	s := NewStore(clock())
	o, err := s.Create("bo", "顾客", "000-0000-0000", oneLine)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("ana", o.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other clerk's order: %v", err)
	}
	if _, err := s.Get("ana", "ORD-99999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing order: %v", err)
	}
	if got, err := s.Get("bo", o.ID); err != nil || got != o {
		t.Errorf("own order: %v, %v", got, err)
	}
}
