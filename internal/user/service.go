package user

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailRequired      = errors.New("email is required")
	ErrPasswordRequired   = errors.New("password is required")
	ErrInvalidEmail       = errors.New("email must be a valid email address")
	ErrEmailRegistered    = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountDeactivated = errors.New("account is deactivated")
	ErrUserNotFound       = errors.New("user not found or inactive")
)

// UserStore is the persistence layer required by Service.
type UserStore interface {
	Create(u User) (User, error)
	GetById(id string) (User, error)
	GetByEmail(email string) (User, error)
	List() ([]User, error)
	Update(id string, u User) (User, error)
	Delete(id string) (User, error)
}

// Service holds the user business rules and delegates persistence to a UserStore.
type Service struct {
	store UserStore
}

func NewService(store UserStore) *Service {
	return &Service{store: store}
}

func validEmail(email string) bool {
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-1 && !strings.ContainsAny(email, " \t")
}

// normalizeEmail lowercases and trims an address so uniqueness and login
// are case-insensitive.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func checkPassword(hashed, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}

// Register validates input, hashes the password and creates a user.
func (s *Service) Register(email, password, fullName string) (User, error) {
	email = normalizeEmail(email)
	if email == "" {
		return User{}, ErrEmailRequired
	}
	if !validEmail(email) {
		return User{}, ErrInvalidEmail
	}
	if password == "" {
		return User{}, ErrPasswordRequired
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}

	u, err := s.store.Create(User{
		Email:          email,
		HashedPassword: hashed,
		FullName:       fullName,
		IsActive:       true,
	})
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// Login verifies credentials and returns the user.
func (s *Service) Login(email, password string) (User, error) {
	u, err := s.store.GetByEmail(normalizeEmail(email))
	if err != nil {
		return User{}, ErrInvalidCredentials
	}
	if !u.IsActive {
		return User{}, ErrAccountDeactivated
	}
	if !checkPassword(u.HashedPassword, password) {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}

// GetById returns a user by id.
func (s *Service) GetById(id string) (User, error) {
	return s.store.GetById(id)
}

// List returns all users.
func (s *Service) List() ([]User, error) {
	return s.store.List()
}

// Update modifies a user's mutable fields. newPassword is hashed when
// non-empty; the existing hash is kept otherwise.
func (s *Service) Update(id string, u User, newPassword string) (User, error) {
	u.Email = normalizeEmail(u.Email)
	if u.Email == "" {
		return User{}, ErrEmailRequired
	}
	if !validEmail(u.Email) {
		return User{}, ErrInvalidEmail
	}

	hashed := u.HashedPassword
	if newPassword != "" {
		var err error
		if hashed, err = hashPassword(newPassword); err != nil {
			return User{}, err
		}
	}
	u.HashedPassword = hashed

	return s.store.Update(id, u)
}

// Delete removes a user.
func (s *Service) Delete(id string) (User, error) {
	return s.store.Delete(id)
}
