package settlement

import "time"

type Settlement struct {
	ID            string    `json:"id"`
	GroupId       string    `json:"group_id"`
	PaidBy        string    `json:"paid_by"`
	PaidTo        string    `json:"paid_to"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"` // cash, bank, upi
	Note          string    `json:"note"`
	SettledAt     time.Time `json:"settled_at"`
	CreatedAt     time.Time `json:"created_at"`
}
