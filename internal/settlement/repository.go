package settlement

import (
	"database/sql"
	"errors"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/google/uuid"
)

const settlementColumns = `id, group_id, paid_by, paid_to, amount, payment_method, COALESCE(note, ''), settled_at, created_at`

// DBStore implements SettlementStore using a SQLite database.
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database to be
// already opened and the schema to be created.
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// InitSchema creates the settlements table if it does not exist.
func (s *DBStore) InitSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS settlements(
		id TEXT PRIMARY KEY,
		group_id TEXT NOT NULL,
		paid_by TEXT NOT NULL,
		paid_to TEXT NOT NULL,
		amount REAL NOT NULL,
		payment_method TEXT NOT NULL DEFAULT 'cash',
		note TEXT,
		settled_at TEXT NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
		FOREIGN KEY (paid_by) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (paid_to) REFERENCES users(id) ON DELETE CASCADE
		);`
	_, err := s.db.Exec(query)
	return err
}

// CreateIndexes creates the indexes used by settlement queries.
func (s *DBStore) CreateIndexes() error {
	const groupIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_settlements_group_id ON settlements(group_id);`
	_, err := s.db.Exec(groupIDIndex)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanSettlement reads a single settlement row and converts stored values to the model.
func scanSettlement(scanner rowScanner) (Settlement, error) {
	var (
		st                   Settlement
		settledStr, createdS string
	)

	if err := scanner.Scan(&st.ID, &st.GroupId, &st.PaidBy, &st.PaidTo, &st.Amount, &st.PaymentMethod, &st.Note, &settledStr, &createdS); err != nil {
		return Settlement{}, err
	}

	var err error
	if st.SettledAt, err = time.Parse(time.RFC3339, settledStr); err != nil {
		return Settlement{}, err
	}
	if st.CreatedAt, err = time.Parse(time.RFC3339, createdS); err != nil {
		return Settlement{}, err
	}
	return st, nil
}

// Create inserts a new settlement.
func (s *DBStore) Create(st Settlement) (Settlement, error) {
	now := time.Now()
	st.ID = uuid.NewString()
	st.SettledAt = now
	st.CreatedAt = now
	formattedTS := now.Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO settlements (id, group_id, paid_by, paid_to, amount, payment_method, note, settled_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.ID,
		st.GroupId,
		st.PaidBy,
		st.PaidTo,
		st.Amount,
		st.PaymentMethod,
		nullIfEmpty(st.Note),
		formattedTS,
		formattedTS,
	)
	if err != nil {
		return Settlement{}, err
	}
	return st, nil
}

// GetById returns a settlement by id.
func (s *DBStore) GetById(id string) (Settlement, error) {
	row := s.db.QueryRow(
		"SELECT "+settlementColumns+" FROM settlements WHERE id = ?",
		id,
	)

	st, err := scanSettlement(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Settlement{}, response.ErrNotFound
		}
		return Settlement{}, err
	}
	return st, nil
}

// ListByGroup returns a group's settlements, most recent first.
func (s *DBStore) ListByGroup(groupID string) ([]Settlement, error) {
	rows, err := s.db.Query(
		"SELECT "+settlementColumns+" FROM settlements WHERE group_id = ? ORDER BY settled_at DESC, created_at DESC",
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settlements := make([]Settlement, 0)
	for rows.Next() {
		st, err := scanSettlement(rows)
		if err != nil {
			return []Settlement{}, err
		}
		settlements = append(settlements, st)
	}
	return settlements, rows.Err()
}

// Delete removes a settlement.
func (s *DBStore) Delete(id string) (Settlement, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Settlement{}, err
	}

	_, err = s.db.Exec(`DELETE FROM settlements WHERE id = ?`, id)
	if err != nil {
		return Settlement{}, err
	}
	return existing, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
