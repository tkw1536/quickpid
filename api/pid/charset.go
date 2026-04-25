package pid

// A CharacterSet to be used in pid generation.
// Must be one of the pre-defined character sets.
type CharacterSet string

// Pre-defined character sets.
const (
	Full     CharacterSet = "full"
	Readable CharacterSet = "readable"
	Hex      CharacterSet = "hex"
	Decimal  CharacterSet = "decimal"
)

var alphabets = map[CharacterSet]string{
	Full:     "0123456789abcdefghijklmnopqrstuvwxyz",
	Readable: "0123456789abcdefghjkmnpqrstvwxyz",
	Hex:      "0123456789abcdef",
	Decimal:  "0123456789",
}

// Valid checks if this is a valid character set.
func (chars CharacterSet) Valid() bool {
	_, ok := alphabets[chars]
	return ok
}

// Alphabet returns the alphabet to be used for the given character set.
func (chars CharacterSet) Alphabet() (string, bool) {
	alphabet, ok := alphabets[chars]
	return alphabet, ok
}
