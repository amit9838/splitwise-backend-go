package expense

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/split"
)

// ExpenseService is the business layer used by the handler.
type ExpenseService interface {
	Create(e Expense, splits []split.SplitInput) (Expense, error)
	GetById(id string) (Expense, error)
	ListByGroup(groupID string) ([]Expense, error)
	Update(id string, e Expense) (Expense, error)
	Delete(id string) (Expense, error)
}

// Handler
type Handler struct {
	service ExpenseService
}

// NewHandler create a new expense handler
func NewHandler(s ExpenseService) *Handler {
	return &Handler{service: s}
}

type splitInput struct {
	UserId     string   `json:"user_id"`
	Amount     *float64 `json:"amount"`
	Percentage *float64 `json:"percentage"`
	Shares     *int     `json:"shares"`
}

// Create handles POST /expenses
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	paidBy := auth.UserIDFrom(r.Context())
	if paidBy == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	var input struct {
		GroupId     string       `json:"group_id"`
		CategoryId  string       `json:"category_id"`
		Amount      float64      `json:"amount"`
		Description string       `json:"description"`
		Currency    string       `json:"currency"`
		SplitType   string       `json:"split_type"`
		ExpenseDate string       `json:"expense_date"`
		Splits      []splitInput `json:"splits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.GroupId == "" || input.CategoryId == "" || input.Amount <= 0 || input.Description == "" || input.ExpenseDate == "" {
		response.WriteError(w, http.StatusBadRequest, "please provide all the required fields [group_id, category_id, amount, description, expense_date]")
		return
	}

	expenseDate, err := time.Parse(time.RFC3339, input.ExpenseDate)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "expense_date must be a valid RFC3339 datetime")
		return
	}

	inputs := make([]split.SplitInput, 0, len(input.Splits))
	for _, s := range input.Splits {
		inputs = append(inputs, split.SplitInput{
			UserID:     s.UserId,
			Amount:     s.Amount,
			Percentage: s.Percentage,
			Shares:     s.Shares,
		})
	}

	created, err := h.service.Create(Expense{
		GroupId:     input.GroupId,
		CategoryId:  input.CategoryId,
		PaidBy:      paidBy,
		Amount:      input.Amount,
		Description: input.Description,
		Currency:    input.Currency,
		SplitType:   split.Type(input.SplitType),
		ExpenseDate: expenseDate,
	}, inputs)
	if err != nil {
		switch {
		case errors.Is(err, ErrGroupIdRequired),
			errors.Is(err, ErrCategoryIdRequired),
			errors.Is(err, ErrPaidByRequired),
			errors.Is(err, ErrAmountRequired),
			errors.Is(err, ErrExpenseDateRequired),
			errors.Is(err, ErrInvalidSplitType),
			errors.Is(err, ErrPayerNotMember),
			errors.Is(err, ErrSplitUserNotMember),
			errors.Is(err, split.ErrNoMembers),
			errors.Is(err, split.ErrSplitsRequired),
			errors.Is(err, split.ErrDuplicateSplitUsers),
			errors.Is(err, split.ErrExactSumMismatch),
			errors.Is(err, split.ErrPercentSum),
			errors.Is(err, split.ErrSharesInvalid),
			errors.Is(err, split.ErrInvalidType):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrGroupNotFound):
			response.WriteError(w, http.StatusNotFound, "group not found")
		case errors.Is(err, ErrCategoryNotFound):
			response.WriteError(w, http.StatusBadRequest, "Category not found")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to create expense")
		}
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// List handles GET /expenses/group/{group_id}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("group_id")

	expenses, err := h.service.ListByGroup(groupID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to list expenses")
		return
	}
	response.WriteJSON(w, http.StatusOK, expenses)
}

// Get handles GET /expenses/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	e, err := h.service.GetById(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "expense not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}
	response.WriteJSON(w, http.StatusOK, e)
}

// Update handles PUT /expenses/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var input struct {
		Amount      *float64 `json:"amount"`
		Description *string  `json:"description"`
		Currency    *string  `json:"currency"`
		SplitType   *string  `json:"split_type"`
		ExpenseDate *string  `json:"expense_date"`
		IsActive    *bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.service.GetById(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "expense not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch expense")
		return
	}

	if input.Amount != nil {
		existing.Amount = *input.Amount
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.Currency != nil {
		existing.Currency = *input.Currency
	}
	if input.SplitType != nil {
		existing.SplitType = split.Type(*input.SplitType)
	}
	if input.ExpenseDate != nil {
		parsed, err := time.Parse(time.RFC3339, *input.ExpenseDate)
		if err != nil {
			response.WriteError(w, http.StatusBadRequest, "expense_date must be a valid RFC3339 datetime")
			return
		}
		existing.ExpenseDate = parsed
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	updated, err := h.service.Update(id, existing)
	if err != nil {
		switch {
		case errors.Is(err, ErrAmountRequired),
			errors.Is(err, ErrExpenseDateRequired),
			errors.Is(err, ErrInvalidSplitType):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, response.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "expense not found")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to update expense")
		}
		return
	}
	response.WriteJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /expenses/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	_, err := h.service.Delete(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "expense not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to delete expense")
		return
	}
	response.WriteJSON(w, http.StatusOK, "Expense deleted successfully!")
}
