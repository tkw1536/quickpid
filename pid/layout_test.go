package pid_test

//spellchecker:words strings testing github quickpid
import (
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/pid"
)

func TestLayout_Space(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		layout  pid.Layout
		want    uint64
		wantErr string
	}{
		{
			name:   "zero_stars",
			layout: pid.Layout{Characters: pid.Decimal, Pattern: "literal"},
			want:   1,
		},
		{
			name:   "decimal_one_star",
			layout: pid.Layout{Characters: pid.Decimal, Pattern: "*"},
			want:   10,
		},
		{
			name:   "decimal_two_stars",
			layout: pid.Layout{Characters: pid.Decimal, Pattern: "**"},
			want:   100,
		},
		{
			name:   "hex_with_literals",
			layout: pid.Layout{Characters: pid.Hex, Pattern: "*-*"},
			want:   16 * 16,
		},
		{
			name:   "readable_three_chunks",
			layout: pid.Layout{Characters: pid.Readable, Pattern: "***-***"},
			want:   32 * 32 * 32 * 32 * 32 * 32,
		},
		{
			name:    "invalid_charset",
			layout:  pid.Layout{Characters: pid.CharacterSet("nope"), Pattern: "*"},
			wantErr: "invalid character set",
		},
		{
			name:    "space_overflow",
			layout:  pid.Layout{Characters: pid.Full, Pattern: strings.Repeat("*", 64)},
			wantErr: "layout space does not fit in uint64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.layout.Space()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Space() error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Space() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Space() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLayout_EncodeIndex_zeroStars(t *testing.T) {
	t.Parallel()

	layout := pid.Layout{Characters: pid.Decimal, Pattern: "fixed-id"}
	got, err := layout.EncodeIndex(0)
	if err != nil {
		t.Fatalf("EncodeIndex(0) error = %v", err)
	}
	if got != "fixed-id" {
		t.Fatalf("EncodeIndex(0) = %q, want %q", got, "fixed-id")
	}
	if _, err := layout.EncodeIndex(1); err == nil || !strings.Contains(err.Error(), "index out of range") {
		t.Fatalf("EncodeIndex(1) error = %v, want index out of range", err)
	}
}

func TestLayout_EncodeIndex_preservesLiterals(t *testing.T) {
	t.Parallel()

	layout := pid.Layout{Characters: pid.Decimal, Pattern: "***-***"}
	space, err := layout.Space()
	if err != nil {
		t.Fatalf("Space() error = %v", err)
	}
	for i := range min(space, 50) {
		got, err := layout.EncodeIndex(i)
		if err != nil {
			t.Fatalf("EncodeIndex(%d) error = %v", i, err)
		}
		if len(got) != 7 || got[3] != '-' {
			t.Fatalf("EncodeIndex(%d) = %q, want form ***-***", i, got)
		}
		for j, c := range got {
			if j == 3 {
				continue
			}
			if c < '0' || c > '9' {
				t.Fatalf("EncodeIndex(%d) = %q, non-decimal at %d", i, got, j)
			}
		}
	}
}

func TestLayout_EncodeIndex_bijection(t *testing.T) {
	t.Parallel()

	layout := pid.Layout{Characters: pid.Decimal, Pattern: "**"}
	space, err := layout.Space()
	if err != nil {
		t.Fatalf("Space() error = %v", err)
	}
	if space != 100 {
		t.Fatalf("Space() = %d, want 100", space)
	}

	seen := make(map[string]uint64, space)
	for i := range space {
		got, err := layout.EncodeIndex(i)
		if err != nil {
			t.Fatalf("EncodeIndex(%d) error = %v", i, err)
		}
		if prev, ok := seen[got]; ok {
			t.Fatalf("EncodeIndex(%d) = %q collides with EncodeIndex(%d)", i, got, prev)
		}
		seen[got] = i
	}
	if uint64(len(seen)) != space {
		t.Fatalf("unique encodings = %d, want %d", len(seen), space)
	}
}

func TestLayout_EncodeIndex_notSequential(t *testing.T) {
	t.Parallel()

	layout := pid.Layout{Characters: pid.Decimal, Pattern: "**"}
	// Naive sequential encodings for indexes 0..2 would be "00","01","02".
	naive := []string{"00", "01", "02"}
	matches := 0
	for i, want := range naive {
		got, err := layout.EncodeIndex(uint64(i))
		if err != nil {
			t.Fatalf("EncodeIndex(%d) error = %v", i, err)
		}
		if got == want {
			matches++
		}
	}
	if matches == len(naive) {
		t.Fatalf("EncodeIndex matched naive sequential encodings for 0..%d", len(naive)-1)
	}
}

func TestLayout_EncodeIndex_outOfRange(t *testing.T) {
	t.Parallel()

	layout := pid.Layout{Characters: pid.Hex, Pattern: "*"}
	if _, err := layout.EncodeIndex(16); err == nil || !strings.Contains(err.Error(), "index out of range") {
		t.Fatalf("EncodeIndex(16) error = %v, want index out of range", err)
	}
}

func TestLayout_EncodeIndex_invalidCharset(t *testing.T) {
	t.Parallel()

	layout := pid.Layout{Characters: pid.CharacterSet("nope"), Pattern: "*"}
	if _, err := layout.EncodeIndex(0); err == nil || !strings.Contains(err.Error(), "invalid character set") {
		t.Fatalf("EncodeIndex(0) error = %v, want invalid character set", err)
	}
}
