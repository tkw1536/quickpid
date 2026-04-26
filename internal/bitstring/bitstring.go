package bitstring

import (
	"io"
	"math/big"
)

// NewReader returns a reader yielding all binary strings in increasing length order:
//
//	0, 1, 00, 01, 10, 11, 000, 001, ...
//
// Bits are packed MSB-first into bytes when read.
//
// This read is intended to produce a predictable sequence of bytes for use in tests.
func NewReader() io.Reader {
	return &reader{}
}

type reader struct {
	n     int     // current length of binary string
	i     big.Int // current index within length n
	limit big.Int // total number of strings within length n
	pos   int     // position within current string
}

var one = big.NewInt(1)

func (r *reader) nextN() {
	// Advance to the next string length and reset iteration state.
	r.n++
	r.pos = 0
	r.i.SetUint64(0)

	// limit = 2^n
	r.limit.SetUint64(1)
	r.limit.Lsh(&r.limit, uint(r.n))
}

func (r *reader) nextBit() byte {
	if r.n == 0 {
		r.nextN() // starts at n=1
	}

	// MSB-first within the n-bit string.
	shift := r.n - 1 - r.pos
	bit := byte(r.i.Bit(shift))

	r.pos++
	if r.pos >= r.n {
		r.pos = 0
		r.i.Add(&r.i, one)
		if r.i.Cmp(&r.limit) >= 0 {
			r.nextN()
		}
	}
	return bit
}

func (r *reader) Read(p []byte) (int, error) {
	for j := range p {
		var b byte
		for range 8 {
			b = (b << 1) | r.nextBit()
		}
		p[j] = b
	}
	return len(p), nil
}
