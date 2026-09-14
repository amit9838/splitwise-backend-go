package user

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
	"github.com/google/uuid"
)

// userColumns uses COALESCE for nullable columns so they scan directly
// into the model. It is only valid in SELECT statements.
const userColumns = `id, email, hashed_password, COALESCE(full_name, ''), is_active, created_at, updated_at`

// DBStore implements UserStore using a SQLite database.
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database to be
// already opened and the schema to be created.
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// InitSchema creates the users table if it does not exist.
func (s *DBStore) InitSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS users(
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		hashed_password TEXT NOT NULL,
		full_name TEXT,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT
		);`
	_, err := s.db.Exec(query)
	return err
}

// CreateIndexes creates the indexes used by user queries. The unique
// index enforces one account per email address, case-insensitively.
func (s *DBStore) CreateIndexes() error {
	const query = `
	DROP INDEX IF EXISTS idx_users_email;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email COLLATE NOCASE);`
	_, err := s.db.Exec(query)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanUser reads a single user row and converts stored values to the model.
func scanUser(scanner rowScanner) (User, error) {
	var (
		u           User
		isActiveInt int
		createdStr  string
		updatedStr  sql.NullString
	)

	if err := scanner.Scan(&u.ID, &u.Email, &u.HashedPassword, &u.FullName, &isActiveInt, &createdStr, &updatedStr); err != nil {
		return User{}, err
	}
	u.IsActive = util.IntToBool(isActiveInt)

	var err error
	if u.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		return User{}, err
	}
	if updatedStr.Valid {
		updated, err := time.Parse(time.RFC3339, updatedStr.String)
		if err != nil {
			return User{}, err
		}
		u.UpdatedAt = &updated
	}
	return u, nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// Create inserts a new user.
func (s *DBStore) Create(u User) (User, error) {
	now := time.Now()
	u.ID = uuid.NewString()
	u.CreatedAt = now
	u.UpdatedAt = nil
	formattedTS := now.Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO users (id, email, hashed_password, full_name, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULL)`,
		u.ID,
		u.Email,
		u.HashedPassword,
		nullIfEmpty(u.FullName),
		util.BoolToInt(u.IsActive),
		formattedTS,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailRegistered
		}
		return User{}, err
	}
	return u, nil
}

// GetById returns a user by id.
func (s *DBStore) GetById(id string) (User, error) {
	row := s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE id = ?",
		id,
	)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, response.ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

// GetByEmail returns a user by email address.
func (s *DBStore) GetByEmail(email string) (User, error) {
	row := s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE email = ?",
		email,
	)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, response.ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

// List returns all users.
func (s *DBStore) List() ([]User, error) {
	rows, err := s.db.Query("SELECT " + userColumns + " FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return []User{}, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Update modifies a user's mutable fields.
func (s *DBStore) Update(id string, u User) (User, error) {
	if _, err := s.GetById(id); err != nil {
		return User{}, err
	}

	_, err := s.db.Exec(
		`UPDATE users SET email = ?, hashed_password = ?, full_name = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		u.Email,
		u.HashedPassword,
		nullIfEmpty(u.FullName),
		util.BoolToInt(u.IsActive),
		time.Now().Format(time.RFC3339),
		id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailRegistered
		}
		return User{}, err
	}
	return s.GetById(id)
}

// Delete removes a user.
func (s *DBStore) Delete(id string) (User, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return User{}, err
	}

	_, err = s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return User{}, err
	}
	return existing, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
