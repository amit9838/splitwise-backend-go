package group

import (
	"errors"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/user"
)

var (
	ErrNameRequired        = errors.New("name is required")
	ErrCreatorRequired     = errors.New("created_by is required")
	ErrNotCreator          = errors.New("only group creator can perform this action")
	ErrCreatorInMembers    = errors.New("creator is added automatically and must not be in member_ids")
	ErrDuplicateMembers    = errors.New("duplicate member ids provided")
	ErrMembersNotFound     = errors.New("one or more members not found")
	ErrAlreadyMember       = errors.New("user is already a member of this group")
	ErrCannotRemoveCreator = errors.New("cannot remove the group creator")
	ErrNotMember           = errors.New("member not found in group")
	ErrUserNotFound        = errors.New("user not found")
)

// GroupStore is the persistence layer for groups required by Service.
type GroupStore interface {
	CreateWithMembers(g Group, creatorID string, memberIDs []string) (Group, error)
	GetById(id string) (Group, error)
	List() ([]Group, error)
	Update(id string, g Group) (Group, error)
	Delete(id string) (Group, error)
}

// MemberStore is the persistence layer for memberships required by Service.
type MemberStore interface {
	AddMember(groupID, userID string) (Member, error)
	RemoveMember(groupID, userID string) error
	GetByGroupAndUser(groupID, userID string) (Member, error)
	ListActiveByGroup(groupID string) ([]Member, error)
	ListGroupIDsByUser(userID string) ([]string, error)
}

// UserLookup provides the user info required by Service.
type UserLookup interface {
	GetById(id string) (user.User, error)
}

// Service holds the group business rules and delegates persistence to
// a GroupStore, a MemberStore and a UserLookup.
type Service struct {
	store   GroupStore
	members MemberStore
	users   UserLookup
}

func NewService(store GroupStore, members MemberStore, users UserLookup) *Service {
	return &Service{store: store, members: members, users: users}
}

// attachMembers loads a group's active members with their user info.
func (s *Service) attachMembers(g *Group) error {
	members, err := s.members.ListActiveByGroup(g.ID)
	if err != nil {
		return err
	}
	for i := range members {
		u, err := s.users.GetById(members[i].UserId)
		if err != nil {
			return err
		}
		members[i].User = &UserBrief{ID: u.ID, Email: u.Email, FullName: u.FullName}
	}
	if members == nil {
		members = make([]Member, 0)
	}
	g.Members = members
	return nil
}

// Create validates and creates a group. The creator is always added as
// a member, along with any extra memberIDs, in one transaction.
func (s *Service) Create(g Group, creatorID string, memberIDs []string) (Group, error) {
	if g.Name == "" {
		return Group{}, ErrNameRequired
	}
	if creatorID == "" {
		return Group{}, ErrCreatorRequired
	}
	if g.Currency == "" {
		g.Currency = "INR"
	}

	seen := make(map[string]bool, len(memberIDs))
	for _, uid := range memberIDs {
		if uid == creatorID {
			return Group{}, ErrCreatorInMembers
		}
		if seen[uid] {
			return Group{}, ErrDuplicateMembers
		}
		seen[uid] = true
		if _, err := s.users.GetById(uid); err != nil {
			return Group{}, ErrMembersNotFound
		}
	}

	g.CreatedBy = creatorID
	g.IsActive = true

	created, err := s.store.CreateWithMembers(g, creatorID, memberIDs)
	if err != nil {
		return Group{}, err
	}
	if err := s.attachMembers(&created); err != nil {
		return Group{}, err
	}
	return created, nil
}

// GetById returns a single group with its members.
func (s *Service) GetById(id string) (Group, error) {
	g, err := s.store.GetById(id)
	if err != nil {
		return Group{}, err
	}
	if err := s.attachMembers(&g); err != nil {
		return Group{}, err
	}
	return g, nil
}

// List returns the groups the user is an active member of, with members.
func (s *Service) List(userID string) ([]Group, error) {
	groupIDs, err := s.members.ListGroupIDsByUser(userID)
	if err != nil {
		return nil, err
	}

	groups := make([]Group, 0)
	for _, gid := range groupIDs {
		g, err := s.store.GetById(gid)
		if err != nil {
			if errors.Is(err, response.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if err := s.attachMembers(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// Update modifies a group's mutable fields.
func (s *Service) Update(id string, g Group) (Group, error) {
	if g.Name == "" {
		return Group{}, ErrNameRequired
	}
	updated, err := s.store.Update(id, g)
	if err != nil {
		return Group{}, err
	}
	if err := s.attachMembers(&updated); err != nil {
		return Group{}, err
	}
	return updated, nil
}

// Delete removes a group. Only the creator may delete it; memberships
// are removed by the foreign key cascade.
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

// AddMember adds a user to a group and returns the updated group.
func (s *Service) AddMember(groupID, userID string) (Group, error) {
	if _, err := s.store.GetById(groupID); err != nil {
		return Group{}, err
	}
	if _, err := s.users.GetById(userID); err != nil {
		return Group{}, ErrUserNotFound
	}
	if _, err := s.members.AddMember(groupID, userID); err != nil {
		return Group{}, err
	}
	return s.GetById(groupID)
}

// RemoveMember soft-deletes a membership. Only the creator can remove
// members, and the creator themselves cannot be removed.
func (s *Service) RemoveMember(groupID, requesterID, userID string) error {
	g, err := s.store.GetById(groupID)
	if err != nil {
		return err
	}
	if g.CreatedBy != requesterID {
		return ErrNotCreator
	}
	if userID == g.CreatedBy {
		return ErrCannotRemoveCreator
	}
	return s.members.RemoveMember(groupID, userID)
}
