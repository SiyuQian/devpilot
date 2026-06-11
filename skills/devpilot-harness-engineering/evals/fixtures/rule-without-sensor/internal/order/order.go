package order

import "time"

// VIOLATION: reads the wall clock directly inside business logic.
// CLAUDE.md bans this, but nothing mechanical catches it.
func (o *Order) MarkPaid() {
	o.PaidAt = time.Now() // should come from an injected Clock
	o.Status = "paid"
}

type Order struct {
	ID     string
	Status string
	PaidAt time.Time
}
