package expense

import (
	"errors"

	"github.com/amit9838/splitwise-backend-go/internal/group"
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
	ErrGroupNotFound       = errors.New("group not found")
	ErrPayerNotMember      = errors.New("payer is not an active member of this group")
	ErrSplitUserNotMember  = errors.New("split user is not an active member of this group")
)

// ExpenseStore is the persistence layer required by Service.
type ExpenseStore interface {
	CreateWithSplits(e Expense, splits []Split) (Expense, []Split, error)
	GetById(id string) (Expense, error)
	ListByGroup(groupID string) ([]Expense, error)
	ListSplitsByExpenseID(expenseID string) ([]Split, error)
	ListSplitsByGroup(groupID string) ([]Split, error)
	Update(id string, e Expense) (Expense, error)
	Delete(id string) (Expense, error)
}

// GroupLookup provides the group info required by Service.
type GroupLookup interface {
	GetById(id string) (group.Group, error)
}

// MemberLookup provides the membership info required by Service.
type MemberLookup interface {
	ListActiveByGroup(groupID string) ([]group.Member, error)
}

// Service holds the expense business rules and delegates persistence to
// an ExpenseStore, with group and membership lookups for validation.
type Service struct {
	store   ExpenseStore
	groups  GroupLookup
	members MemberLookup
}

func NewService(store ExpenseStore, groups GroupLookup, members MemberLookup) *Service {
	return &Service{store: store, groups: groups, members: members}
}

// validateMutable checks the fields that may be set on create and update.
func validateMutable(e Expense) error {
	if e.Amount <= 0 {
		return ErrAmountRequired
	}
	if e.ExpenseDate.IsZero() {
		return ErrExpenseDateRequired
	}
	if e.SplitType != "" && !e.SplitType.Valid() {
		return ErrInvalidSplitType
	}
	return nil
}

// attachSplits loads an expense's splits into the model.
func (s *Service) attachSplits(e *Expense) error {
	splits, err := s.store.ListSplitsByExpenseID(e.ID)
	if err != nil {
		return err
	}
	if splits == nil {
		splits = make([]Split, 0)
	}
	e.Splits = splits
	return nil
}

// Create validates an expense, computes its splits from the split type
// and stores everything in one transaction.
func (s *Service) Create(e Expense, inputs []split.SplitInput) (Expense, error) {
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
	if e.Currency == "" {
		e.Currency = "INR"
	}
	if err := validateMutable(e); err != nil {
		return Expense{}, err
	}

	if _, err := s.groups.GetById(e.GroupId); err != nil {
		return Expense{}, ErrGroupNotFound
	}

	members, err := s.members.ListActiveByGroup(e.GroupId)
	if err != nil {
		return Expense{}, err
	}
	memberIDs := make([]string, 0, len(members))
	memberSet := make(map[string]bool, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.UserId)
		memberSet[m.UserId] = true
	}
	if len(memberSet) == 0 {
		return Expense{}, split.ErrNoMembers
	}
	if !memberSet[e.PaidBy] {
		return Expense{}, ErrPayerNotMember
	}
	for _, in := range inputs {
		if !memberSet[in.UserID] {
			return Expense{}, ErrSplitUserNotMember
		}
	}

	computed, err := split.Compute(split.Request{
		Type:      e.SplitType,
		Total:     e.Amount,
		MemberIDs: memberIDs,
		Splits:    inputs,
	})
	if err != nil {
		return Expense{}, err
	}

	splits := make([]Split, 0, len(computed))
	for _, c := range computed {
		splits = append(splits, Split{
			UserId:     c.UserID,
			Amount:     c.Amount,
			Percentage: c.Percentage,
			Shares:     c.Shares,
		})
	}

	created, _, err := s.store.CreateWithSplits(e, splits)
	if err != nil {
		return Expense{}, err
	}
	return created, nil
}

// GetById returns an expense with its splits.
func (s *Service) GetById(id string) (Expense, error) {
	e, err := s.store.GetById(id)
	if err != nil {
		return Expense{}, err
	}
	if err := s.attachSplits(&e); err != nil {
		return Expense{}, err
	}
	return e, nil
}

// ListByGroup returns a group's active expenses with their splits.
func (s *Service) ListByGroup(groupID string) ([]Expense, error) {
	expenses, err := s.store.ListByGroup(groupID)
	if err != nil {
		return nil, err
	}
	for i := range expenses {
		if err := s.attachSplits(&expenses[i]); err != nil {
			return nil, err
		}
	}
	return expenses, nil
}

// Update modifies the mutable fields (amount, description, currency,
// split_type, expense_date, is_active) of an existing expense.
// group_id, category_id and paid_by are immutable; splits are not recomputed.
func (s *Service) Update(id string, e Expense) (Expense, error) {
	if e.Currency == "" {
		e.Currency = "INR"
	}
	if err := validateMutable(e); err != nil {
		return Expense{}, err
	}
	updated, err := s.store.Update(id, e)
	if err != nil {
		return Expense{}, err
	}
	if err := s.attachSplits(&updated); err != nil {
		return Expense{}, err
	}
	return updated, nil
}

// Delete removes an expense and its splits.
func (s *Service) Delete(id string) (Expense, error) {
	return s.store.Delete(id)
}
