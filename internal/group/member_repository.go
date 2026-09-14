package group

import (
	"database/sql"
	"errors"
	"time"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
	"github.com/google/uuid"
)

const memberColumns = `id, group_id, user_id, is_active, joined_at`

// MemberDBStore implements MemberStore using a SQLite database.
type MemberDBStore struct {
	db *sql.DB
}

// NewMemberDBStore creates a new MemberDBStore. It expects the database
// to be already opened and the schema to be created.
func NewMemberDBStore(db *sql.DB) *MemberDBStore {
	return &MemberDBStore{db: db}
}

// InitSchema creates the group_members table if it does not exist.
func (s *MemberDBStore) InitSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS group_members(
		id TEXT PRIMARY KEY,
		group_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		is_active INTEGER NOT NULL DEFAULT 1,
		joined_at TEXT NOT NULL,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(group_id, user_id)
		);`
	_, err := s.db.Exec(query)
	return err
}

// CreateIndexes creates the indexes used by membership queries.
func (s *MemberDBStore) CreateIndexes() error {
	const groupIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);`
	const userIDIndex = `
	CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id);`

	if _, err := s.db.Exec(groupIDIndex); err != nil {
		return err
	}
	_, err := s.db.Exec(userIDIndex)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type memberRowScanner interface {
	Scan(dest ...any) error
}

// scanMember reads a single membership row and converts stored values to the model.
func scanMember(scanner memberRowScanner) (Member, error) {
	var (
		m           Member
		isActiveInt int
		joinedStr   string
	)

	if err := scanner.Scan(&m.ID, &m.GroupId, &m.UserId, &isActiveInt, &joinedStr); err != nil {
		return Member{}, err
	}
	m.IsActive = util.IntToBool(isActiveInt)

	var err error
	if m.JoinedAt, err = time.Parse(time.RFC3339, joinedStr); err != nil {
		return Member{}, err
	}
	return m, nil
}

// GetByGroupAndUser returns a membership row (active or not) for a group and user.
func (s *MemberDBStore) GetByGroupAndUser(groupID, userID string) (Member, error) {
	row := s.db.QueryRow(
		"SELECT "+memberColumns+" FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID,
		userID,
	)

	m, err := scanMember(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Member{}, response.ErrNotFound
		}
		return Member{}, err
	}
	return m, nil
}

// ListActiveByGroup returns the active memberships of a group.
func (s *MemberDBStore) ListActiveByGroup(groupID string) ([]Member, error) {
	rows, err := s.db.Query(
		"SELECT "+memberColumns+" FROM group_members WHERE group_id = ? AND is_active = 1",
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]Member, 0)
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return []Member{}, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// ListGroupIDsByUser returns ids of groups the user is an active member of.
func (s *MemberDBStore) ListGroupIDsByUser(userID string) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT group_id FROM group_members WHERE user_id = ? AND is_active = 1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groupIDs := make([]string, 0)
	for rows.Next() {
		var gid string
		if err := rows.Scan(&gid); err != nil {
			return []string{}, err
		}
		groupIDs = append(groupIDs, gid)
	}
	return groupIDs, rows.Err()
}

// AddMember adds a user to a group. Re-adding a previously removed
// user reactivates the original membership row.
func (s *MemberDBStore) AddMember(groupID, userID string) (Member, error) {
	existing, err := s.GetByGroupAndUser(groupID, userID)
	if err == nil {
		if existing.IsActive {
			return Member{}, ErrAlreadyMember
		}
		_, err = s.db.Exec(
			`UPDATE group_members SET is_active = 1 WHERE id = ?`,
			existing.ID,
		)
		if err != nil {
			return Member{}, err
		}
		existing.IsActive = true
		return existing, nil
	}
	if !errors.Is(err, response.ErrNotFound) {
		return Member{}, err
	}

	now := time.Now()
	m := Member{
		ID:       uuid.NewString(),
		GroupId:  groupID,
		UserId:   userID,
		JoinedAt: now,
		IsActive: true,
	}
	_, err = s.db.Exec(
		`INSERT INTO group_members (id, group_id, user_id, is_active, joined_at) VALUES (?, ?, ?, 1, ?)`,
		m.ID,
		m.GroupId,
		m.UserId,
		now.Format(time.RFC3339),
	)
	if err != nil {
		return Member{}, err
	}
	return m, nil
}

// RemoveMember soft-deletes a membership (is_active = 0).
func (s *MemberDBStore) RemoveMember(groupID, userID string) error {
	res, err := s.db.Exec(
		`UPDATE group_members SET is_active = 0 WHERE group_id = ? AND user_id = ? AND is_active = 1`,
		groupID,
		userID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotMember
	}
	return nil
}
