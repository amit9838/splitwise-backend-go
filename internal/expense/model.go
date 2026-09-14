package expense

import (
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/split"
)

type Expense struct {
	ID          string     `json:"id"`
	GroupId     string     `json:"group_id"`
	CategoryId  string     `json:"category_id"`
	PaidBy      string     `json:"paid_by"`
	Amount      float64    `json:"amount"`
	Description string     `json:"description"`
	Currency    string     `json:"currency"`
	SplitType   split.Type `json:"split_type"` // EQUAL, EXACT, PERCENTAGE, SHARES
	ExpenseDate time.Time  `json:"expense_date"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Splits      []Split    `json:"splits"`
}

// Split is one user's share of an expense.
type Split struct {
	ID         string   `json:"id"`
	ExpenseId  string   `json:"-"`
	UserId     string   `json:"user_id"`
	Amount     float64  `json:"amount"`
	Percentage *float64 `json:"percentage"`
	Shares     *int     `json:"shares"`
}
