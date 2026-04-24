package api_test

import (
	"testing"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/apitest"
)

func checkPID(t *testing.T, format api.PIDFormat, expect []string) {
	t.Helper()

	r := apitest.NewFakeRandReader()
	for i, want := range expect {
		got, err := api.GeneratePID(format, r)
		if err != nil {
			t.Fatalf("GeneratePID(%+v) call %d: %v", format, i, err)
		}
		if got != want {
			t.Fatalf("GeneratePID(%+v) call %d: got %q want %q", format, i, got, want)
		}
	}
}

func TestGeneratePID(t *testing.T) {
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
			checkPID(t, tc.format, tc.expect)
		})
	}

	t.Run("invalid", func(t *testing.T) {
		r := apitest.NewFakeRandReader()
		_, err := api.GeneratePID(api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharacters("nope")}, r)
		if err != api.ErrInvalidPIDFormat {
			t.Fatalf("GeneratePID(invalid): got err %v want %v", err, api.ErrInvalidPIDFormat)
		}
	})
}

func TestValidatePIDFormat(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		if err := api.ValidatePIDFormat(api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}); err != nil {
			t.Fatalf("ValidatePIDFormat(ok): %v", err)
		}
	})

	t.Run("emptyPattern", func(t *testing.T) {
		if err := api.ValidatePIDFormat(api.PIDFormat{Pattern: "", Characters: api.PIDCharactersFull}); err != api.ErrInvalidPIDFormat {
			t.Fatalf("ValidatePIDFormat(empty): got %v want %v", err, api.ErrInvalidPIDFormat)
		}
	})

	t.Run("noStars", func(t *testing.T) {
		if err := api.ValidatePIDFormat(api.PIDFormat{Pattern: "---___", Characters: api.PIDCharactersFull}); err != api.ErrInvalidPIDFormat {
			t.Fatalf("ValidatePIDFormat(noStars): got %v want %v", err, api.ErrInvalidPIDFormat)
		}
	})

	t.Run("invalidPatternCharacters", func(t *testing.T) {
		if err := api.ValidatePIDFormat(api.PIDFormat{Pattern: "**a-**", Characters: api.PIDCharactersFull}); err != api.ErrInvalidPIDFormat {
			t.Fatalf("ValidatePIDFormat(patternChars): got %v want %v", err, api.ErrInvalidPIDFormat)
		}
	})

	t.Run("invalidCharacters", func(t *testing.T) {
		if err := api.ValidatePIDFormat(api.PIDFormat{Pattern: "***", Characters: api.PIDCharacters("nope")}); err != api.ErrInvalidPIDFormat {
			t.Fatalf("ValidatePIDFormat(chars): got %v want %v", err, api.ErrInvalidPIDFormat)
		}
	})
}
