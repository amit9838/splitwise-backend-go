package group

import (
	"database/sql"
	"errors"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
	"github.com/google/uuid"
)

const groupColumns = `id, name, created_by, simplify_debts, currency, is_active, created_at, updated_at`

// DBStore implements GroupStore using a SQLite database.
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database to be
// already opened and the schema to be created.
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// InitSchema creates the groups table if it does not exist.
func (s *DBStore) InitSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS groups(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_by TEXT NOT NULL,
		simplify_debts INTEGER NOT NULL DEFAULT 1,
		currency TEXT NOT NULL DEFAULT 'INR',
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
		);`
	_, err := s.db.Exec(query)
	return err
}

// CreateIndexes creates the indexes used by group queries.
func (s *DBStore) CreateIndexes() error {
	const query = `
	CREATE INDEX IF NOT EXISTS idx_groups_created_by ON groups(created_by);
	`
	_, err := s.db.Exec(query)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanGroup reads a single group row and converts stored values to the model.
func scanGroup(scanner rowScanner) (Group, error) {
	var (
		g                        Group
		simplifyInt, isActiveInt int
		createdStr, updatedStr   string
	)

	if err := scanner.Scan(&g.ID, &g.Name, &g.CreatedBy, &simplifyInt, &g.Currency, &isActiveInt, &createdStr, &updatedStr); err != nil {
		return Group{}, err
	}
	g.SimplifyDebts = util.IntToBool(simplifyInt)
	g.IsActive = util.IntToBool(isActiveInt)

	var err error
	if g.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		return Group{}, err
	}
	if g.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr); err != nil {
		return Group{}, err
	}
	return g, nil
}

// CreateWithMembers inserts a group, the creator's membership and any
// additional member memberships in a single transaction.
func (s *DBStore) CreateWithMembers(g Group, creatorID string, memberIDs []string) (Group, error) {
	now := time.Now()
	g.ID = uuid.NewString()
	g.CreatedAt = now
	g.UpdatedAt = now
	formattedTS := now.Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO groups (id, name, created_by, simplify_debts, currency, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ID,
		g.Name,
		g.CreatedBy,
		util.BoolToInt(g.SimplifyDebts),
		g.Currency,
		util.BoolToInt(g.IsActive),
		formattedTS,
		formattedTS,
	)
	if err != nil {
		return Group{}, err
	}

	const memberInsert = `INSERT INTO group_members (id, group_id, user_id, is_active, joined_at) VALUES (?, ?, ?, 1, ?)`
	_, err = tx.Exec(memberInsert, uuid.NewString(), g.ID, creatorID, formattedTS)
	if err != nil {
		return Group{}, err
	}
	for _, uid := range memberIDs {
		_, err = tx.Exec(memberInsert, uuid.NewString(), g.ID, uid, formattedTS)
		if err != nil {
			return Group{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Group{}, err
	}
	return g, nil
}

// GetById returns a group by id.
func (s *DBStore) GetById(id string) (Group, error) {
	row := s.db.QueryRow(
		"SELECT "+groupColumns+" FROM groups WHERE id = ?",
		id,
	)

	g, err := scanGroup(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Group{}, response.ErrNotFound
		}
		return Group{}, err
	}
	return g, nil
}

// List returns all groups.
func (s *DBStore) List() ([]Group, error) {
	rows, err := s.db.Query("SELECT " + groupColumns + " FROM groups")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]Group, 0)
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return []Group{}, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// Update modifies a group's mutable fields.
func (s *DBStore) Update(id string, g Group) (Group, error) {
	if _, err := s.GetById(id); err != nil {
		return Group{}, err
	}

	_, err := s.db.Exec(
		`UPDATE groups SET name = ?, simplify_debts = ?, currency = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		g.Name,
		util.BoolToInt(g.SimplifyDebts),
		g.Currency,
		util.BoolToInt(g.IsActive),
		time.Now().Format(time.RFC3339),
		id,
	)
	if err != nil {
		return Group{}, err
	}
	return s.GetById(id)
}

// Delete removes a group.
func (s *DBStore) Delete(id string) (Group, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Group{}, err
	}

	_, err = s.db.Exec(`DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return Group{}, err
	}
	return existing, nil
}
