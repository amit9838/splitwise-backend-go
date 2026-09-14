package balance

import (
	"errors"
	"net/http"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

// BalanceService is the business layer used by the handler.
type BalanceService interface {
	GroupBalances(groupID string) (*GroupBalances, error)
	MyBalances(userID string) (*MyBalances, error)
}

// Handler
type Handler struct {
	service BalanceService
}

// NewHandler creates a new balance Handler.
func NewHandler(s BalanceService) *Handler {
	return &Handler{service: s}
}

// Group handles GET /balances/group/{group_id}
func (h *Handler) Group(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("group_id")

	balances, err := h.service.GroupBalances(groupID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			response.WriteError(w, http.StatusNotFound, "group not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to compute balances")
		return
	}
	response.WriteJSON(w, http.StatusOK, balances)
}

// Me handles GET /balances/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	balances, err := h.service.MyBalances(userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to compute balances")
		return
	}
	response.WriteJSON(w, http.StatusOK, balances)
}
