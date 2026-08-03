//spellchecker:words credentials
package credentials

//spellchecker:words strings testing
import (
	"strings"
	"testing"
)

//spellchecker:words Äpfel

func TestEqualFoldASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{
			name: "empty strings",
			a:    "",
			b:    "",
			want: true,
		},
		{
			name: "identical lowercase",
			a:    "hello",
			b:    "hello",
			want: true,
		},
		{
			name: "identical uppercase",
			a:    "HELLO",
			b:    "HELLO",
			want: true,
		},
		{
			name: "ASCII case fold both ways",
			a:    "HeLLo",
			b:    "hEllO",
			want: true,
		},
		{
			name: "ASCII case fold single letter",
			a:    "A",
			b:    "a",
			want: true,
		},
		{
			name: "ASCII case fold Z",
			a:    "Z",
			b:    "z",
			want: true,
		},
		{
			name: "different lengths",
			a:    "abc",
			b:    "ab",
			want: false,
		},
		{
			name: "different ASCII letters",
			a:    "abc",
			b:    "abd",
			want: false,
		},
		{
			name: "empty and non-empty",
			a:    "",
			b:    "a",
			want: false,
		},
		{
			name: "digits and punctuation unchanged",
			a:    "User-1!",
			b:    "user-1!",
			want: true,
		},
		{
			name: "same non-ASCII runes",
			a:    "café",
			b:    "café",
			want: true,
		},
		{
			name: "non-ASCII case difference is not folded",
			a:    "Ä",
			b:    "ä",
			want: false,
		},
		{
			name: "mixed ASCII fold with identical non-ASCII",
			a:    "CAFÉ",
			b:    "cafÉ",
			want: true,
		},
		{
			name: "mixed ASCII fold with non-ASCII case difference",
			a:    "CAFÉ",
			b:    "café",
			want: false,
		},
		{
			name: "greek letters not folded",
			a:    "Α",
			b:    "α",
			want: false,
		},
		{
			name: "different rune counts with multi-byte UTF-8",
			a:    "a",
			b:    "á",
			want: false,
		},
		{
			name: "same rune count different code points",
			a:    "π",
			b:    "Π",
			want: false,
		},
		{
			name: "Bearer scheme style",
			a:    "Bearer ",
			b:    "bearer ",
			want: true,
		},
		{
			name: "Basic scheme style",
			a:    "BASIC ",
			b:    "basic ",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := equalFoldAscii(tt.a, tt.b); got != tt.want {
				t.Errorf("EqualFoldASCII(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if got := equalFoldAscii(tt.b, tt.a); got != tt.want {
				t.Errorf("EqualFoldASCII(%q, %q) = %v, want %v (symmetry)", tt.b, tt.a, got, tt.want)
			}
		})
	}
}

func TestHasPrefixAsciiFold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		prefix string
		want   bool
	}{
		{
			name:   "empty prefix on empty string",
			s:      "",
			prefix: "",
			want:   true,
		},
		{
			name:   "empty prefix on non-empty string",
			s:      "Bearer token",
			prefix: "",
			want:   true,
		},
		{
			name:   "exact match",
			s:      "Bearer ",
			prefix: "Bearer ",
			want:   true,
		},
		{
			name:   "ASCII case fold Bearer",
			s:      "bearer token",
			prefix: "Bearer ",
			want:   true,
		},
		{
			name:   "ASCII case fold Basic",
			s:      "BASIC credentials",
			prefix: "basic ",
			want:   true,
		},
		{
			name:   "mixed case scheme",
			s:      "BeArEr xyz",
			prefix: "bearer ",
			want:   true,
		},
		{
			name:   "prefix longer than string",
			s:      "Bea",
			prefix: "Bearer ",
			want:   false,
		},
		{
			name:   "non-matching prefix",
			s:      "Token abc",
			prefix: "Bearer ",
			want:   false,
		},
		{
			name:   "prefix equals full string with fold",
			s:      "HELLO",
			prefix: "hello",
			want:   true,
		},
		{
			name:   "matching ASCII prefix of longer string",
			s:      "Authorization-Value",
			prefix: "authorization",
			want:   true,
		},
		{
			name:   "same non-ASCII prefix",
			s:      "café-latte",
			prefix: "café",
			want:   true,
		},
		{
			name:   "non-ASCII case difference is not folded",
			s:      "Äpfel",
			prefix: "äpfel",
			want:   false,
		},
		{
			name:   "mixed ASCII fold with identical non-ASCII",
			s:      "CAFÉ-drink",
			prefix: "cafÉ",
			want:   true,
		},
		{
			name:   "mixed ASCII fold with non-ASCII case difference",
			s:      "CAFÉ-drink",
			prefix: "café",
			want:   false,
		},
		{
			name:   "digits and punctuation in prefix",
			s:      "User-1!rest",
			prefix: "user-1!",
			want:   true,
		},
		{
			name:   "almost matching prefix",
			s:      "Bear token",
			prefix: "Bearer ",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := hasPrefixAsciiFold(tt.s, tt.prefix); got != tt.want {
				t.Errorf("HasPrefixAsciiFold(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestTrimPrefixAsciiFold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		prefix string
		want   string
	}{
		{
			name:   "empty prefix on empty string",
			s:      "",
			prefix: "",
			want:   "",
		},
		{
			name:   "empty prefix leaves string unchanged",
			s:      "Bearer token",
			prefix: "",
			want:   "Bearer token",
		},
		{
			name:   "exact prefix match strips all",
			s:      "Bearer ",
			prefix: "Bearer ",
			want:   "",
		},
		{
			name:   "ASCII case fold Bearer",
			s:      "bearer token",
			prefix: "Bearer ",
			want:   "token",
		},
		{
			name:   "ASCII case fold Basic",
			s:      "BASIC credentials",
			prefix: "basic ",
			want:   "credentials",
		},
		{
			name:   "mixed case scheme",
			s:      "BeArEr xyz",
			prefix: "bearer ",
			want:   "xyz",
		},
		{
			name:   "prefix longer than string returns original",
			s:      "Bea",
			prefix: "Bearer ",
			want:   "Bea",
		},
		{
			name:   "non-matching prefix returns original",
			s:      "Token abc",
			prefix: "Bearer ",
			want:   "Token abc",
		},
		{
			name:   "prefix equals full string with fold",
			s:      "HELLO",
			prefix: "hello",
			want:   "",
		},
		{
			name:   "matching ASCII prefix of longer string",
			s:      "Authorization-Value",
			prefix: "authorization",
			want:   "-Value",
		},
		{
			name:   "same non-ASCII prefix",
			s:      "café-latte",
			prefix: "café",
			want:   "-latte",
		},
		{
			name:   "non-ASCII case difference is not folded",
			s:      "Äpfel",
			prefix: "äpfel",
			want:   "Äpfel",
		},
		{
			name:   "mixed ASCII fold with identical non-ASCII",
			s:      "CAFÉ-drink",
			prefix: "cafÉ",
			want:   "-drink",
		},
		{
			name:   "mixed ASCII fold with non-ASCII case difference",
			s:      "CAFÉ-drink",
			prefix: "café",
			want:   "CAFÉ-drink",
		},
		{
			name:   "digits and punctuation in prefix",
			s:      "User-1!rest",
			prefix: "user-1!",
			want:   "rest",
		},
		{
			name:   "almost matching prefix returns original",
			s:      "Bear token",
			prefix: "Bearer ",
			want:   "Bear token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := trimPrefixAsciiFold(tt.s, tt.prefix); got != tt.want {
				t.Errorf("TrimPrefixAsciiFold(%q, %q) = %q, want %q", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestEqualFoldASCIIDiffersFromStringsEqualFold(t *testing.T) {
	t.Parallel()

	// strings.EqualFold performs Unicode case folding; EqualFoldASCII must not.
	pairs := [][2]string{
		{"Ä", "ä"},
		{"İ", "i\u0307"}, // capital I with dot above vs i + combining dot
		{"Σ", "σ"},
		{"K", "k"}, // kelvin sign vs ASCII k
	}

	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		t.Run(a+"/"+b, func(t *testing.T) {
			t.Parallel()

			if !strings.EqualFold(a, b) {
				t.Skipf("strings.EqualFold(%q, %q) is false; pair does not illustrate the difference", a, b)
			}
			if equalFoldAscii(a, b) {
				t.Errorf("EqualFoldASCII(%q, %q) = true, want false (must not Unicode-fold)", a, b)
			}
		})
	}
}
