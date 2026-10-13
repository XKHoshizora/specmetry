package orders

import (
	"errors"
	"testing"
	"time"
)

// R1: after five wrong passwords the sixth login is refused as locked; 15
// minutes later the clerk can log in again.
func TestLockoutAfterFiveFailures(t *testing.T) {
	now := time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC)
	a := NewAccounts(func() time.Time { return now })
	a.Add("ana", "correct horse")
	for i := 1; i <= 5; i++ {
		if err := a.Login("ana", "wrong"); !errors.Is(err, ErrBadCredentials) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if err := a.Login("ana", "correct horse"); !errors.Is(err, ErrLocked) {
		t.Fatalf("sixth attempt: %v", err)
	}
	now = now.Add(15 * time.Minute)
	if err := a.Login("ana", "correct horse"); err != nil {
		t.Fatalf("after the lockout: %v", err)
	}
}

func TestUnknownClerk(t *testing.T) {
	a := NewAccounts(time.Now)
	if err := a.Login("nobody", "x"); !errors.Is(err, ErrBadCredentials) {
		t.Fatal(err)
	}
}
