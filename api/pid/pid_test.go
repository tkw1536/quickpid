package pid

import (
	"errors"
	"testing"

	"github.com/tkw1536/quickpid/internal/bitstring"
)

func TestGeneratePID(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		format Format
		expect []string
	}{
		"full_legacyShape": {
			format: Format{Pattern: "***-***", Characters: Full},
			expect: []string{
				"yd6-lc0",
				"tha-yrf",
				"chc-pds",
			},
		},
		"readable_legacyShape": {
			format: Format{Pattern: "***-***", Characters: Readable},
			expect: []string{
				"61e-x08",
				"hs2-akv",
				"0hc-5hg",
			},
		},
		"readable_threeChunks": {
			format: Format{Pattern: "***-***-***", Characters: Readable},
			expect: []string{
				"61e-x08-hs2",
				"akv-0hc-5hg",
				"ndd-k1s-enn",
			},
		},
		"full_random64": {
			format: Format{Pattern: "****************************************************************", Characters: Full},
			expect: []string{
				"yd6lc0thayrfchcpds59x79p651pd3dvc4wgkpk0iogbsx0wdt0tm49f8q4c6xgm",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := bitstring.NewReader()
			for i, want := range tc.expect {
				got, err := tc.format.Generate(r)
				if err != nil {
					t.Fatalf("GeneratePID(%+v) call %d: %v", tc.format, i, err)
				}
				if got != want {
					t.Fatalf("GeneratePID(%+v) call %d: got %q want %q", tc.format, i, got, want)
				}
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		r := bitstring.NewReader()
		_, err := Format{Pattern: "***-***", Characters: CharacterSet("nope")}.Generate(r)
		if !errors.Is(err, errInvalidCharacterSet) {
			t.Fatalf("GeneratePID(invalid): got err %v want %v", err, errInvalidCharacterSet)
		}
	})
}

func TestValidatePIDFormat(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name   string
		format Format
		wantIs []error
	}

	cases := []testCase{
		{
			name:   "ok",
			format: Format{Pattern: "***-***", Characters: Full},
			wantIs: nil,
		},
		{
			name:   "emptyPattern",
			format: Format{Pattern: "", Characters: Full},
			wantIs: []error{errInvalidPattern, errMissingAsterisk},
		},
		{
			name:   "noStars",
			format: Format{Pattern: "---___", Characters: Full},
			wantIs: []error{errInvalidPattern, errMissingAsterisk},
		},
		{
			name:   "invalidPatternCharacters",
			format: Format{Pattern: "**a-**", Characters: Full},
			wantIs: []error{errInvalidPattern, errInvalidCharacters},
		},
		{
			name:   "invalidCharacters",
			format: Format{Pattern: "***", Characters: CharacterSet("nope")},
			wantIs: []error{errInvalidCharacterSet},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotErr := tc.format.Valid()
			if len(tc.wantIs) == 0 {
				if gotErr != nil {
					t.Fatalf("Valid(%s): got err %v, want nil", tc.name, gotErr)
				}
				return
			}
			if gotErr == nil {
				t.Fatalf("Valid(%s): got nil, want error", tc.name)
			}
			for _, want := range tc.wantIs {
				if !errors.Is(gotErr, want) {
					t.Fatalf("Valid(%s): got err %v, want errors.Is(..., %v)", tc.name, gotErr, want)
				}
			}
		})
	}
}
