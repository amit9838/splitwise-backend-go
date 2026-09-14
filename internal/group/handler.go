package group

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

// GroupService is the business layer used by the handler.
type GroupService interface {
	Create(g Group, creatorID string, memberIDs []string) (Group, error)
	GetById(id string) (Group, error)
	List(userID string) ([]Group, error)
	Update(id string, g Group) (Group, error)
	Delete(id string, requesterID string) (Group, error)
	AddMember(groupID, userID string) (Group, error)
	RemoveMember(groupID, requesterID, userID string) error
}

// Handler
type Handler struct {
	service GroupService
}

// NewHandler creates a new group Handler.
func NewHandler(s GroupService) *Handler {
	return &Handler{service: s}
}

// Create handles POST /groups
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	creatorID := auth.UserIDFrom(r.Context())
	if creatorID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	var input struct {
		Name          string   `json:"name"`
		MemberIDs     []string `json:"member_ids"`
		SimplifyDebts *bool    `json:"simplify_debts"`
		Currency      string   `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		response.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	simplifyDebts := true
	if input.SimplifyDebts != nil {
		simplifyDebts = *input.SimplifyDebts
	}

	created, err := h.service.Create(Group{
		Name:          input.Name,
		SimplifyDebts: simplifyDebts,
		Currency:      input.Currency,
	}, creatorID, input.MemberIDs)
	if err != nil {
		switch {
		case errors.Is(err, ErrNameRequired),
			errors.Is(err, ErrCreatorRequired),
			errors.Is(err, ErrCreatorInMembers),
			errors.Is(err, ErrDuplicateMembers):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrMembersNotFound):
			response.WriteError(w, http.StatusNotFound, err.Error())
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to create group")
		}
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// List handles GET /groups
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	groups, err := h.service.List(userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	response.WriteJSON(w, http.StatusOK, groups)
}

// Get handles GET /groups/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g, err := h.service.GetById(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "group not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch group")
		return
	}
	response.WriteJSON(w, http.StatusOK, g)
}

// Update handles PUT /groups/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var input struct {
		Name          *string `json:"name"`
		SimplifyDebts *bool   `json:"simplify_debts"`
		Currency      *string `json:"currency"`
		IsActive      *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.service.GetById(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "group not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch group")
		return
	}

	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.SimplifyDebts != nil {
		existing.SimplifyDebts = *input.SimplifyDebts
	}
	if input.Currency != nil {
		existing.Currency = *input.Currency
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	updated, err := h.service.Update(id, existing)
	if err != nil {
		if errors.Is(err, ErrNameRequired) {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "group not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to update group")
		return
	}
	response.WriteJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /groups/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	requesterID := auth.UserIDFrom(r.Context())
	if requesterID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	deleted, err := h.service.Delete(id, requesterID)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "group not found")
			return
		}
		if errors.Is(err, ErrNotCreator) {
			response.WriteError(w, http.StatusForbidden, "only group creator can delete the group")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to delete group")
		return
	}
	response.WriteJSON(w, http.StatusOK, fmt.Sprintf("Group '%s' deleted successfully!", deleted.Name))
}

// AddMember handles POST /groups/{group_id}/members
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("group_id")

	var input struct {
		UserId string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.UserId == "" {
		response.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	g, err := h.service.AddMember(groupID, input.UserId)
	if err != nil {
		switch {
		case errors.Is(err, response.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "group not found")
		case errors.Is(err, ErrUserNotFound):
			response.WriteError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, ErrAlreadyMember):
			response.WriteError(w, http.StatusBadRequest, "User is already a member of this group")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to add member")
		}
		return
	}
	response.WriteJSON(w, http.StatusOK, g)
}

// RemoveMember handles DELETE /groups/{group_id}/members/{user_id}
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID := r.PathValue("group_id")
	userID := r.PathValue("user_id")
	requesterID := auth.UserIDFrom(r.Context())
	if requesterID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	err := h.service.RemoveMember(groupID, requesterID, userID)
	if err != nil {
		switch {
		case errors.Is(err, response.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "group not found")
		case errors.Is(err, ErrNotCreator):
			response.WriteError(w, http.StatusForbidden, "only group creator can remove members")
		case errors.Is(err, ErrCannotRemoveCreator):
			response.WriteError(w, http.StatusBadRequest, "Cannot remove the group creator")
		case errors.Is(err, ErrNotMember):
			response.WriteError(w, http.StatusNotFound, "Member not found in group")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to remove member")
		}
		return
	}
	response.WriteJSON(w, http.StatusOK, "Member removed successfully!")
}
