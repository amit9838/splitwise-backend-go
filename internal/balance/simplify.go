package balance

import (
	"math"
	"sort"

	"github.com/amit9838/splitwise-backend-go/internal/pkg/util"
)

// Transfer is one suggested payment that settles group debt.
type Transfer struct {
	FromUserId string  `json:"from_user_id"`
	ToUserId   string  `json:"to_user_id"`
	Amount     float64 `json:"amount"`
}

// Simplify reduces a set of per-user net balances to the minimal number
// of transfers (greedy max-debtor / max-creditor matching).
func Simplify(net map[string]float64) []Transfer {
	type party struct {
		id  string
		amt float64
	}

	var debtors, creditors []party
	for id, amt := range net {
		amt = util.Round2(amt)
		switch {
		case amt < -0.004:
			debtors = append(debtors, party{id, -amt})
		case amt > 0.004:
			creditors = append(creditors, party{id, amt})
		}
	}

	sort.Slice(debtors, func(i, j int) bool {
		if debtors[i].amt != debtors[j].amt {
			return debtors[i].amt > debtors[j].amt
		}
		return debtors[i].id < debtors[j].id
	})
	sort.Slice(creditors, func(i, j int) bool {
		if creditors[i].amt != creditors[j].amt {
			return creditors[i].amt > creditors[j].amt
		}
		return creditors[i].id < creditors[j].id
	})

	transfers := make([]Transfer, 0)
	di, ci := 0, 0
	for di < len(debtors) && ci < len(creditors) {
		d, c := &debtors[di], &creditors[ci]
		amt := util.Round2(math.Min(d.amt, c.amt))
		if amt > 0.004 {
			transfers = append(transfers, Transfer{FromUserId: d.id, ToUserId: c.id, Amount: amt})
		}
		d.amt = util.Round2(d.amt - amt)
		c.amt = util.Round2(c.amt - amt)
		if d.amt <= 0.004 {
			di++
		}
		if c.amt <= 0.004 {
			ci++
		}
	}
	return transfers
}
