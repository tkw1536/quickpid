package api

import (
	"io"
)

// PIDCharacters selects a character set for PID generation.
type PIDCharacters string

const (
	PIDCharactersFull     PIDCharacters = "full"
	PIDCharactersReadable PIDCharacters = "readable"
	PIDCharactersHex      PIDCharacters = "hex"
	PIDCharactersDecimal  PIDCharacters = "decimal"
)

// PIDFormat configures PID generation for a namespace.
type PIDFormat struct {
	Pattern    string        `json:"pattern"`
	Characters PIDCharacters `json:"characters"`
}

const alphanumeric36 = "0123456789abcdefghijklmnopqrstuvwxyz"
const crockford32NoU = "0123456789abcdefghjkmnpqrstvwxyz"
const lowercaseHex16 = "0123456789abcdef"
const decimal10 = "0123456789"

func readFull(rand io.Reader, buf []byte) error {
	_, err := io.ReadFull(rand, buf)
	return err
}

func pidAlphabet(chars PIDCharacters) (string, bool) {
	switch chars {
	case PIDCharactersFull:
		return alphanumeric36, true
	case PIDCharactersReadable:
		return crockford32NoU, true
	case PIDCharactersHex:
		return lowercaseHex16, true
	case PIDCharactersDecimal:
		return decimal10, true
	default:
		return "", false
	}
}

func isValidPIDPattern(pattern string) bool {
	if pattern == "" {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*', '-', '_':
			// ok
		default:
			return false
		}
	}
	return true
}

// ValidatePIDFormat validates format and returns ErrInvalidPIDFormat on failure.
func ValidatePIDFormat(format PIDFormat) error {
	if !isValidPIDPattern(format.Pattern) {
		return ErrInvalidPIDFormat
	}
	if _, ok := pidAlphabet(format.Characters); !ok {
		return ErrInvalidPIDFormat
	}
	for i := 0; i < len(format.Pattern); i++ {
		if format.Pattern[i] == '*' {
			return nil
		}
	}
	return ErrInvalidPIDFormat
}

// GeneratePID generates a new PID according to format, using rand for randomness.
//
// It replaces each '*' in format.Pattern with a random character from format.Characters,
// and leaves '-' and '_' unchanged.
func GeneratePID(format PIDFormat, rand io.Reader) (string, error) {
	if err := ValidatePIDFormat(format); err != nil {
		return "", err
	}
	alphabet, _ := pidAlphabet(format.Characters)

	starCount := 0
	for i := 0; i < len(format.Pattern); i++ {
		if format.Pattern[i] == '*' {
			starCount++
		}
	}

	buf := make([]byte, starCount)
	if err := readFull(rand, buf); err != nil {
		return "", err
	}

	out := make([]byte, len(format.Pattern))
	n := byte(len(alphabet))
	j := 0
	for i := 0; i < len(format.Pattern); i++ {
		switch format.Pattern[i] {
		case '*':
			out[i] = alphabet[buf[j]%n]
			j++
		case '-', '_':
			out[i] = format.Pattern[i]
		}
	}
	return string(out), nil
}
