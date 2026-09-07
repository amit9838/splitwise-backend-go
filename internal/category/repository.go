package category

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Categories are public
// TODO: make it grouped scopped in future
type CategoryStore interface {
	Create(c Category) (Category, error)
	GetById(id string) (Category, error)
	Update(id string, g_id string, c Category) (Category, error)
	ListByGroup(g_id string) ([]Category, error)
	Delete(id string, g_id string) (Category, error)
}

// DBStore implements Store using a SQLite database
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database
// to be already opened and the schema to be created.

func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

func (s *DBStore) InitSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS categories(
		id TEXT PRIMARY KEY,
		group_id TEXT NOT NULL,
		name TEXT NOT NULL,
		is_active INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
		);`
	_, err := s.db.Exec(query)
	return err
}

func (s *DBStore) CreateIndexes() error {
	const query = `
	CREATE INDEX IF NOT EXISTS idx_categories_group_id ON categories(group_id);
	`
	_, err := s.db.Exec(query)
	return err
}

// Create category
func (s *DBStore) Create(c Category) (Category, error) {
	now := time.Now()
	c.ID = uuid.NewString()
	c.CreatedAt = now
	c.UpdatedAt = now

	// format timestamp
	formatted_ts := now.Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO categories (id, group_id, name, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID,
		c.GroupId,
		c.Name,
		boolToInt(c.IsActive),
		formatted_ts,
		formatted_ts,
	)
	if err != nil {
		return Category{}, err
	}
	return c, nil
}

// Get category
func (s *DBStore) GetById(id string) (Category, error) {
	var c Category
	var isActiveInt int
	var createdStr, updatedStr string
	row := s.db.QueryRow(
		"SELECT id, group_id, name, is_active, created_at, updated_at from categories where id = ?",
		id,
	)

	err := row.Scan(&c.ID, &c.GroupId, &c.Name, &isActiveInt, &createdStr, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, ErrNotFound
		}
		return Category{}, err
	}
	c.IsActive = intToBool(isActiveInt)
	c.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return Category{}, err
	}
	c.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr)
	if err != nil {
		return Category{}, err
	}
	return c, nil
}

// List categories
func (s *DBStore) ListByGroup(g_id string) ([]Category, error) {
	rows, err := s.db.Query(
		"SELECT id, group_id, name, is_active, created_at, updated_at from categories where group_id = ?",
		g_id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// New structures categories
	categories := make([]Category, 0)
	for rows.Next() {
		var c Category
		var isActiveInt int
		var createdStr, updatedStr string

		err := rows.Scan(&c.ID, &c.GroupId, &c.Name, &isActiveInt, &createdStr, &updatedStr)
		if err != nil {
			return []Category{}, err
		}
		c.IsActive = intToBool(isActiveInt)
		c.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
		if err != nil {
			return []Category{}, err
		}
		c.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr)
		if err != nil {
			return []Category{}, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// Update categories
func (s *DBStore) Update(id string, g_id string, c Category) (Category, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Category{}, err
	}
	if existing.GroupId != g_id {
		return Category{}, ErrNotFound
	}

	now := time.Now()
	formatted_ts := now.Format(time.RFC3339)

	_, err = s.db.Exec(
		`UPDATE categories SET name = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		c.Name,
		boolToInt(c.IsActive),
		formatted_ts,
		id,
	)
	if err != nil {
		return Category{}, err
	}

	return s.GetById(id)
}

// Delete categories
func (s *DBStore) Delete(id string, g_id string) (Category, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Category{}, err
	}
	if existing.GroupId != g_id {
		return Category{}, ErrNotFound
	}

	_, err = s.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return Category{}, err
	}

	return existing, nil
}
