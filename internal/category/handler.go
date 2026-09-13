package category

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

// CategoryService is the business layer used by the handler.
type CategoryService interface {
	Create(c Category) (Category, error)
	GetById(id, groupID string) (Category, error)
	ListByGroup(groupID string) ([]Category, error)
	Update(id, groupID string, c Category) (Category, error)
	Delete(id, groupID string) (Category, error)
}

// Handler
type Handler struct {
	service CategoryService
}

// NewHandler creates a new category Handler.
func NewHandler(s CategoryService) *Handler {
	return &Handler{service: s}
}

// Create handles POST /categories
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		GroupId string `json:"group_id"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.GroupId == "" || input.Name == "" {
		response.WriteError(w, http.StatusBadRequest, "group_id and name are required")
		return
	}

	created, err := h.service.Create(Category{
		GroupId: input.GroupId,
		Name:    input.Name,
	})
	if err != nil {
		if errors.Is(err, ErrGroupRequired) || errors.Is(err, ErrNameRequired) {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrDuplicateName) {
			response.WriteError(w, http.StatusBadRequest, "Category already exists!")
			return
		}
		if errors.Is(err, ErrGroupNotFound) {
			response.WriteError(w, http.StatusNotFound, "group not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to create category")
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// Get handles GET /categories/{group_id}/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g_id := r.PathValue("group_id")

	category, err := h.service.GetById(id, g_id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "category not found")
			return
		}
		if errors.Is(err, ErrInvalidAccess) {
			response.WriteError(w, http.StatusForbidden, "invalid access")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch category")
		return
	}

	response.WriteJSON(w, http.StatusOK, category)
}

// List handles GET /categories/{group_id}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	g_id := r.PathValue("group_id")

	categories, err := h.service.ListByGroup(g_id)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	response.WriteJSON(w, http.StatusOK, categories)
}

// Update handles PUT /categories/{group_id}/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g_id := r.PathValue("group_id")

	var input struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" || input.IsActive == nil {
		response.WriteError(w, http.StatusBadRequest, "name and is_active are required")
		return
	}

	category, err := h.service.Update(id, g_id, Category{
		Name:     input.Name,
		IsActive: *input.IsActive,
	})
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "category not found")
			return
		}
		if errors.Is(err, ErrNameRequired) {
			response.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrDuplicateName) {
			response.WriteError(w, http.StatusBadRequest, "Category already exists!")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to update category")
		return
	}

	response.WriteJSON(w, http.StatusOK, category)
}

// Delete handles DELETE /categories/{group_id}/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g_id := r.PathValue("group_id")

	deleted, err := h.service.Delete(id, g_id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "category not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}
	response.WriteJSON(w, http.StatusOK, fmt.Sprintf("Category %s deleted successfully!", deleted.Name))
}
