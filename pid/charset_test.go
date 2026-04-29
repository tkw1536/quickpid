package pid_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/pid"
)

func TestCharacterSetValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		chars   pid.CharacterSet
		wantErr string
	}{
		{name: "ok_full", chars: pid.Full, wantErr: ""},
		{name: "ok_readable", chars: pid.Readable, wantErr: ""},
		{name: "ok_hex", chars: pid.Hex, wantErr: ""},
		{name: "ok_decimal", chars: pid.Decimal, wantErr: ""},

		{name: "unknown_empty", chars: "", wantErr: "unknown character set"},
		{name: "unknown_custom", chars: pid.CharacterSet("nope"), wantErr: "unknown character set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.chars.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate(%q): got err %v, want nil", string(tt.chars), err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate(%q): got nil, want error %q", string(tt.chars), tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Validate(%q): got err %q want %q", string(tt.chars), err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCharacterSetAlphabet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		chars        pid.CharacterSet
		wantAlphabet string
		wantOK       bool
	}{
		{
			name:         "full",
			chars:        pid.Full,
			wantAlphabet: "0123456789abcdefghijklmnopqrstuvwxyz",
			wantOK:       true,
		},
		{
			name:         "readable",
			chars:        pid.Readable,
			wantAlphabet: "0123456789abcdefghjkmnpqrstvwxyz",
			wantOK:       true,
		},
		{
			name:         "hex",
			chars:        pid.Hex,
			wantAlphabet: "0123456789abcdef",
			wantOK:       true,
		},
		{
			name:         "decimal",
			chars:        pid.Decimal,
			wantAlphabet: "0123456789",
			wantOK:       true,
		},
		{
			name:         "unknown",
			chars:        pid.CharacterSet("nope"),
			wantAlphabet: "",
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotAlphabet, gotOK := tt.chars.Alphabet()
			if gotOK != tt.wantOK {
				t.Fatalf("Alphabet(%q): got ok %v want %v", string(tt.chars), gotOK, tt.wantOK)
			}
			if gotAlphabet != tt.wantAlphabet {
				t.Fatalf("Alphabet(%q): got %q want %q", string(tt.chars), gotAlphabet, tt.wantAlphabet)
			}
		})
	}
}

func TestCharacterSetPick(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		chars    pid.CharacterSet
		rand     io.Reader
		wantRune rune
		wantErrs []string
	}{
		"decimal_success": {
			chars:    pid.Decimal,
			rand:     bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b}),
			wantRune: '1',
		},
		"unknown_character_set": {
			chars:    pid.CharacterSet("nope"),
			rand:     bytes.NewReader(nil),
			wantErrs: []string{"unknown character set"},
		},
		"read_random_error": {
			chars:    pid.Decimal,
			rand:     errorReader{err: io.ErrUnexpectedEOF},
			wantErrs: []string{"failed to read random", "unexpected EOF"},
		},
		"all_attempts_failed": {
			chars:    pid.Decimal,
			rand:     bytes.NewReader(bytes.Repeat([]byte{0xff}, 100*8 /* 8 * pid.maxPickRetryAttempts*/)),
			wantErrs: []string{"repeated attempts to pick character fell outside of range"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.chars.Pick(tc.rand)
			if len(tc.wantErrs) == 0 {
				if err != nil {
					t.Fatalf("Pick(%q): got err %v, want nil", tc.chars, err)
				}
				if got != tc.wantRune {
					t.Fatalf("Pick(%q): got %q, want %q", tc.chars, got, tc.wantRune)
				}
				return
			}

			if err == nil {
				t.Fatalf("Pick(%q): got nil, want error", tc.chars)
			}
			errStr := err.Error()
			for _, wantErr := range tc.wantErrs {
				if !strings.Contains(errStr, wantErr) {
					t.Fatalf("Pick(%q): got err %q, want substring %q", tc.chars, errStr, wantErr)
				}
			}
		})
	}
}

// errReader always return the error when being read.
type errorReader struct {
	err error
}

func (r errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
