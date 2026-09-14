package expense

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
	"github.com/google/uuid"
)

const expenseColumns = `id, group_id, category_id, paid_by, amount, description, currency, split_type, expense_date, is_active, created_at, updated_at`

// DBStore implements ExpenseStore using a SQLite database
type DBStore struct {
	db *sql.DB
}

// NewDBStore creates a new DBStore. It expects the database to be
// already opened and the schema to be created
func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// InitSchema creates the expenses and expense_splits tables if they do
// not exist, and migrates older expenses tables by adding any missing
// columns.
func (s *DBStore) InitSchema() error {
	const query = `
	CREATE TABLE IF NOT EXISTS expenses(
	id TEXT PRIMARY KEY,
	group_id TEXT NOT NULL,
	paid_by TEXT NOT NULL,
	category_id TEXT NOT NULL,
	amount REAL NOT NULL,
	description TEXT,
	currency TEXT NOT NULL DEFAULT 'INR',
	split_type TEXT NOT NULL DEFAULT 'EQUAL',
	expense_date TEXT NOT NULL,
	is_active INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
	FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
	);`
	const splitsQuery = `
	CREATE TABLE IF NOT EXISTS expense_splits(
	id TEXT PRIMARY KEY,
	expense_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	amount REAL NOT NULL,
	percentage REAL,
	shares INTEGER,
	FOREIGN KEY (expense_id) REFERENCES expenses(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	if _, err := s.db.Exec(query); err != nil {
		return err
	}
	if _, err := s.db.Exec(splitsQuery); err != nil {
		return err
	}
	return s.migrate()
}

// migrate adds columns introduced after the first release to expenses
// tables created by older versions of the schema.
func (s *DBStore) migrate() error {
	cols, err := s.tableColumns("expenses")
	if err != nil {
		return err
	}
	if !cols["currency"] {
		if _, err := s.db.Exec(`ALTER TABLE expenses ADD COLUMN currency TEXT NOT NULL DEFAULT 'INR'`); err != nil {
			return err
		}
	}
	if !cols["is_active"] {
		if _, err := s.db.Exec(`ALTER TABLE expenses ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1`); err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStore) tableColumns(table string) (map[string]bool, error) {
	rows, err := s.db.Query("SELECT name FROM pragma_table_info('" + table + "')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// Create Indexes
func (s *DBStore) CreateIndexes() error {
	const groupIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_expense_group_id ON expenses(group_id);`
	const categoryIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_expense_category_id ON expenses(category_id);`
	const expenseIDSplitIndex = `
	CREATE INDEX IF NOT EXISTS idx_expense_splits_expense_id ON expense_splits(expense_id);`
	const userIDSplitIndex = `
	CREATE INDEX IF NOT EXISTS idx_expense_splits_user_id ON expense_splits(user_id);`

	for _, q := range []string{groupIDIndex, categoryIDIndex, expenseIDSplitIndex, userIDSplitIndex} {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanExpense reads a single expense row and converts stored values to the model.
func scanExpense(scanner rowScanner) (Expense, error) {
	var (
		e                                      Expense
		isActiveInt                            int
		expenseDateStr, createdStr, updatedStr string
	)

	if err := scanner.Scan(&e.ID, &e.GroupId, &e.CategoryId, &e.PaidBy, &e.Amount, &e.Description, &e.Currency, &e.SplitType, &expenseDateStr, &isActiveInt, &createdStr, &updatedStr); err != nil {
		return Expense{}, err
	}
	e.IsActive = util.IntToBool(isActiveInt)

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

// scanSplit reads a single split row and converts stored values to the model.
func scanSplit(scanner rowScanner) (Split, error) {
	var (
		sp         Split
		percentage sql.NullFloat64
		shares     sql.NullInt64
	)

	if err := scanner.Scan(&sp.ID, &sp.ExpenseId, &sp.UserId, &sp.Amount, &percentage, &shares); err != nil {
		return Split{}, err
	}
	if percentage.Valid {
		p := percentage.Float64
		sp.Percentage = &p
	}
	if shares.Valid {
		sh := int(shares.Int64)
		sp.Shares = &sh
	}
	return sp, nil
}

// isForeignKeyViolation reports whether err is a SQLite FOREIGN KEY constraint failure.
func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

// CreateWithSplits inserts an expense and its splits in a single transaction.
func (s *DBStore) CreateWithSplits(e Expense, splits []Split) (Expense, []Split, error) {
	now := time.Now()
	e.ID = uuid.NewString()
	e.CreatedAt = now
	e.UpdatedAt = now
	e.IsActive = true
	formattedTS := now.Format(time.RFC3339)
	expenseDateTS := e.ExpenseDate.Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return Expense{}, nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO expenses (
		id, group_id, category_id, paid_by, amount, description, currency, split_type, expense_date, is_active, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID,
		e.GroupId,
		e.CategoryId,
		e.PaidBy,
		e.Amount,
		e.Description,
		e.Currency,
		e.SplitType,
		expenseDateTS,
		util.BoolToInt(e.IsActive),
		formattedTS,
		formattedTS,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			return Expense{}, nil, ErrCategoryNotFound
		}
		return Expense{}, nil, err
	}

	const splitInsert = `INSERT INTO expense_splits (id, expense_id, user_id, amount, percentage, shares) VALUES (?, ?, ?, ?, ?, ?)`
	stored := make([]Split, 0, len(splits))
	for _, sp := range splits {
		sp.ID = uuid.NewString()
		sp.ExpenseId = e.ID
		if _, err := tx.Exec(splitInsert, sp.ID, sp.ExpenseId, sp.UserId, sp.Amount, sp.Percentage, sp.Shares); err != nil {
			return Expense{}, nil, err
		}
		stored = append(stored, sp)
	}

	if err := tx.Commit(); err != nil {
		return Expense{}, nil, err
	}
	e.Splits = stored
	return e, stored, nil
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

// ListByGroup returns active expenses belonging to a group.
func (s *DBStore) ListByGroup(groupID string) ([]Expense, error) {
	rows, err := s.db.Query(
		"SELECT "+expenseColumns+" FROM expenses WHERE group_id = ? AND is_active = 1",
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

// ListSplitsByExpenseID returns the splits of one expense.
func (s *DBStore) ListSplitsByExpenseID(expenseID string) ([]Split, error) {
	rows, err := s.db.Query(
		"SELECT id, expense_id, user_id, amount, percentage, shares FROM expense_splits WHERE expense_id = ?",
		expenseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	splits := make([]Split, 0)
	for rows.Next() {
		sp, err := scanSplit(rows)
		if err != nil {
			return []Split{}, err
		}
		splits = append(splits, sp)
	}
	return splits, rows.Err()
}

// ListSplitsByGroup returns the splits of all active expenses in a group.
func (s *DBStore) ListSplitsByGroup(groupID string) ([]Split, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.expense_id, s.user_id, s.amount, s.percentage, s.shares
		FROM expense_splits s
		JOIN expenses e ON s.expense_id = e.id
		WHERE e.group_id = ? AND e.is_active = 1`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	splits := make([]Split, 0)
	for rows.Next() {
		sp, err := scanSplit(rows)
		if err != nil {
			return []Split{}, err
		}
		splits = append(splits, sp)
	}
	return splits, rows.Err()
}

// Update modifies an expense's mutable fields. Splits are not recomputed.
func (s *DBStore) Update(id string, e Expense) (Expense, error) {
	if _, err := s.GetById(id); err != nil {
		return Expense{}, err
	}

	_, err := s.db.Exec(
		`UPDATE expenses SET amount = ?, description = ?, currency = ?, split_type = ?, expense_date = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		e.Amount,
		e.Description,
		e.Currency,
		e.SplitType,
		e.ExpenseDate.Format(time.RFC3339),
		util.BoolToInt(e.IsActive),
		time.Now().Format(time.RFC3339),
		id,
	)
	if err != nil {
		return Expense{}, err
	}
	return s.GetById(id)
}

// Delete removes an expense. Its splits are removed by the foreign key cascade.
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
