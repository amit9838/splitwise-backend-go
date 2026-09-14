package expense

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/google/uuid"
)

const expenseColumns = `id, 
						group_id, 
						category_id, 
						paid_by, 
						amount, 
						description, 
						split_type, 
						expense_date, 
						created_at, 
						updated_at`

// DBStore implements ExpenseStore using a SQLite database
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database to be
// already opened and the schema to be created
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// Create Expense table if doesn't exist
func (s *DBStore) InitSchema() error {
	const query = `
	CREATE TABLE IF NOT EXISTS expenses(
	id TEXT PRIMARY KEY,
	group_id TEXT NOT NULL,
	paid_by TEXT NOT NULL,
	category_id TEXT NOT NULL,
	amount REAL NOT NULL,
	description TEXT,
	split_type TEXT NOT NULL DEFAULT 'EQUAL',
	expense_date TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
	);`
	_, err := s.db.Exec(query)
	return err
}

// Create Indexes
func (s *DBStore) CreateIndexes() error {
	const groupIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_expense_group_id ON expenses(group_id);`
	const categoryIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_expense_category_id ON expenses(category_id);`
	if _, err := s.db.Exec(groupIDIndex); err != nil {
		return err
	}
	_, err := s.db.Exec(categoryIDIndex)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanExpense reads a single expense row and converts stored values to the model.
func scanExpense(scanner rowScanner) (Expense, error) {
	var (
		e                                      Expense
		expenseDateStr, createdStr, updatedStr string
	)

	if err := scanner.Scan(&e.ID, &e.GroupId, &e.CategoryId, &e.PaidBy, &e.Amount, &e.Description, &e.SplitType, &expenseDateStr, &createdStr, &updatedStr); err != nil {
		return Expense{}, err
	}

	var err error
	if e.ExpenseDate, err = time.Parse(time.RFC3339, expenseDateStr); err != nil {
		return Expense{}, err
	}
	if e.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		return Expense{}, err
	}
	if e.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr); err != nil {
		return Expense{}, err
	}
	return e, nil
}

// Insert a new expense into the database
func (s *DBStore) Create(e Expense) (Expense, error) {
	now := time.Now()
	e.ID = uuid.NewString()
	e.CreatedAt = now
	e.UpdatedAt = now
	formattedTS := now.Format(time.RFC3339)
	expenseDateTS := e.ExpenseDate.Format(time.RFC3339)

	_, err := s.db.Exec(`
		INSERT INTO expenses (
		id, 
		group_id, 
		category_id, 
		paid_by, 
		amount, 
		description, 
		split_type, 
		expense_date, 
		created_at, 
		updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID,
		e.GroupId,
		e.CategoryId,
		e.PaidBy,
		e.Amount,
		e.Description,
		e.SplitType,
		expenseDateTS,
		formattedTS,
		formattedTS,
	)

	if err != nil {
		if isForeignKeyViolation(err) {
			return Expense{}, ErrCategoryNotFound
		}
		return Expense{}, err
	}
	return e, nil
}

// isForeignKeyViolation reports whether err is a SQLite FOREIGN KEY constraint failure.
func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

// GetById returns an expense by id.
func (s *DBStore) GetById(id string) (Expense, error) {
	row := s.db.QueryRow(
		"SELECT "+expenseColumns+" FROM expenses WHERE id = ?",
		id,
	)

	e, err := scanExpense(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Expense{}, response.ErrNotFound
		}
		return Expense{}, err
	}
	return e, nil
}

// List returns all expenses.
func (s *DBStore) List() ([]Expense, error) {
	rows, err := s.db.Query("SELECT " + expenseColumns + " FROM expenses")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := make([]Expense, 0)
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return []Expense{}, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

// ListByGroup returns all expenses belonging to a group.
func (s *DBStore) ListByGroup(groupID string) ([]Expense, error) {
	rows, err := s.db.Query(
		"SELECT "+expenseColumns+" FROM expenses WHERE group_id = ?",
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expenses := make([]Expense, 0)
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return []Expense{}, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

// Update modifies an expense's mutable fields.
func (s *DBStore) Update(id string, e Expense) (Expense, error) {
	if _, err := s.GetById(id); err != nil {
		return Expense{}, err
	}

	_, err := s.db.Exec(
		`UPDATE expenses SET amount = ?, description = ?, split_type = ?, expense_date = ?, updated_at = ? WHERE id = ?`,
		e.Amount,
		e.Description,
		e.SplitType,
		e.ExpenseDate.Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
		id,
	)
	if err != nil {
		return Expense{}, err
	}
	return s.GetById(id)
}

// Delete removes an expense.
func (s *DBStore) Delete(id string) (Expense, error) {
	existing, err := s.GetById(id)
	if err != nil {
		return Expense{}, err
	}

	_, err = s.db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return Expense{}, err
	}
	return existing, nil
}
