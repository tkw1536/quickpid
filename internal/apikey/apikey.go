// Package apikey generates API keys and produces stored prefix+digest representations
// suitable for authentication backends.
//
// Keys are random alphanumeric strings. When persisted, the leading prefix is kept
// in plaintext to speed up lookup; the digest covers only the remaining secret portion.
//
//spellchecker:words apikey
package apikey

//spellchecker:words crypto subtle errors github bicpid
import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"

	"github.com/tkw1536/bicpid/pid"
)

// HashFunc hashes the secret (non-prefix) portion of a key.
//
// A nil HashFunc uses SHA-256.
type HashFunc func(key []byte) []byte

// Hash hashes the key using this function.
func (hash HashFunc) Hash(key []byte) []byte {
	if hash == nil {
		sum := sha256.Sum256(key)
		return sum[:]
	}

	return hash(key)
}

// Stored is what gets persisted for an API key.
//
// Prefix is stored in plaintext to support indexed lookup. Digest is a hash of the
// secret portion after the prefix; the full secret is never stored.
type Stored struct {
	Prefix string
	Digest []byte
}

// Format defines the shape of generated API keys and how they are hashed.
type Format struct {
	Charset pid.CharacterSet // the character set to use for key generation

	Length    int // Length is the total number of alphanumeric characters in a key.
	PrefixLen int // PrefixLen is how many leading characters are stored in plaintext for lookup.

	Hasher HashFunc
}

// Default is a Format with recommended production defaults.
//
// It generates 32-character lowercase alphanumeric keys, stores an 8-character
// lookup prefix, and hashes the secret suffix with SHA-256.
var Default = Format{
	Charset:   pid.Full,
	Length:    32,
	PrefixLen: 8,
}

var (
	errInvalidPrefixLen = errors.New("prefix length must be positive")
	errInvalidLength    = errors.New("key length must exceed prefix length")
	errInvalidCharset   = errors.New("charset must be valid")
	errInvalidKeyLength = errors.New("key length does not match format")
	errInvalidKeyChar   = errors.New("key contains character outside alphabet")
)

// Validate checks that this format is usable for key generation and verification.
func (f Format) Validate() error {
	if f.PrefixLen <= 0 {
		return errInvalidPrefixLen
	}
	if f.Length <= f.PrefixLen {
		return errInvalidLength
	}
	if err := f.Charset.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidCharset, err)
	}
	return nil
}

// Generate creates a new random API key using rand for randomness.
//
// The returned key has exactly Length characters, each drawn uniformly from Alphabet.
func (f Format) Generate(rand io.Reader) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}

	runes := make([]rune, f.Length)
	for i := range runes {
		r, err := f.Charset.Pick(rand)
		if err != nil {
			return "", fmt.Errorf("Charset.Pick: %w", err)
		}
		runes[i] = r
	}
	return string(runes), nil
}

// Prefix returns the lookup prefix of key: the first PrefixLen characters.
//
// The prefix is stored in plaintext alongside the digest to speed up lookup.
func (f Format) Prefix(key string) (string, error) {
	prefix, _, err := f.splitKey(key)
	return prefix, err
}

// Hash returns the stored prefix and digest for key.
//
// The digest is computed over the secret portion after the prefix. The prefix is
// the first PrefixLen characters of key, stored separately to support indexed lookup.
func (f Format) Hash(key string) (Stored, error) {
	prefix, secret, err := f.splitKey(key)
	if err != nil {
		return Stored{}, err
	}

	digest := f.Hasher.Hash([]byte(secret))
	return Stored{Prefix: prefix, Digest: digest}, nil
}

// Verify checks that key matches stored.
//
// It first compares the lookup prefix, then verifies the digest of the secret
// suffix using constant-time comparison. It returns false on any format error or mismatch.
func (f Format) Verify(key string, stored Stored) bool {
	prefix, secret, err := f.splitKey(key)
	if err != nil {
		return false
	}
	if prefix != stored.Prefix {
		return false
	}

	digest := f.Hasher.Hash([]byte(secret))
	return digestMatches(stored.Digest, digest)
}

func (f Format) splitKey(key string) (prefix, secret string, err error) {
	if err := f.Validate(); err != nil {
		return "", "", err
	}
	if err := validateKey(f, key); err != nil {
		return "", "", err
	}

	runes := []rune(key)
	prefix = string(runes[:f.PrefixLen])
	secret = string(runes[f.PrefixLen:])
	return prefix, secret, nil
}

func validateKey(f Format, key string) error {
	runes := []rune(key)
	if len(runes) != f.Length {
		return errInvalidKeyLength
	}

	chars, ok := f.Charset.Alphabet()
	if !ok {
		return errInvalidCharset
	}

	allowed := make(map[rune]struct{}, len(chars))
	for _, r := range chars {
		allowed[r] = struct{}{}
	}

	for _, r := range runes {
		if _, ok := allowed[r]; !ok {
			return errInvalidKeyChar
		}
	}
	return nil
}

func digestMatches(stored, computed []byte) bool {
	return subtle.ConstantTimeCompare(stored, computed) == 1
}
