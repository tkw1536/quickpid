package server

import (
	"crypto/rand"
	"encoding/base64"
)

const randomPIDEntropyBytes = 16

// DefaultPIDMaxAttempts is the suggested maximum tries (including the first) when allocating a PID
// after a collision in the backing store.
const DefaultPIDMaxAttempts = 8

// RandomAlphanumericPID returns a random alphanumeric string for use as a PID.
// It does not use any client-supplied request data.
func RandomAlphanumericPID() (string, error) {
	buf := make([]byte, randomPIDEntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(buf)
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
	}
	if len(out) >= 12 {
		return string(out[:12]), nil
	}
	const hexd = "0123456789abcdef"
	b := make([]byte, 0, 16)
	for _, x := range buf {
		b = append(b, hexd[x>>4], hexd[x&0xf])
	}
	return string(b[:16]), nil
}

const alphanumeric62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// SixCharPIDWithDashes returns a PID of the form "abc-def" (three random alphanumeric characters,
// a hyphen, then three more). It does not use any client-supplied request data.
func SixCharPIDWithDashes() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	const n = len(alphanumeric62)
	out := make([]byte, 7)
	for i := 0; i < 3; i++ {
		out[i] = alphanumeric62[buf[i]%byte(n)]
	}
	out[3] = '-'
	for i := 0; i < 3; i++ {
		out[4+i] = alphanumeric62[buf[3+i]%byte(n)]
	}
	return string(out), nil
}
