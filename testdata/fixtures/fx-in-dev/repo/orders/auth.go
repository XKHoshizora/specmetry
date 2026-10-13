// Package orders is the order management service of the fx-in-dev fixture.
package orders

import (
	"errors"
	"time"
)

// ErrBadCredentials is returned for an unknown clerk or a wrong password.
var ErrBadCredentials = errors.New("bad credentials")

// ErrLocked is returned while an account is locked after failed logins.
var ErrLocked = errors.New("account locked")

const (
	maxFailures = 5
	lockout     = 15 * time.Minute
)

type account struct {
	secret      string
	failures    int
	lockedUntil time.Time
}

// Accounts checks clerks' logins and locks an account for 15 minutes after
// five wrong passwords in a row (R1).
type Accounts struct {
	now      func() time.Time
	accounts map[string]*account
}

// NewAccounts returns an empty set of accounts that reads the time from now.
func NewAccounts(now func() time.Time) *Accounts {
	return &Accounts{now: now, accounts: map[string]*account{}}
}

// Add registers a clerk.
func (a *Accounts) Add(clerk, secret string) {
	a.accounts[clerk] = &account{secret: secret}
}

// Login checks a clerk's password.
func (a *Accounts) Login(clerk, secret string) error {
	acc, ok := a.accounts[clerk]
	if !ok {
		return ErrBadCredentials
	}
	if a.now().Before(acc.lockedUntil) {
		return ErrLocked
	}
	if acc.secret != secret {
		acc.failures++
		if acc.failures >= maxFailures {
			acc.failures = 0
			acc.lockedUntil = a.now().Add(lockout)
		}
		return ErrBadCredentials
	}
	acc.failures = 0
	return nil
}
