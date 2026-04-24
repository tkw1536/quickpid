package api

import (
	"testing"
	"io"
)

type fakeRandReader struct {
	n   int
	i   uint64
	pos int
}

var _ io.Reader = (*fakeRandReader)(nil)

func (r *fakeRandReader) nextBit() byte {
	if r.n == 0 {
		r.n = 1
		r.i = 0
		r.pos = 0
	}
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

func (r *fakeRandReader) Read(p []byte) (int, error) {
	for j := range p {
		var b byte
		for range 8 {
			b = (b << 1) | r.nextBit()
		}
		p[j] = b
	}
	return len(p), nil
}

func checkPID(t *testing.T, generator PIDGenerator, expect []string) {
	t.Helper()

	r := &fakeRandReader{}
	for i, want := range expect {
		got, err := GeneratePID(generator, r)
		if err != nil {
			t.Fatalf("GeneratePID(%q) call %d: %v", generator, i, err)
		}
		if got != want {
			t.Fatalf("GeneratePID(%q) call %d: got %q want %q", generator, i, got, want)
		}
	}
}

func TestGeneratePID(t *testing.T) {
	testCases := map[PIDGenerator][]string{
		PIDGeneratorLegacy: {
			"yd6-lc0",
			"tha-yrf",
			"chc-pds",
		},
		PIDGeneratorReadable6: {
			"61e-x08",
			"hs2-akv",
			"0hc-5hg",
		},
		PIDGeneratorReadable9: {
			"61e-x08-hs2",
			"akv-0hc-5hg",
			"ndd-k1s-enn",
		},
		PIDGeneratorRandom64: {
			"yd6lc0thayrfchcpds59x79p651pd3dvc4wgkpk0iogbsx0wdt0tm49f8q4c6xgm",
		},
		PIDGeneratorUUID4: {
			"46c14e5d-c048-4159-a26a-f37bc0110c85",
			"31d0952d-8d73-4119-8e95-b5f19d6f9df7",
			"c00420c4-1461-4824-a2cc-34e3d04524d4",
		},
	}

	for gen, expect := range testCases {
		t.Run(string(gen), func(t *testing.T) {
			checkPID(t, gen, expect)
		})
	}

	t.Run("invalid", func(t *testing.T) {
		r := &fakeRandReader{}
		_, err := GeneratePID(PIDGenerator("nope"), r)
		if err != ErrInvalidPIDGenerator {
			t.Fatalf("GeneratePID(invalid): got err %v want %v", err, ErrInvalidPIDGenerator)
		}
	})
}

