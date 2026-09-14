package settlement

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

// SettlementService is the business layer used by the handler.
type SettlementService interface {
	Create(st Settlement, payerID string) (Settlement, error)
	GetById(id string) (Settlement, error)
	ListByGroup(groupID string) ([]Settlement, error)
	Delete(id string, requesterID string) (Settlement, error)
}

// Handler
type Handler struct {
	service SettlementService
}

// NewHandler creates a new settlement Handler.
func NewHandler(s SettlementService) *Handler {
	return &Handler{service: s}
}

// Create handles POST /settlements
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	payerID := auth.UserIDFrom(r.Context())
	if payerID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	var input struct {
		GroupId       string  `json:"group_id"`
		PaidTo        string  `json:"paid_to"`
		Amount        float64 `json:"amount"`
		PaymentMethod string  `json:"payment_method"`
		Note          string  `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.Create(Settlement{
		GroupId:       input.GroupId,
		PaidTo:        input.PaidTo,
		Amount:        input.Amount,
		PaymentMethod: input.PaymentMethod,
		Note:          input.Note,
	}, payerID)
	if err != nil {
		switch {
		case errors.Is(err, ErrGroupIdRequired),
			errors.Is(err, ErrPaidToRequired),
			errors.Is(err, ErrAmountRequired),
			errors.Is(err, ErrCannotSettleSelf),
			errors.Is(err, ErrRecipientNotMember),
			errors.Is(err, ErrPayerNotMember),
			errors.Is(err, ErrInvalidMethod):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrGroupNotFound):
			response.WriteError(w, http.StatusNotFound, "group not found")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to create settlement")
		}
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// List handles GET /settlements/group/{group_id}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("group_id")

	settlements, err := h.service.ListByGroup(groupID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to list settlements")
		return
	}
	response.WriteJSON(w, http.StatusOK, settlements)
}

// Delete handles DELETE /settlements/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	requesterID := auth.UserIDFrom(r.Context())
	if requesterID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	_, err := h.service.Delete(id, requesterID)
	if err != nil {
		switch {
		case errors.Is(err, response.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "settlement not found")
		case errors.Is(err, ErrNotPayer):
			response.WriteError(w, http.StatusForbidden, "Only the payer can delete a settlement")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to delete settlement")
		}
		return
	}
	response.WriteJSON(w, http.StatusOK, "Settlement deleted successfully!")
}
