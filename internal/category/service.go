package category

import "errors"

var (
	ErrGroupRequired = errors.New("group_id is required")
	ErrNameRequired  = errors.New("name is required")
	ErrInvalidAccess = errors.New("invalid access")
)

// CategoryStore is the persistence layer required by Service.
type CategoryStore interface {
	Create(c Category) (Category, error)
	GetById(id string) (Category, error)
	Update(id, groupID string, c Category) (Category, error)
	ListByGroup(groupID string) ([]Category, error)
	Delete(id, groupID string) (Category, error)
}

// Service holds the category business rules and delegates persistence to a CategoryStore.
type Service struct {
	store CategoryStore
}

func NewService(store CategoryStore) *Service {
	return &Service{store: store}
}

// Create validates and creates a category. New categories are active by default.
func (s *Service) Create(c Category) (Category, error) {
	if c.GroupId == "" {
		return Category{}, ErrGroupRequired
	}
	if c.Name == "" {
		return Category{}, ErrNameRequired
	}
	c.IsActive = true
	return s.store.Create(c)
}

// GetById returns a category scoped to its group.
func (s *Service) GetById(id, groupID string) (Category, error) {
	c, err := s.store.GetById(id)
	if err != nil {
		return Category{}, err
	}
	if c.GroupId != groupID {
		return Category{}, ErrInvalidAccess
	}
	return c, nil
}

// ListByGroup returns all categories for a group.
func (s *Service) ListByGroup(groupID string) ([]Category, error) {
	return s.store.ListByGroup(groupID)
}

// Update modifies a category scoped to its group.
func (s *Service) Update(id, groupID string, c Category) (Category, error) {
	if c.Name == "" {
		return Category{}, ErrNameRequired
	}
	return s.store.Update(id, groupID, c)
}

// Delete removes a category scoped to its group.
func (s *Service) Delete(id, groupID string) (Category, error) {
	return s.store.Delete(id, groupID)
}
