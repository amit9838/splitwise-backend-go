package category

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler
type Handler struct {
	store CategoryStore
}

// Create handler
func NewHandler(s CategoryStore) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		GroupId string `json:"group_id"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.GroupId == "" || input.Name == "" {
		writeError(w, http.StatusBadRequest, "group_id and name are required")
		return
	}

	new_category := Category{
		GroupId:  input.GroupId,
		Name:     input.Name,
		IsActive: true,
	}
	created, err := h.store.Create(new_category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Get handles GET /categories/{group_id}/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g_id := r.PathValue("group_id")
	category, err := h.store.GetById(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		writeError(w, http.StatusBadRequest, "failed to fetch categories")
		return
	}

	// check group permission
	if category.GroupId != g_id {
		writeError(w, http.StatusForbidden, "Invalid access")
		return
	}

	writeJSON(w, http.StatusOK, category)
}

// List handles GET /todos/{group_id}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	g_id := r.PathValue("group_id")
	categories, err := h.store.ListByGroup(g_id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	writeJSON(w, http.StatusOK, categories)

}

// Get handles PUT /categories/{group_id}/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g_id := r.PathValue("group_id")
	var input struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name == "" || input.IsActive == nil {
		writeError(w, http.StatusBadRequest, "name and is_active are required")
		return
	}

	update_category := Category{
		Name:     input.Name,
		IsActive: *input.IsActive,
	}

	category, err := h.store.Update(id, g_id, update_category)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		writeError(w, http.StatusBadRequest, "failed to update category")
		return
	}

	writeJSON(w, http.StatusOK, category)
}

// List handles DELETE /categories/{group_id}/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g_id := r.PathValue("group_id")
	categories, err := h.store.Delete(id, g_id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}
	writeJSON(w, http.StatusOK, categories)

}
