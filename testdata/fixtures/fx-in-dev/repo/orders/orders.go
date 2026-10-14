package orders

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Status is an order's state.
type Status string

// Order states.
const (
	Pending   Status = "pending"
	Shipped   Status = "shipped"
	Cancelled Status = "cancelled"
)

// Line is one item of an order.
type Line struct {
	SKU       string
	Quantity  int
	UnitPrice int // in cents
}

// Order is a customer's order, created by a clerk (its owner).
type Order struct {
	ID           string
	Owner        string
	Customer     string
	Phone        string
	Lines        []Line
	Status       Status
	CreatedAt    time.Time
	CancelReason string
}

// ErrInvalid is returned for an order that cannot be created.
var ErrInvalid = errors.New("invalid order")

// ErrNotFound is returned for an order that does not exist or belongs to
// another clerk, so the two cannot be told apart (R4).
var ErrNotFound = errors.New("order not found")

// Store keeps orders in memory.
type Store struct {
	now    func() time.Time
	next   int
	orders map[string]*Order
}

// NewStore returns an empty store that reads the time from now.
func NewStore(now func() time.Time) *Store {
	return &Store{now: now, orders: map[string]*Order{}}
}

// Create records a pending order for owner and gives it a unique ID (R2).
func (s *Store) Create(owner, customer, phone string, lines []Line) (*Order, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("%w: no lines", ErrInvalid)
	}
	for _, l := range lines {
		if l.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity of %s must be positive", ErrInvalid, l.SKU)
		}
	}
	s.next++
	o := &Order{
		ID:        fmt.Sprintf("ORD-%05d", s.next),
		Owner:     owner,
		Customer:  customer,
		Phone:     phone,
		Lines:     lines,
		Status:    Pending,
		CreatedAt: s.now(),
	}
	s.orders[o.ID] = o
	return o, nil
}

// Get returns one of owner's orders (R4).
func (s *Store) Get(owner, id string) (*Order, error) {
	o, ok := s.orders[id]
	if !ok || o.Owner != owner {
		return nil, ErrNotFound
	}
	return o, nil
}

// List returns page (from 1) of owner's orders, newest first, size orders
// per page (R3).
func (s *Store) List(owner string, page, size int) []*Order {
	var mine []*Order
	for _, o := range s.orders {
		if o.Owner == owner {
			mine = append(mine, o)
		}
	}
	sort.Slice(mine, func(i, j int) bool {
		if !mine[i].CreatedAt.Equal(mine[j].CreatedAt) {
			return mine[i].CreatedAt.After(mine[j].CreatedAt)
		}
		return mine[i].ID > mine[j].ID
	})
	start := (page - 1) * size
	if page < 1 || size < 1 || start >= len(mine) {
		return nil
	}
	return mine[start:min(start+size, len(mine))]
}
