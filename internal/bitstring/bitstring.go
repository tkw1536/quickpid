package bitstring

import "io"

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
	// current binary string length
	n int
	// current index within length n
	i uint64
	// position within current string [0..n]
	pos int
}

func (r *reader) nextBit() byte {
	if r.n == 0 {
		r.n = 1
		r.i = 0
		r.pos = 0
	}

	// MSB-first within the n-bit string.
	shift := r.n - 1 - r.pos
	bit := byte((r.i >> shift) & 1)

	r.pos++
	if r.pos >= r.n {
		r.pos = 0
		r.i++
		if r.i >= (uint64(1) << r.n) {
			r.n++
			r.i = 0
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
