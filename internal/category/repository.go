package category

import (
	"database/sql"
	"errors"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
	"github.com/google/uuid"
)

const categoryColumns = `id, group_id, name, is_active, created_at, updated_at`

// DBStore implements CategoryStore using a SQLite database.
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database to be
// already opened and the schema to be created.
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// InitSchema creates the categories table if it does not exist.
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

// CreateIndexes creates the indexes used by category queries.
func (s *DBStore) CreateIndexes() error {
	const query = `
	CREATE INDEX IF NOT EXISTS idx_categories_group_id ON categories(group_id);
	`
	_, err := s.db.Exec(query)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanCategory reads a single category row and converts stored values to the model.
func scanCategory(scanner rowScanner) (Category, error) {
	var (
		c                      Category
		isActiveInt            int
		createdStr, updatedStr string
	)

	if err := scanner.Scan(&c.ID, &c.GroupId, &c.Name, &isActiveInt, &createdStr, &updatedStr); err != nil {
		return Category{}, err
	}
	c.IsActive = util.IntToBool(isActiveInt)

	var err error
	if c.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		return Category{}, err
	}
	if c.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr); err != nil {
		return Category{}, err
	}
	return c, nil
}

// Create inserts a new category.
func (s *DBStore) Create(c Category) (Category, error) {
	now := time.Now()
	c.ID = uuid.NewString()
	c.CreatedAt = now
	c.UpdatedAt = now
	formattedTS := now.Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO categories (id, group_id, name, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID,
		c.GroupId,
		c.Name,
		util.BoolToInt(c.IsActive),
		formattedTS,
		formattedTS,
	)
	if err != nil {
		return Category{}, err
	}
	return c, nil
}

// GetById returns a category by id.
func (s *DBStore) GetById(id string) (Category, error) {
	row := s.db.QueryRow(
		"SELECT "+categoryColumns+" FROM categories WHERE id = ?",
		id,
	)

	c, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, response.ErrNotFound
		}
		return Category{}, err
	}
	return c, nil
}

// ListByGroup returns all categories belonging to a group.
func (s *DBStore) ListByGroup(groupID string) ([]Category, error) {
	rows, err := s.db.Query(
		"SELECT "+categoryColumns+" FROM categories WHERE group_id = ?",
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return []Category{}, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// Update modifies a category scoped to its group.
func (s *DBStore) Update(id, groupID string, c Category) (Category, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Category{}, err
	}
	if existing.GroupId != groupID {
		return Category{}, response.ErrNotFound
	}

	_, err = s.db.Exec(
		`UPDATE categories SET name = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		c.Name,
		util.BoolToInt(c.IsActive),
		time.Now().Format(time.RFC3339),
		id,
	)
	if err != nil {
		return Category{}, err
	}
	return s.GetById(id)
}

// Delete removes a category scoped to its group.
func (s *DBStore) Delete(id, groupID string) (Category, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Category{}, err
	}
	if existing.GroupId != groupID {
		return Category{}, response.ErrNotFound
	}

	_, err = s.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return Category{}, err
	}
	return existing, nil
}
