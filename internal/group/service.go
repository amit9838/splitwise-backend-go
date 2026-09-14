package group

import "errors"

var (
	ErrNameRequired    = errors.New("name is required")
	ErrCreatorRequired = errors.New("created_by is required")
	ErrNotCreator      = errors.New("only group creator can perform this action")
)

// GroupStore is the persistence layer required by Service.
type GroupStore interface {
	Create(g Group) (Group, error)
	GetById(id string) (Group, error)
	List() ([]Group, error)
	Update(id string, g Group) (Group, error)
	Delete(id string) (Group, error)
}

// Service holds the group business rules and delegates persistence to a GroupStore.
type Service struct {
	store GroupStore
}

func NewService(store GroupStore) *Service {
	return &Service{store: store}
}

// Create validates and creates a group. New groups are always active.
func (s *Service) Create(g Group) (Group, error) {
	if g.Name == "" {
		return Group{}, ErrNameRequired
	}
	if g.CreatedBy == "" {
		return Group{}, ErrCreatorRequired
	}
	if g.Currency == "" {
		g.Currency = "INR"
	}
	g.IsActive = true
	return s.store.Create(g)
}

// GetById returns a single group.
func (s *Service) GetById(id string) (Group, error) {
	return s.store.GetById(id)
}

// List returns all groups.
func (s *Service) List() ([]Group, error) {
	return s.store.List()
}

// Update modifies a group's mutable fields.
func (s *Service) Update(id string, g Group) (Group, error) {
	if g.Name == "" {
		return Group{}, ErrNameRequired
	}
	return s.store.Update(id, g)
}

// Delete removes a group. Only the creator may delete it.
func (s *Service) Delete(id string, requesterID string) (Group, error) {
	existing, err := s.store.GetById(id)
	if err != nil {
		return Group{}, err
	}
	if existing.CreatedBy != requesterID {
		return Group{}, ErrNotCreator
	}
	return s.store.Delete(id)
}
