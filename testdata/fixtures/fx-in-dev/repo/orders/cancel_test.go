package orders

import (
	"errors"
	"testing"
)

// R5: a pending order is cancelled with its reason; a shipped one is not.
func TestCancel(t *testing.T) {
	s := NewStore(clock())
	pending, _ := s.Create("ana", "顾客", "000-0000-0000", oneLine)
	shipped, _ := s.Create("ana", "顾客", "000-0000-0000", oneLine)
	if err := s.Ship("ana", shipped.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel("ana", pending.ID, ""); !errors.Is(err, ErrReasonRequired) {
		t.Errorf("no reason: %v", err)
	}
	if err := s.Cancel("ana", pending.ID, "顾客改变主意"); err != nil {
		t.Fatal(err)
	}
	if pending.Status != Cancelled || pending.CancelReason != "顾客改变主意" {
		t.Errorf("cancelled order = %+v", pending)
	}
	if err := s.Cancel("ana", shipped.ID, "太晚了"); !errors.Is(err, ErrShipped) {
		t.Errorf("shipped order: %v", err)
	}
}
