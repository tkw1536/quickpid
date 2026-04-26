package pid_test

import (
	"testing"
	"strings"

	"github.com/tkw1536/quickpid/api/pid"
	"github.com/tkw1536/quickpid/internal/bitstring"
)

func TestFormat_Validate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name   string
		format pid.Format
		wantIn []string
	}

	cases := []testCase{
		{
			name:   "ok",
			format: pid.Format{Pattern: "***-***", Characters: pid.Full},
			wantIn: nil,
		},
		{
			name:   "emptyPattern",
			format: pid.Format{Pattern: "", Characters: pid.Full},
			wantIn: []string{"invalid pattern", "pattern must contain '*'"},
		},
		{
			name:   "noStars",
			format: pid.Format{Pattern: "---___", Characters: pid.Full},
			wantIn: []string{"invalid pattern", "pattern must contain '*'"},
		},
		{
			name:   "invalidPatternCharacters",
			format: pid.Format{Pattern: "**a-**", Characters: pid.Full},
			wantIn: []string{"invalid pattern", "pattern contains invalid characters"},
		},
		{
			name:   "invalidCharacters",
			format: pid.Format{Pattern: "***", Characters: pid.CharacterSet("nope")},
			wantIn: []string{"invalid character set", "unknown character set"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotErr := tc.format.Validate()
			if len(tc.wantIn) == 0 {
				if gotErr != nil {
					t.Fatalf("Validate(%s): got err %v, want nil", tc.name, gotErr)
				}
				return
			}
			if gotErr == nil {
				t.Fatalf("Validate(%s): got nil, want error", tc.name)
			}
			errStr := gotErr.Error()
			for _, want := range tc.wantIn {
				if !strings.Contains(errStr, want) {
					t.Fatalf("Validate(%s): got err %q, want substring %q", tc.name, errStr, want)
				}
			}
		})
	}
}

func TestFormat_Generate(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		format pid.Format
		expect []string
	}{
		"full_legacyShape": {
			format: pid.Format{Pattern: "***-***", Characters: pid.Full},
			expect: []string{
				"yd6-lc0",
				"tha-yrf",
				"chc-pds",
			},
		},
		"readable_legacyShape": {
			format: pid.Format{Pattern: "***-***", Characters: pid.Readable},
			expect: []string{
				"61e-x08",
				"hs2-akv",
				"0hc-5hg",
			},
		},
		"readable_threeChunks": {
			format: pid.Format{Pattern: "***-***-***", Characters: pid.Readable},
			expect: []string{
				"61e-x08-hs2",
				"akv-0hc-5hg",
				"ndd-k1s-enn",
			},
		},
		"full_random64": {
			format: pid.Format{Pattern: "****************************************************************", Characters: pid.Full},
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
		_, err := pid.Format{Pattern: "***-***", Characters: pid.CharacterSet("nope")}.Generate(r)
		if err == nil {
			t.Fatalf("GeneratePID(invalid): got nil want error")
		}
		if !strings.Contains(err.Error(), "invalid character set") {
			t.Fatalf("GeneratePID(invalid): got err %q want substring %q", err.Error(), "invalid character set")
		}
	})
}
