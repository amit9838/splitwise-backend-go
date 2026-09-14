package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

type ctxKey struct{}

// UserIDFrom returns the authenticated user ID stored by RequireAuth.
func UserIDFrom(ctx context.Context) string {
	uid, _ := ctx.Value(ctxKey{}).(string)
	return uid
}

// RequireAuth wraps protected handlers, verifying the Bearer access token
// and injecting the authenticated user ID into the request context.
func RequireAuth(m *Manager) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				response.WriteError(w, http.StatusUnauthorized, "Invalid token")
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))

			userID, err := m.Parse(token, TokenTypeAccess)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, err.Error())
				return
			}

			next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, userID)))
		}
	}
}
