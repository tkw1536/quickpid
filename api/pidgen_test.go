package api_test

import (
	"testing"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/apitest"
)

func TestGeneratePID(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		format api.PIDFormat
		expect []string
	}{
		"full_legacyShape": {
			format: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull},
			expect: []string{
				"yd6-lc0",
				"tha-yrf",
				"chc-pds",
			},
		},
		"readable_legacyShape": {
			format: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersReadable},
			expect: []string{
				"61e-x08",
				"hs2-akv",
				"0hc-5hg",
			},
		},
		"readable_threeChunks": {
			format: api.PIDFormat{Pattern: "***-***-***", Characters: api.PIDCharactersReadable},
			expect: []string{
				"61e-x08-hs2",
				"akv-0hc-5hg",
				"ndd-k1s-enn",
			},
		},
		"full_random64": {
			format: api.PIDFormat{Pattern: "****************************************************************", Characters: api.PIDCharactersFull},
			expect: []string{
				"yd6lc0thayrfchcpds59x79p651pd3dvc4wgkpk0iogbsx0wdt0tm49f8q4c6xgm",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := apitest.NewFakeRandReader()
			for i, want := range tc.expect {
				got, err := api.GeneratePID(tc.format, r)
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

		r := apitest.NewFakeRandReader()
		_, err := api.GeneratePID(api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharacters("nope")}, r)
		if err != api.ErrInvalidPIDFormat {
			t.Fatalf("GeneratePID(invalid): got err %v want %v", err, api.ErrInvalidPIDFormat)
		}
	})
}

func TestValidatePIDFormat(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		format    api.PIDFormat
		wantError error
	}

	cases := []testCase{
		{
			name:      "ok",
			format:    api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull},
			wantError: nil,
		},
		{
			name:      "emptyPattern",
			format:    api.PIDFormat{Pattern: "", Characters: api.PIDCharactersFull},
			wantError: api.ErrInvalidPIDFormat,
		},
		{
			name:      "noStars",
			format:    api.PIDFormat{Pattern: "---___", Characters: api.PIDCharactersFull},
			wantError: api.ErrInvalidPIDFormat,
		},
		{
			name:      "invalidPatternCharacters",
			format:    api.PIDFormat{Pattern: "**a-**", Characters: api.PIDCharactersFull},
			wantError: api.ErrInvalidPIDFormat,
		},
		{
			name:      "invalidCharacters",
			format:    api.PIDFormat{Pattern: "***", Characters: api.PIDCharacters("nope")},
			wantError: api.ErrInvalidPIDFormat,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotErr := api.ValidatePIDFormat(tc.format)
			if gotErr != tc.wantError {
				t.Fatalf("ValidatePIDFormat(%s): got err %v, want %v", tc.name, gotErr, tc.wantError)
			}
		})
	}
}
