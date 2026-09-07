package category

import "time"

type Category struct {
	ID        string    `json:"id"`
	GroupId   string    `json:"group_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
