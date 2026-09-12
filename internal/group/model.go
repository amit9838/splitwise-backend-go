package group

import "time"

type Group struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	CreatedBy     string    `json:"created_by"`
	SimplifyDebts bool      `json:"simplify_debts"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
