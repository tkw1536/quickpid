package pid

//spellchecker:words errors
//spellchecker:words SplitMix Vigna unimi Knuth Steele Flood splittable congruential
import (
	"errors"
	"fmt"
	"math/bits"
	"strings"
)

// Layout is an alphabet + template used to encode dense indexes as IDs.
//
// In Pattern, each '*' is a variable slot filled from [CharacterSet]; any other rune is a literal.
// Zero '*' is allowed: [Layout.Space] is then 1, and [Layout.EncodeIndex] with index 0
// returns Pattern unchanged.
type Layout struct {
	Characters CharacterSet
	Pattern    string
}

var (
	errIndexOutOfRange = errors.New("index out of range for layout space")
	errSpaceOverflow   = errors.New("layout space does not fit in uint64")
)

// Seeds for deriving the per-space linear congruential generator (LCG) parameters (m, c).
//
// An LCG is the recurrence x -> (a*x + c) mod m. Here a single LCG step permutes
// indexes; it is not used as a streaming PRNG.
//
// Both values come from Sebastiano Vigna's SplitMix64 reference ([splitmix64.c];
// also Steele/Lea/Flood, [Fast splittable pseudorandom number generators]).
// This package does not implement SplitMix64; it only reuses the published constants so the
// bijection is stable and has a clear provenance.
//
// [splitmix64.c]: http://xorshift.di.unimi.it/splitmix64.c
// [Fast splittable pseudorandom number generators]: https://dl.acm.org/doi/10.1145/2660193.2660195
const (
	// scrambleMultiplierSeed is SplitMix64's first avalanche multiplier.
	scrambleMultiplierSeed = uint64(0xbf58476d1ce4e5b9)

	// scrambleAdditiveSeed is SplitMix64's fixed state increment.
	// Numerically it equals floor(2^64 / phi) with phi = (1 + sqrt(5)) / 2
	// (golden-ratio multiplicative hashing). The 32-bit form floor(2^32 / phi) = 0x9e3779b9
	// is the constant commonly cited in Knuth.
	scrambleAdditiveSeed = uint64(0x9e3779b97f4a7c15)
)

// Space is the number of distinct IDs this layout can represent (len(alphabet)^count('*')).
// With zero stars, Space is 1.
//
// Returns an error if the character set is invalid or the space does not fit in uint64.
func (l Layout) Space() (uint64, error) {
	alphabet, stars, err := l.parts()
	if err != nil {
		return 0, err
	}
	return spaceSize(uint64(len(alphabet)), stars)
}

// EncodeIndex maps a dense index in [0, n) to an ID that still looks pseudo-random,
// so consecutive counters do not produce consecutive-looking strings.
// n is the value returned by [Layout.Space].
// The mapping is bijective on the layout space so every ID remains reachable.
//
// With zero stars, only index 0 is valid and the result is Pattern as-is.
func (l Layout) EncodeIndex(index uint64) (string, error) {
	alphabet, stars, err := l.parts()
	if err != nil {
		return "", err
	}
	space, err := spaceSize(uint64(len(alphabet)), stars)
	if err != nil {
		return "", err
	}
	if index >= space {
		return "", fmt.Errorf("%w: %d >= %d", errIndexOutOfRange, index, space)
	}
	return encodeRaw(l.Pattern, alphabet, stars, scramble(index, space)), nil
}

func (l Layout) parts() (alphabet []rune, stars int, err error) {
	if err := l.Characters.Validate(); err != nil {
		return nil, 0, fmt.Errorf("%w: %w", errInvalidCharacterSet, err)
	}
	s, ok := l.Characters.Alphabet()
	if !ok {
		return nil, 0, errInvalidCharacterSet
	}
	alphabet = []rune(s)
	for _, c := range l.Pattern {
		if c == '*' {
			stars++
		}
	}
	return alphabet, stars, nil
}

func spaceSize(base uint64, stars int) (uint64, error) {
	if stars == 0 {
		return 1, nil
	}
	if base == 0 {
		return 0, errEmptyCharacterSet
	}
	space := uint64(1)
	for range stars {
		hi, lo := bits.Mul64(space, base)
		if hi != 0 {
			return 0, errSpaceOverflow
		}
		space = lo
	}
	return space, nil
}

// scramble permutes {0 .. space-1} so nearby indexes land far apart in the ID space.
// It applies one LCG step (see [scrambleAdditiveSeed], [scrambleMultiplierSeed]):
// (i*m + c) mod space. Choosing m with gcd(m, space) == 1 makes that step a bijection,
// which fill-based allocation needs (no permanently skipped IDs).
func scramble(i, space uint64) uint64 {
	if space <= 1 {
		return 0
	}
	m := multiplierFor(space)
	c := scrambleAdditiveSeed % space
	if c%2 == 0 {
		c = (c + 1) % space
	}
	return (mulMod(i, m, space) + c) % space
}

func multiplierFor(space uint64) uint64 {
	m := scrambleMultiplierSeed % space
	if m == 0 {
		m = 1
	}
	for gcd(m, space) != 1 {
		m++
		if m == space {
			m = 1
		}
	}
	return m
}

func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// mulMod computes (a*b) mod m without overflowing a uint64 intermediate.
func mulMod(a, b, m uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	_, rem := bits.Div64(hi, lo, m)
	return rem
}

// encodeRaw places value into the layout template using the alphabet as digits.
func encodeRaw(pattern string, alphabet []rune, stars int, value uint64) string {
	if stars == 0 {
		return pattern
	}
	base := uint64(len(alphabet))
	digits := make([]rune, stars)
	n := value
	for i := stars - 1; i >= 0; i-- {
		digits[i] = alphabet[n%base]
		n /= base
	}

	var b strings.Builder
	b.Grow(len(pattern))
	di := 0
	for _, c := range pattern {
		if c == '*' {
			b.WriteRune(digits[di])
			di++
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}
