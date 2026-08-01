//spellchecker:words strict
package strict_test

//spellchecker:words encoding json strings testing time github quickpid internal strict
import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/internal/strict"
)

func ExampleString() {
	var ok strict.String
	_ = json.Unmarshal([]byte(`"hello"`), &ok)
	fmt.Println(ok)

	var bad strict.String
	err := json.Unmarshal([]byte(`null`), &bad)
	fmt.Println(err)

	// Output:
	// hello
	// json: cannot unmarshal JSON null into Go strict.String: can only unmarshal string literal
}

func ExampleBool() {
	var okTrue strict.Bool
	_ = json.Unmarshal([]byte(`true`), &okTrue)
	fmt.Println(okTrue)

	var okFalse strict.Bool
	_ = json.Unmarshal([]byte(`false`), &okFalse)
	fmt.Println(okFalse)

	var bad strict.Bool
	err := json.Unmarshal([]byte(`null`), &bad)
	fmt.Println(err)

	// Output:
	// true
	// false
	// json: cannot unmarshal JSON null into Go strict.Bool: can only unmarshal boolean literal
}

func ExampleTime() {
	var ok strict.Time
	_ = json.Unmarshal([]byte(`"2026-12-31T00:00:00Z"`), &ok)
	fmt.Println(ok.Time().UTC().Format(time.RFC3339))

	var badNull strict.Time
	err := json.Unmarshal([]byte(`null`), &badNull)
	fmt.Println(err)

	// Output:
	// 2026-12-31T00:00:00Z
	// json: cannot unmarshal JSON null into Go strict.Time: can only unmarshal string literal
}

func TestTime_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		want      string
		wantErr   bool
		wantErrIn []string
	}{
		{
			name: "ok_rfc3339",
			body: `"2026-12-31T00:00:00Z"`,
			want: "2026-12-31T00:00:00Z",
		},
		{
			name: "ok_rfc3339WithOffset",
			body: `"2026-12-31T01:00:00+01:00"`,
			want: "2026-12-31T01:00:00+01:00",
		},
		{
			name:      "fail_null",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_number",
			body:      `123`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_bool",
			body:      `true`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_object",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_malformed",
			body:      `"not-a-time"`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal RFC3339 time string"},
		},
		{
			name:      "fail_dateOnly",
			body:      `"2026-12-31"`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal RFC3339 time string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got strict.Time
			err := json.Unmarshal([]byte(tt.body), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
				return
			}
			if got.Time().Format(time.RFC3339) != tt.want {
				t.Fatalf("Time(): got %q want %q", got.Time().Format(time.RFC3339), tt.want)
			}
		})
	}
}

func ExampleStringSlice() {
	var ok strict.StringSlice
	_ = json.Unmarshal([]byte(`["a","b"]`), &ok)
	fmt.Println(ok.Strings())

	var empty strict.StringSlice
	_ = json.Unmarshal([]byte(`[]`), &empty)
	fmt.Println(empty.Strings())

	var badNull strict.StringSlice
	err := json.Unmarshal([]byte(`null`), &badNull)
	fmt.Println(err)

	// Output:
	// [a b]
	// []
	// json: cannot unmarshal JSON null into Go strict.StringSlice: can only unmarshal JSON array
}

func TestStringSlice_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		want      []string
		wantErr   bool
		wantErrIn []string
	}{
		{
			name: "ok_twoStrings",
			body: `["a","b"]`,
			want: []string{"a", "b"},
		},
		{
			name: "ok_oneString",
			body: `["only"]`,
			want: []string{"only"},
		},
		{
			name: "ok_empty",
			body: `[]`,
			want: []string{},
		},
		{
			name: "ok_emptyStringElement",
			body: `[""]`,
			want: []string{""},
		},
		{
			name:      "fail_null",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal JSON array"},
		},
		{
			name:      "fail_string",
			body:      `"hello"`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal JSON array"},
		},
		{
			name:      "fail_object",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal JSON array"},
		},
		{
			name:      "fail_number",
			body:      `123`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal JSON array"},
		},
		{
			name:      "fail_bool",
			body:      `true`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal JSON array"},
		},
		{
			name:      "fail_nullElement",
			body:      `["a",null]`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_numberElement",
			body:      `[1,"a"]`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_boolElement",
			body:      `["a",true]`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_objectElement",
			body:      `[{}]`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
		{
			name:      "fail_nestedArrayElement",
			body:      `[["a"]]`,
			wantErr:   true,
			wantErrIn: []string{"can only unmarshal string literal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got strict.StringSlice
			err := json.Unmarshal([]byte(tt.body), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
				return
			}
			if got.Strings() == nil {
				t.Fatalf("Strings(): got nil want %v", tt.want)
			}
			if len(got.Strings()) != len(tt.want) {
				t.Fatalf("Strings(): got %v want %v", got.Strings(), tt.want)
			}
			for i := range tt.want {
				if got.Strings()[i] != tt.want[i] {
					t.Fatalf("Strings(): got %v want %v", got.Strings(), tt.want)
				}
			}
		})
	}
}

func TestStringSlice_Strings(t *testing.T) {
	t.Parallel()

	s := strict.StringSlice{"x", "y"}
	got := s.Strings()
	want := []string{"x", "y"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Strings(): got %v want %v", got, want)
	}

	empty := strict.StringSlice{}
	if got := empty.Strings(); got == nil || len(got) != 0 {
		t.Fatalf("Strings() empty: got %#v want non-nil empty slice", got)
	}
}
