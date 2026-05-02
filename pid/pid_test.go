package pid_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/internal/bitstring"
	"github.com/tkw1536/quickpid/pid"
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

func TestFormat_IsValid(t *testing.T) {
	t.Parallel()

	validFmt := pid.Format{Pattern: "***-***", Characters: pid.Full}

	tests := []struct {
		name   string
		format pid.Format
		pid    string
		want   bool
	}{
		{
			name:   "example_from_generate_shape",
			format: validFmt,
			pid:    "x5x-jcc",
			want:   true,
		},
		{
			name:   "hex_charset_valid",
			format: pid.Format{Pattern: "**", Characters: pid.Hex},
			pid:    "a3",
			want:   true,
		},
		{
			name:   "decimal_charset_valid",
			format: pid.Format{Pattern: "***", Characters: pid.Decimal},
			pid:    "042",
			want:   true,
		},
		{
			name:   "readable_excludes_ambiguous_i",
			format: pid.Format{Pattern: "*", Characters: pid.Readable},
			pid:    "m",
			want:   true,
		},
		{
			name:   "wrong_separator",
			format: validFmt,
			pid:    "x5x_jcc",
			want:   false,
		},
		{
			name:   "too_short",
			format: validFmt,
			pid:    "x5-jcc",
			want:   false,
		},
		{
			name:   "too_long",
			format: validFmt,
			pid:    "x5xx-jcc",
			want:   false,
		},
		{
			name:   "wildcard_not_in_alphabet_uppercase",
			format: validFmt,
			pid:    "X5x-jcc",
			want:   false,
		},
		{
			name:   "hex_invalid_char",
			format: pid.Format{Pattern: "**", Characters: pid.Hex},
			pid:    "gg",
			want:   false,
		},
		{
			name:   "readable_invalid_char_i",
			format: pid.Format{Pattern: "*", Characters: pid.Readable},
			pid:    "i",
			want:   false,
		},
		{
			name:   "invalid_format_unknown_charset",
			format: pid.Format{Pattern: "***", Characters: pid.CharacterSet("nope")},
			pid:    "abc",
			want:   false,
		},
		{
			name:   "invalid_pattern_no_stars",
			format: pid.Format{Pattern: "---", Characters: pid.Full},
			pid:    "---",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.format.IsValid(tt.pid); got != tt.want {
				t.Fatalf("IsValid(%q): got %v, want %v", tt.pid, got, tt.want)
			}
		})
	}

	t.Run("matches_generate_output", func(t *testing.T) {
		t.Parallel()
		format := pid.Format{Pattern: "***-***", Characters: pid.Readable}
		r := bitstring.NewReader()
		for range 3 {
			s, err := format.Generate(r)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if !format.IsValid(s) {
				t.Fatalf("IsValid(%q) after Generate: got false, want true", s)
			}
		}
	})
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
				"x5x-jcc",
				"jy8-zkp",
				"wt6-ezs",
			},
		},
		"readable_legacyShape": {
			format: pid.Format{Pattern: "***-***", Characters: pid.Readable},
			expect: []string{
				"s5s-q4m",
				"76m-f4h",
				"md2-pf4",
			},
		},
		"readable_threeChunks": {
			format: pid.Format{Pattern: "***-***-***", Characters: pid.Readable},
			expect: []string{
				"s5s-q4m-76m",
				"f4h-md2-pf4",
				"hnf-6yz-135",
			},
		},
		"full_random64": {
			format: pid.Format{Pattern: "****************************************************************", Characters: pid.Full},
			expect: []string{
				"x5xjccjy8zkpwt6ezs5t7yy3dzl7tf1n9vh3pbxj5rdzl7tf1n9vh3pbr1gowxnb",
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

func TestFormat_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		want      pid.Format
		wantErrIn []string
	}{
		{
			name:    "ok",
			body:    `{"pattern":"***","characters":"full"}`,
			wantErr: false,
			want:    pid.Format{Pattern: "***", Characters: pid.Full},
		},
		{
			name:      "nullBody_isError",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "arrayBody_isError",
			body:      `[]`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "stringBody_isError",
			body:      `"x"`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "unknownField_isError",
			body:      `{"pattern":"***","characters":"full","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
		},
		{
			name:      "missingPattern",
			body:      `{"characters":"full"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "pattern"},
		},
		{
			name:      "missingCharacters",
			body:      `{"pattern":"***"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "characters"},
		},
		{
			name:      "patternNull_isError",
			body:      `{"pattern":null,"characters":"full"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal field"},
		},
		{
			name:      "charactersNull_isError",
			body:      `{"pattern":"***","characters":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal field"},
		},
		{
			name:      "patternNumber_isError",
			body:      `{"pattern":1,"characters":"full"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal field"},
		},
		{
			name:      "charactersNumber_isError",
			body:      `{"pattern":"***","characters":1}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal field"},
		},
		{
			name:      "invalidPattern_isError",
			body:      `{"pattern":"---","characters":"full"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid format", "invalid pattern"},
		},
		{
			name:      "invalidCharacterSet_isError",
			body:      `{"pattern":"***","characters":"nope"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid format", "invalid character set"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got pid.Format
			err := json.Unmarshal([]byte(tt.body), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if len(tt.wantErrIn) > 0 {
					es := err.Error()
					for _, wantIn := range tt.wantErrIn {
						if !strings.Contains(es, wantIn) {
							t.Fatalf("error: got %q want substring %q", es, wantIn)
						}
					}
				}
				return
			}
			if got != tt.want {
				t.Fatalf("format: got %+v want %+v", got, tt.want)
			}
		})
	}

	t.Run("nonValidatingUnmarshal", func(t *testing.T) {
		t.Parallel()

		type formatNoValidate struct {
			Pattern    string `json:"pattern"`
			Characters string `json:"characters"`
		}

		nonValidatingTests := []struct {
			name string
			body string
			want formatNoValidate
		}{
			{
				name: "missingFields_becomeZeroValues",
				body: `{}`,
				want: formatNoValidate{Pattern: "", Characters: ""},
			},
			{
				name: "nullFields_becomeZeroValues",
				body: `{"pattern":null,"characters":null}`,
				want: formatNoValidate{Pattern: "", Characters: ""},
			},
		}

		for _, tt := range nonValidatingTests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var got formatNoValidate
				if err := json.Unmarshal([]byte(tt.body), &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got %+v want %+v", got, tt.want)
				}
			})
		}
	})
}
