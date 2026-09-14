package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

// UserService is the business layer used by the handler.
type UserService interface {
	Register(email, password, fullName string) (User, error)
	Login(email, password string) (User, error)
	GetById(id string) (User, error)
	List() ([]User, error)
	Update(id string, u User, newPassword string) (User, error)
	Delete(id string) (User, error)
}

// Handler
type Handler struct {
	service UserService
	auth    *auth.Manager
}

// NewHandler creates a new user Handler.
func NewHandler(s UserService, a *auth.Manager) *Handler {
	return &Handler{service: s, auth: a}
}

func (h *Handler) writeTokenPair(w http.ResponseWriter, userID string) {
	access, refresh, err := h.auth.NewTokenPair(userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "bearer",
	})
}

// Register handles POST /auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.service.Register(input.Email, input.Password, input.FullName)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailRequired),
			errors.Is(err, ErrPasswordRequired),
			errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrEmailRegistered):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to register user")
		}
		return
	}
	response.WriteJSON(w, http.StatusCreated, created)
}

// Login handles POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.service.Login(input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.WriteError(w, http.StatusUnauthorized, err.Error())
		case errors.Is(err, ErrAccountDeactivated):
			response.WriteError(w, http.StatusForbidden, err.Error())
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to login")
		}
		return
	}
	h.writeTokenPair(w, u.ID)
}

// Refresh handles POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.RefreshToken == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	userID, err := h.auth.Parse(input.RefreshToken, auth.TokenTypeRefresh)
	if err != nil {
		response.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}

	u, err := h.service.GetById(userID)
	if err != nil || !u.IsActive {
		response.WriteError(w, http.StatusUnauthorized, "User not found or inactive")
		return
	}
	h.writeTokenPair(w, u.ID)
}

// Me handles GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFrom(r.Context())
	if userID == "" {
		response.WriteError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	u, err := h.service.GetById(userID)
	if err != nil {
		response.WriteError(w, http.StatusUnauthorized, "User not found")
		return
	}
	response.WriteJSON(w, http.StatusOK, u)
}

// List handles GET /users
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List()
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	response.WriteJSON(w, http.StatusOK, users)
}

// Get handles GET /users/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	u, err := h.service.GetById(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	response.WriteJSON(w, http.StatusOK, u)
}

// Create handles POST /users
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.Register(w, r)
}

// Update handles PUT /users/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var input struct {
		Email    *string `json:"email"`
		FullName *string `json:"full_name"`
		Password *string `json:"password"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.service.GetById(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	if input.Email != nil {
		existing.Email = *input.Email
	}
	if input.FullName != nil {
		existing.FullName = *input.FullName
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}
	newPassword := ""
	if input.Password != nil {
		newPassword = *input.Password
	}

	updated, err := h.service.Update(id, existing, newPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailRequired),
			errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrEmailRegistered):
			response.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, response.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "user not found")
		default:
			response.WriteError(w, http.StatusInternalServerError, "failed to update user")
		}
		return
	}
	response.WriteJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /users/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	_, err := h.service.Delete(id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	response.WriteJSON(w, http.StatusOK, "User deleted successfully!")
}
