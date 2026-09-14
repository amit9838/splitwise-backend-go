package expense

import (
	"errors"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/split"
)

var (
	ErrGroupIdRequired     = errors.New("group_id is required")
	ErrCategoryIdRequired  = errors.New("category_id is required")
	ErrPaidByRequired      = errors.New("paid_by is required")
	ErrAmountRequired      = errors.New("amount must be greater than zero")
	ErrExpenseDateRequired = errors.New("expense_date is required")
	ErrInvalidSplitType    = errors.New("split_type must be one of EQUAL, EXACT, PERCENTAGE, SHARES")
	ErrCategoryNotFound    = errors.New("category not found")
)

type ExpenseStore interface {
	Create(e Expense) (Expense, error)
	GetById(id string) (Expense, error)
	List() ([]Expense, error)
	ListByGroup(groupID string) ([]Expense, error)
	Update(id string, e Expense) (Expense, error)
	Delete(id string) (Expense, error)
}

type Service struct {
	store ExpenseStore
}

func NewService(store ExpenseStore) *Service {
	return &Service{store: store}
}

// validateMutable checks the fields that may be set on create and update.
func validateMutable(e Expense) error {
	if e.Amount <= 0 {
		return ErrAmountRequired
	}
	if e.ExpenseDate.IsZero() {
		return ErrExpenseDateRequired
	}
	if !e.SplitType.Valid() {
		return ErrInvalidSplitType
	}
	return nil
}

// Create Expense
func (s *Service) Create(e Expense) (Expense, error) {
	if e.GroupId == "" {
		return Expense{}, ErrGroupIdRequired
	}
	if e.CategoryId == "" {
		return Expense{}, ErrCategoryIdRequired
	}
	if e.PaidBy == "" {
		return Expense{}, ErrPaidByRequired
	}
	if e.SplitType == "" {
		e.SplitType = split.Equal
	}
	if err := validateMutable(e); err != nil {
		return Expense{}, err
	}
	return s.store.Create(e)
}

// GetById returns an expense by id.
func (s *Service) GetById(id string) (Expense, error) {
	return s.store.GetById(id)
}

// List returns all expenses.
func (s *Service) List() ([]Expense, error) {
	return s.store.List()
}

// ListByGroup returns all expenses for a group.
func (s *Service) ListByGroup(groupID string) ([]Expense, error) {
	return s.store.ListByGroup(groupID)
}

// Update modifies the mutable fields (amount, description, split_type,
// expense_date) of an existing expense. group_id, category_id and paid_by
// are immutable.
func (s *Service) Update(id string, e Expense) (Expense, error) {
	if err := validateMutable(e); err != nil {
		return Expense{}, err
	}
	return s.store.Update(id, e)
}

// Delete removes an expense.
func (s *Service) Delete(id string) (Expense, error) {
	return s.store.Delete(id)
}
