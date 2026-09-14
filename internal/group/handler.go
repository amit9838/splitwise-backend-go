package group

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

// GroupService is the business layer used by the handler.
type GroupService interface {
	Create(g Group) (Group, error)
	GetById(id string) (Group, error)
	List() ([]Group, error)
	Update(id string, g Group) (Group, error)
	Delete(id string, requesterID string) (Group, error)
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
	var input struct {
		Name          string `json:"name"`
		CreatedBy     string `json:"created_by"`
		SimplifyDebts *bool  `json:"simplify_debts"`
		Currency      string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" || input.CreatedBy == "" {
		response.WriteError(w, http.StatusBadRequest, "name and created_by are required")
		return
	}

	simplifyDebts := true
	if input.SimplifyDebts != nil {
		simplifyDebts = *input.SimplifyDebts
	}

	created, err := h.service.Create(Group{
		Name:          input.Name,
		CreatedBy:     input.CreatedBy,
		SimplifyDebts: simplifyDebts,
		Currency:      input.Currency,
	})
	if err != nil {
		if errors.Is(err, ErrNameRequired) || errors.Is(err, ErrCreatorRequired) {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to create group")
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// List handles GET /groups
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.List()
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
	requesterID := r.Header.Get("X-User-ID")
	if requesterID == "" {
		response.WriteError(w, http.StatusBadRequest, "X-User-ID header is required")
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
