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
	Amount      float32    `json:"amount"`
	Description string     `json:"description"`
	SplitType   split.Type `json:"split_type"` // EQUAL, EXACT, PERCENTAGE, SHARES
	ExpenseDate time.Time  `json:"expense_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
