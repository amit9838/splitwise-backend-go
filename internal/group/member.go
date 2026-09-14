package group

import "time"

// Member is a group membership row.
type Member struct {
	ID       string     `json:"id"`
	GroupId  string     `json:"-"`
	UserId   string     `json:"user_id"`
	User     *UserBrief `json:"user,omitempty"`
	JoinedAt time.Time  `json:"joined_at"`
	IsActive bool       `json:"is_active"`
}

// UserBrief is the user info embedded in member payloads.
type UserBrief struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}
