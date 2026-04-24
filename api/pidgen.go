package api

import (
	"encoding/hex"
	"fmt"
	"io"
)

// PIDGenerator selects a PID generation algorithm.
type PIDGenerator string

const (
	PIDGeneratorLegacy    PIDGenerator = "legacy"
	PIDGeneratorReadable6 PIDGenerator = "readable6"
	PIDGeneratorReadable9 PIDGenerator = "readable9"
	PIDGeneratorRandom64  PIDGenerator = "random64"
	PIDGeneratorUUID4     PIDGenerator = "uuid4"
)

type pidGeneratorFunc func(io.Reader) (string, error)

var pidGenerators = map[PIDGenerator]pidGeneratorFunc{
	PIDGeneratorLegacy:    pidLegacy,
	PIDGeneratorReadable6: pidReadable6,
	PIDGeneratorReadable9: pidReadable9,
	PIDGeneratorRandom64:  pidRandom64,
	PIDGeneratorUUID4:     pidUUID4,
}

// Generator returns a PID generator function and whether it exists.
func Generator(gen PIDGenerator) (func(io.Reader) (string, error), bool) {
	fn, ok := pidGenerators[gen]
	return fn, ok
}

// GeneratePID generates a new PID according to gen, using rand for randomness.
func GeneratePID(gen PIDGenerator, rand io.Reader) (string, error) {
	fn := pidGenerators[gen]
	if fn == nil {
		return "", ErrInvalidPIDGenerator
	}
	return fn(rand)
}

const alphanumeric36 = "0123456789abcdefghijklmnopqrstuvwxyz"
const crockford32NoU = "0123456789abcdefghjkmnpqrstvwxyz"

func readFull(rand io.Reader, buf []byte) error {
	_, err := io.ReadFull(rand, buf)
	return err
}

func pidLegacy(rand io.Reader) (string, error) {
	buf := make([]byte, 6)
	if err := readFull(rand, buf); err != nil {
		return "", err
	}
	const n = byte(len(alphanumeric36))
	out := make([]byte, 7)
	for i := range 3 {
		out[i] = alphanumeric36[buf[i]%n]
	}
	out[3] = '-'
	for i := range 3 {
		out[4+i] = alphanumeric36[buf[3+i]%n]
	}
	return string(out), nil
}

func pidReadable6(rand io.Reader) (string, error) {
	buf := make([]byte, 6)
	if err := readFull(rand, buf); err != nil {
		return "", err
	}
	const n = byte(len(crockford32NoU))
	out := make([]byte, 7)
	for i := range 3 {
		out[i] = crockford32NoU[buf[i]%n]
	}
	out[3] = '-'
	for i := range 3 {
		out[4+i] = crockford32NoU[buf[3+i]%n]
	}
	return string(out), nil
}

func pidReadable9(rand io.Reader) (string, error) {
	buf := make([]byte, 9)
	if err := readFull(rand, buf); err != nil {
		return "", err
	}
	const n = byte(len(crockford32NoU))
	out := make([]byte, 11)
	for i := range 3 {
		out[i] = crockford32NoU[buf[i]%n]
	}
	out[3] = '-'
	for i := range 3 {
		out[4+i] = crockford32NoU[buf[3+i]%n]
	}
	out[7] = '-'
	for i := range 3 {
		out[8+i] = crockford32NoU[buf[6+i]%n]
	}
	return string(out), nil
}

func pidRandom64(rand io.Reader) (string, error) {
	buf := make([]byte, 64)
	if err := readFull(rand, buf); err != nil {
		return "", err
	}
	const n = byte(len(alphanumeric36))
	out := make([]byte, 64)
	for i := range out {
		out[i] = alphanumeric36[buf[i]%n]
	}
	return string(out), nil
}

func pidUUID4(rand io.Reader) (string, error) {
	var b [16]byte
	if err := readFull(rand, b[:]); err != nil {
		return "", err
	}

	// RFC 4122 variant + version 4.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var hex32 [32]byte
	hex.Encode(hex32[:], b[:])
	// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex32[0:8], hex32[8:12], hex32[12:16], hex32[16:20], hex32[20:32]), nil
}
