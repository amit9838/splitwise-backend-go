package split

// Type represents how an expense is split among participants.
type Type string

const (
	Equal      Type = "EQUAL"
	Exact      Type = "EXACT"
	Percentage Type = "PERCENTAGE"
	Shares     Type = "SHARES"
)

// Valid reports whether t is a supported split type.
func (t Type) Valid() bool {
	switch t {
	case Equal, Exact, Percentage, Shares:
		return true
	default:
		return false
	}
}
