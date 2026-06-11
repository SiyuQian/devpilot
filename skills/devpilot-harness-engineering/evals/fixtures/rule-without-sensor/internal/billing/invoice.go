package billing

import "time"

// VIOLATION: another direct time.Now() in domain code.
func NewInvoice(id string) *Invoice {
	return &Invoice{ID: id, IssuedAt: time.Now()}
}

type Invoice struct {
	ID       string
	IssuedAt time.Time
}
