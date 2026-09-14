package split

import (
	"errors"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
)

var (
	ErrNoMembers           = errors.New("group has no members")
	ErrSplitsRequired      = errors.New("splits are required for this split type")
	ErrDuplicateSplitUsers = errors.New("duplicate user in splits")
	ErrExactSumMismatch    = errors.New("sum of exact splits must equal total amount")
	ErrPercentSum          = errors.New("percentages must sum to 100")
	ErrSharesInvalid       = errors.New("total shares must be greater than 0")
	ErrInvalidType         = errors.New("unknown split type")
)

// SplitInput is one user-provided split entry. The field relevant to
// the split type must be set.
type SplitInput struct {
	UserID     string
	Amount     *float64
	Percentage *float64
	Shares     *int
}

// Computed is a resolved split with the final amount for a user.
type Computed struct {
	UserID     string
	Amount     float64
	Percentage *float64
	Shares     *int
}

// Request describes a split computation.
type Request struct {
	Type      Type
	Total     float64
	MemberIDs []string // active group members, used by EQUAL
	Splits    []SplitInput
}

// Compute resolves a Request into per-user amounts, rounded to two
// decimals. Any rounding remainder is assigned to the first entry.
func Compute(r Request) ([]Computed, error) {
	switch r.Type {
	case Equal:
		return computeEqual(r)
	case Exact:
		return computeExact(r)
	case Percentage:
		return computePercentage(r)
	case Shares:
		return computeShares(r)
	default:
		return nil, ErrInvalidType
	}
}

func computeEqual(r Request) ([]Computed, error) {
	if len(r.MemberIDs) == 0 {
		return nil, ErrNoMembers
	}

	each := util.Round2(r.Total / float64(len(r.MemberIDs)))
	results := make([]Computed, 0, len(r.MemberIDs))
	sum := 0.0
	for _, uid := range r.MemberIDs {
		results = append(results, Computed{UserID: uid, Amount: each})
		sum = util.Round2(sum + each)
	}
	results[0].Amount = util.Round2(results[0].Amount + util.Round2(r.Total-sum))
	return results, nil
}

func computeExact(r Request) ([]Computed, error) {
	if len(r.Splits) == 0 {
		return nil, ErrSplitsRequired
	}

	seen := make(map[string]bool, len(r.Splits))
	results := make([]Computed, 0, len(r.Splits))
	sum := 0.0
	for _, s := range r.Splits {
		if s.UserID == "" || s.Amount == nil {
			return nil, ErrSplitsRequired
		}
		if seen[s.UserID] {
			return nil, ErrDuplicateSplitUsers
		}
		seen[s.UserID] = true

		amount := util.Round2(*s.Amount)
		results = append(results, Computed{UserID: s.UserID, Amount: amount})
		sum = util.Round2(sum + amount)
	}
	if sum != util.Round2(r.Total) {
		return nil, ErrExactSumMismatch
	}
	return results, nil
}

func computePercentage(r Request) ([]Computed, error) {
	if len(r.Splits) == 0 {
		return nil, ErrSplitsRequired
	}

	seen := make(map[string]bool, len(r.Splits))
	results := make([]Computed, 0, len(r.Splits))
	pctSum := 0.0
	for _, s := range r.Splits {
		if s.UserID == "" || s.Percentage == nil {
			return nil, ErrSplitsRequired
		}
		if seen[s.UserID] {
			return nil, ErrDuplicateSplitUsers
		}
		seen[s.UserID] = true

		pct := *s.Percentage
		pctSum = util.Round2(pctSum + pct)
		amount := util.Round2(r.Total * pct / 100)
		results = append(results, Computed{UserID: s.UserID, Amount: amount, Percentage: &pct})
	}
	if pctSum != 100 {
		return nil, ErrPercentSum
	}

	sum := 0.0
	for _, c := range results {
		sum = util.Round2(sum + c.Amount)
	}
	results[0].Amount = util.Round2(results[0].Amount + util.Round2(r.Total-sum))
	return results, nil
}

func computeShares(r Request) ([]Computed, error) {
	if len(r.Splits) == 0 {
		return nil, ErrSplitsRequired
	}

	seen := make(map[string]bool, len(r.Splits))
	results := make([]Computed, 0, len(r.Splits))
	totalShares := 0
	for _, s := range r.Splits {
		if s.UserID == "" || s.Shares == nil || *s.Shares <= 0 {
			return nil, ErrSharesInvalid
		}
		if seen[s.UserID] {
			return nil, ErrDuplicateSplitUsers
		}
		seen[s.UserID] = true

		totalShares += *s.Shares
		shares := *s.Shares
		results = append(results, Computed{UserID: s.UserID, Shares: &shares})
	}

	for i := range results {
		results[i].Amount = util.Round2(r.Total * float64(*results[i].Shares) / float64(totalShares))
	}
	if totalShares <= 0 {
		return nil, ErrSharesInvalid
	}

	sum := 0.0
	for _, c := range results {
		sum = util.Round2(sum + c.Amount)
	}
	results[0].Amount = util.Round2(results[0].Amount + util.Round2(r.Total-sum))
	return results, nil
}
