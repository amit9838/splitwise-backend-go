package settlement

import (
	"errors"

	"github.com/amit9838/splitwise-backend-go/internal/group"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
)

var (
	ErrGroupIdRequired    = errors.New("group_id is required")
	ErrPaidToRequired     = errors.New("paid_to is required")
	ErrAmountRequired     = errors.New("settlement amount must be positive")
	ErrCannotSettleSelf   = errors.New("cannot settle with yourself")
	ErrRecipientNotMember = errors.New("recipient is not a member of this group")
	ErrPayerNotMember     = errors.New("payer is not a member of this group")
	ErrGroupNotFound      = errors.New("group not found")
	ErrInvalidMethod      = errors.New("payment_method must be one of cash, bank, upi")
	ErrNotPayer           = errors.New("only the payer can delete a settlement")
)

// SettlementStore is the persistence layer required by Service.
type SettlementStore interface {
	Create(st Settlement) (Settlement, error)
	GetById(id string) (Settlement, error)
	ListByGroup(groupID string) ([]Settlement, error)
	Delete(id string) (Settlement, error)
}

// GroupLookup provides the group info required by Service.
type GroupLookup interface {
	GetById(id string) (group.Group, error)
}

// MemberLookup provides the membership info required by Service.
type MemberLookup interface {
	GetByGroupAndUser(groupID, userID string) (group.Member, error)
}

// Service holds the settlement business rules and delegates persistence
// to a SettlementStore, with group and membership lookups for validation.
type Service struct {
	store   SettlementStore
	groups  GroupLookup
	members MemberLookup
}

func NewService(store SettlementStore, groups GroupLookup, members MemberLookup) *Service {
	return &Service{store: store, groups: groups, members: members}
}

func validMethod(method string) bool {
	switch method {
	case "cash", "bank", "upi":
		return true
	default:
		return false
	}
}

// isActiveMember reports whether userID is an active member of the group.
func (s *Service) isActiveMember(groupID, userID string) (bool, error) {
	m, err := s.members.GetByGroupAndUser(groupID, userID)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return m.IsActive, nil
}

// Create validates and records a settlement paid by payerID.
func (s *Service) Create(st Settlement, payerID string) (Settlement, error) {
	if st.GroupId == "" {
		return Settlement{}, ErrGroupIdRequired
	}
	if st.PaidTo == "" {
		return Settlement{}, ErrPaidToRequired
	}
	if st.Amount <= 0 {
		return Settlement{}, ErrAmountRequired
	}
	if st.PaymentMethod == "" {
		st.PaymentMethod = "cash"
	}
	if !validMethod(st.PaymentMethod) {
		return Settlement{}, ErrInvalidMethod
	}

	if _, err := s.groups.GetById(st.GroupId); err != nil {
		return Settlement{}, ErrGroupNotFound
	}
	if st.PaidTo == payerID {
		return Settlement{}, ErrCannotSettleSelf
	}

	payerActive, err := s.isActiveMember(st.GroupId, payerID)
	if err != nil {
		return Settlement{}, err
	}
	if !payerActive {
		return Settlement{}, ErrPayerNotMember
	}

	recipientActive, err := s.isActiveMember(st.GroupId, st.PaidTo)
	if err != nil {
		return Settlement{}, err
	}
	if !recipientActive {
		return Settlement{}, ErrRecipientNotMember
	}

	st.PaidBy = payerID
	return s.store.Create(st)
}

// GetById returns a settlement by id.
func (s *Service) GetById(id string) (Settlement, error) {
	return s.store.GetById(id)
}

// ListByGroup returns a group's settlements, most recent first.
func (s *Service) ListByGroup(groupID string) ([]Settlement, error) {
	return s.store.ListByGroup(groupID)
}

// Delete removes a settlement. Only the payer can delete it.
func (s *Service) Delete(id string, requesterID string) (Settlement, error) {
	existing, err := s.store.GetById(id)
	if err != nil {
		return Settlement{}, err
	}
	if existing.PaidBy != requesterID {
		return Settlement{}, ErrNotPayer
	}
	return s.store.Delete(id)
}
