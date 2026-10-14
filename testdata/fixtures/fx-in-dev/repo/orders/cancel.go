package orders

import "errors"

// ErrReasonRequired is returned when a cancellation gives no reason (R5).
var ErrReasonRequired = errors.New("a reason is required")

// ErrShipped is returned when cancelling an order that has shipped (R5).
var ErrShipped = errors.New("order has shipped")

// Ship marks one of owner's orders as shipped.
func (s *Store) Ship(owner, id string) error {
	o, err := s.Get(owner, id)
	if err != nil {
		return err
	}
	o.Status = Shipped
	return nil
}

// Cancel cancels one of owner's orders that has not shipped, recording why
// (R5).
func (s *Store) Cancel(owner, id, reason string) error {
	if reason == "" {
		return ErrReasonRequired
	}
	o, err := s.Get(owner, id)
	if err != nil {
		return err
	}
	if o.Status == Shipped {
		return ErrShipped
	}
	o.Status = Cancelled
	o.CancelReason = reason
	return nil
}
