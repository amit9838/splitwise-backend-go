package balance

import (
	"errors"

	"github.com/amit9838/splitwise-backend-go/internal/expense"
	"github.com/amit9838/splitwise-backend-go/internal/group"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
	"github.com/amit9838/splitwise-backend-go/internal/settlement"
	"github.com/amit9838/splitwise-backend-go/internal/user"
)

var ErrGroupNotFound = errors.New("group not found")

// ExpenseLookup provides expense and split data.
type ExpenseLookup interface {
	ListByGroup(groupID string) ([]expense.Expense, error)
	ListSplitsByGroup(groupID string) ([]expense.Split, error)
}

// SettlementLookup provides settlement data.
type SettlementLookup interface {
	ListByGroup(groupID string) ([]settlement.Settlement, error)
}

// GroupLookup provides group data.
type GroupLookup interface {
	GetById(id string) (group.Group, error)
}

// MemberLookup provides membership data.
type MemberLookup interface {
	ListActiveByGroup(groupID string) ([]group.Member, error)
	ListGroupIDsByUser(userID string) ([]string, error)
}

// UserLookup provides user data.
type UserLookup interface {
	GetById(id string) (user.User, error)
}

// Balance is one member's net position in a group.
type Balance struct {
	UserId     string  `json:"user_id"`
	Email      string  `json:"email"`
	FullName   string  `json:"full_name"`
	NetBalance float64 `json:"net_balance"`
	Status     string  `json:"status"` // owed, owes, settled
}

// GroupBalances is the full balance sheet of a group.
type GroupBalances struct {
	GroupId               string     `json:"group_id"`
	GroupName             string     `json:"group_name"`
	Balances              []Balance  `json:"balances"`
	SimplifiedSettlements []Transfer `json:"simplified_settlements"`
	TotalSettlements      int        `json:"total_settlements"`
}

// MyGroupBalance is the current user's position in one group.
type MyGroupBalance struct {
	GroupId       string     `json:"group_id"`
	GroupName     string     `json:"group_name"`
	MyNetBalance  float64    `json:"my_net_balance"`
	MySettlements []Transfer `json:"my_settlements"`
}

// MyBalances is the current user's position across all groups.
type MyBalances struct {
	UserId        string           `json:"user_id"`
	TotalOwedToMe float64          `json:"total_owed_to_me"`
	TotalIOwe     float64          `json:"total_i_owe"`
	NetBalance    float64          `json:"net_balance"`
	Groups        []MyGroupBalance `json:"groups"`
}

// Service computes balances from expenses, splits and settlements.
type Service struct {
	expenses    ExpenseLookup
	settlements SettlementLookup
	groups      GroupLookup
	members     MemberLookup
	users       UserLookup
}

func NewService(expenses ExpenseLookup, settlements SettlementLookup, groups GroupLookup, members MemberLookup, users UserLookup) *Service {
	return &Service{expenses: expenses, settlements: settlements, groups: groups, members: members, users: users}
}

// computeNet returns each user's net balance in a group: what they paid
// (expenses + settlements) minus what they owe (splits + settlements received).
func (s *Service) computeNet(groupID string) (map[string]float64, error) {
	expenses, err := s.expenses.ListByGroup(groupID)
	if err != nil {
		return nil, err
	}
	splits, err := s.expenses.ListSplitsByGroup(groupID)
	if err != nil {
		return nil, err
	}
	settlements, err := s.settlements.ListByGroup(groupID)
	if err != nil {
		return nil, err
	}

	net := make(map[string]float64)
	for _, e := range expenses {
		net[e.PaidBy] = util.Round2(net[e.PaidBy] + e.Amount)
	}
	for _, sp := range splits {
		net[sp.UserId] = util.Round2(net[sp.UserId] - sp.Amount)
	}
	for _, st := range settlements {
		net[st.PaidBy] = util.Round2(net[st.PaidBy] + st.Amount)
		net[st.PaidTo] = util.Round2(net[st.PaidTo] - st.Amount)
	}
	return net, nil
}

func balanceStatus(net float64) string {
	switch {
	case net > 0.004:
		return "owed"
	case net < -0.004:
		return "owes"
	default:
		return "settled"
	}
}

// GroupBalances returns the balances and simplified settlements of a group.
func (s *Service) GroupBalances(groupID string) (*GroupBalances, error) {
	g, err := s.groups.GetById(groupID)
	if err != nil {
		return nil, ErrGroupNotFound
	}

	members, err := s.members.ListActiveByGroup(groupID)
	if err != nil {
		return nil, err
	}

	net, err := s.computeNet(groupID)
	if err != nil {
		return nil, err
	}

	balances := make([]Balance, 0, len(members))
	for _, m := range members {
		u, err := s.users.GetById(m.UserId)
		if err != nil {
			return nil, err
		}
		nb := util.Round2(net[m.UserId])
		balances = append(balances, Balance{
			UserId:     m.UserId,
			Email:      u.Email,
			FullName:   u.FullName,
			NetBalance: nb,
			Status:     balanceStatus(nb),
		})
	}

	transfers := Simplify(net)
	return &GroupBalances{
		GroupId:               groupID,
		GroupName:             g.Name,
		Balances:              balances,
		SimplifiedSettlements: transfers,
		TotalSettlements:      len(transfers),
	}, nil
}

// MyBalances returns the current user's position across their groups.
func (s *Service) MyBalances(userID string) (*MyBalances, error) {
	groupIDs, err := s.members.ListGroupIDsByUser(userID)
	if err != nil {
		return nil, err
	}

	out := MyBalances{UserId: userID, Groups: make([]MyGroupBalance, 0, len(groupIDs))}
	for _, gid := range groupIDs {
		g, err := s.groups.GetById(gid)
		if err != nil {
			if errors.Is(err, response.ErrNotFound) {
				continue
			}
			return nil, err
		}

		net, err := s.computeNet(gid)
		if err != nil {
			return nil, err
		}
		myNet := util.Round2(net[userID])

		mySettlements := make([]Transfer, 0)
		for _, t := range Simplify(net) {
			if t.FromUserId == userID || t.ToUserId == userID {
				mySettlements = append(mySettlements, t)
			}
		}

		if myNet > 0 {
			out.TotalOwedToMe = util.Round2(out.TotalOwedToMe + myNet)
		} else if myNet < 0 {
			out.TotalIOwe = util.Round2(out.TotalIOwe - myNet)
		}

		out.Groups = append(out.Groups, MyGroupBalance{
			GroupId:       gid,
			GroupName:     g.Name,
			MyNetBalance:  myNet,
			MySettlements: mySettlements,
		})
	}
	out.NetBalance = util.Round2(out.TotalOwedToMe - out.TotalIOwe)
	return &out, nil
}
