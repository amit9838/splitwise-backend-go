package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("invalid or expired token")
	ErrInvalidTokenType = errors.New("invalid token type")
)

// Claims are the JWT claims carried by both access and refresh tokens.
type Claims struct {
	UserID string `json:"uid"`
	Type   string `json:"typ"`
	jwt.RegisteredClaims
}

// Manager issues and verifies JWT access (30m) and refresh (7d) tokens.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string) *Manager {
	return &Manager{
		secret:     []byte(secret),
		accessTTL:  30 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

func (m *Manager) newToken(userID, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// NewTokenPair returns a fresh access + refresh token pair for userID.
func (m *Manager) NewTokenPair(userID string) (access string, refresh string, err error) {
	access, err = m.newToken(userID, TokenTypeAccess, m.accessTTL)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}
	refresh, err = m.newToken(userID, TokenTypeRefresh, m.refreshTTL)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}
	return access, refresh, nil
}

// Parse validates a token of the expected type and returns its user ID.
func (m *Manager) Parse(token string, expectedType string) (string, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}
	if claims.Type != expectedType {
		return "", ErrInvalidTokenType
	}
	return claims.UserID, nil
}
