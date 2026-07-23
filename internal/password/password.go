// Package password hashes and verifies user passwords.
//
//spellchecker:words password
package password

//spellchecker:words golang crypto bcrypt
import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Hash hashes a plaintext password.
func Hash(plaintext string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt.GenerateFromPassword: %w", err)
	}
	return hash, nil
}

// Verify reports whether plaintext matches hash.
func Verify(plaintext string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(plaintext))
	return err == nil
}
