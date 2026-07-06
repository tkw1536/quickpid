package apikey_test

//spellchecker:words apikey crypto sha256 errors fmt io strings testing github quickpid internal bitstring
import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/tkw1536/bicpid/internal/apikey"
	"github.com/tkw1536/bicpid/internal/bitstring"
	"github.com/tkw1536/bicpid/pid"
)

func testFormat() apikey.Format {
	return apikey.Format{
		Length:    16,
		PrefixLen: 4,
		Charset:   pid.Full,
	}
}

func TestFormat_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		format  apikey.Format
		wantErr string
	}{
		{
			name:   "ok",
			format: testFormat(),
		},
		{
			name: "zeroPrefixLen",
			format: apikey.Format{
				Length:    16,
				PrefixLen: 0,
				Charset:   pid.Full,
			},
			wantErr: "prefix length must be positive",
		},
		{
			name: "lengthEqualsPrefixLen",
			format: apikey.Format{
				Length:    4,
				PrefixLen: 4,
				Charset:   pid.Full,
			},
			wantErr: "key length must exceed prefix length",
		},
		{
			name: "invalidCharset",
			format: apikey.Format{
				Length:    16,
				PrefixLen: 4,
				Charset:   pid.CharacterSet("invalid"),
			},
			wantErr: "charset must be valid: unknown character set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.format.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestDefault_Validate(t *testing.T) {
	t.Parallel()

	if err := apikey.Default.Validate(); err != nil {
		t.Fatalf("Default.Validate() error = %v", err)
	}
	if apikey.Default.Length != 32 || apikey.Default.PrefixLen != 8 {
		t.Fatalf("Default = %+v", apikey.Default)
	}
}

func TestFormat_Generate_deterministic(t *testing.T) {
	t.Parallel()

	format := apikey.Format{
		Length:    12,
		PrefixLen: 4,
		Charset:   pid.Full,
	}

	key, err := format.Generate(bitstring.NewReader())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if key != "x5xjccjy8zkp" {
		t.Fatalf("Generate() = %q, want x5xjccjy8zkp", key)
	}
}

func TestFormat_Generate_invalidFormat(t *testing.T) {
	t.Parallel()

	_, err := apikey.Format{
		Length:    4,
		PrefixLen: 4,
		Charset:   pid.Full,
	}.Generate(bitstring.NewReader())
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
}

func TestFormat_Generate_randError(t *testing.T) {
	t.Parallel()

	_, err := testFormat().Generate(errReader{})
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
	if !errors.Is(err, errReaderErr) {
		t.Fatalf("Generate() error = %v, want %v", err, errReaderErr)
	}
}

func TestFormat_Prefix(t *testing.T) {
	t.Parallel()

	format := testFormat()
	key := "abcaabbaccabbaab"

	prefix, err := format.Prefix(key)
	if err != nil {
		t.Fatalf("Prefix() error = %v", err)
	}
	if prefix != "abca" {
		t.Fatalf("Prefix() = %q, want abca", prefix)
	}
}

func TestFormat_Prefix_invalidKey(t *testing.T) {
	t.Parallel()

	format := testFormat()

	if _, err := format.Prefix("short"); err == nil {
		t.Fatal("Prefix() short key error = nil, want error")
	}
	if _, err := format.Prefix("x5xjccjy8zkp"); err == nil {
		t.Fatal("Prefix() invalid char error = nil, want error")
	}
}

func TestFormat_Hash(t *testing.T) {
	t.Parallel()

	format := testFormat()
	key := "abcaabbaccabbaab"

	stored, err := format.Hash(key)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if stored.Prefix != "abca" {
		t.Fatalf("Hash().Prefix = %q, want abca", stored.Prefix)
	}

	wantDigest := sha256.Sum256([]byte("abbaccabbaab"))
	if string(stored.Digest) != string(wantDigest[:]) {
		t.Fatalf("Hash().Digest mismatch")
	}
}

func TestFormat_Hash_nilHasherUsesSHA256(t *testing.T) {
	t.Parallel()

	format := apikey.Format{
		Length:    16,
		PrefixLen: 4,
		Charset:   pid.Full,
	}
	key := "abcaabbaccabbaab"

	stored, err := format.Hash(key)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	want := sha256.Sum256([]byte("abbaccabbaab"))
	if string(stored.Digest) != string(want[:]) {
		t.Fatal("Hash() with nil Hasher did not use SHA-256")
	}
}

func TestFormat_Hash_customHasher(t *testing.T) {
	t.Parallel()

	format := apikey.Format{
		Length:    16,
		PrefixLen: 4,
		Charset:   pid.Full,
		Hasher: func(secret []byte) []byte {
			return []byte("custom:" + string(secret))
		},
	}
	key := "abcaabbaccabbaab"

	stored, err := format.Hash(key)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if string(stored.Digest) != "custom:abbaccabbaab" {
		t.Fatalf("Hash().Digest = %q", stored.Digest)
	}
}

func TestFormat_Verify(t *testing.T) {
	t.Parallel()

	format := testFormat()
	key := "abcaabbaccabbaab"

	stored, err := format.Hash(key)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	tests := []struct {
		name      string
		key       string
		stored    apikey.Stored
		wantMatch bool
	}{
		{
			name:      "match",
			key:       key,
			stored:    stored,
			wantMatch: true,
		},
		{
			name:      "wrongSuffix",
			key:       "abcaabbaccabbaac",
			stored:    stored,
			wantMatch: false,
		},
		{
			name:      "wrongStoredPrefix",
			key:       key,
			stored:    apikey.Stored{Prefix: "bbbb", Digest: stored.Digest},
			wantMatch: false,
		},
		{
			name:      "tamperedDigest",
			key:       key,
			stored:    apikey.Stored{Prefix: stored.Prefix, Digest: append([]byte(nil), stored.Digest...)},
			wantMatch: false,
		},
		{
			name:      "changedPrefixChar",
			key:       "cbcaabbaccabbaab",
			stored:    stored,
			wantMatch: false,
		},
		{
			name:      "sameSuffixDifferentPrefix",
			key:       "bbcaabbaccabbaab",
			stored:    stored,
			wantMatch: false,
		},
		{
			name:      "correctPrefixWrongSuffix",
			key:       "abca" + strings.Repeat("c", format.Length-format.PrefixLen),
			stored:    stored,
			wantMatch: false,
		},
		{
			name:      "invalidKeyLength",
			key:       "short",
			stored:    stored,
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.name == "tamperedDigest" {
				tc.stored.Digest[0] ^= 0xff
			}

			got := format.Verify(tc.key, tc.stored)
			if got != tc.wantMatch {
				t.Fatalf("Verify() = %v, want %v", got, tc.wantMatch)
			}
		})
	}
}

func ExampleFormat_roundTrip() {
	format := apikey.Format{
		Length:    12,
		PrefixLen: 4,
		Charset:   pid.Full,
	}

	key, err := format.Generate(bitstring.NewReader())
	if err != nil {
		panic(err)
	}

	stored, err := format.Hash(key)
	if err != nil {
		panic(err)
	}

	prefix, err := format.Prefix(key)
	if err != nil {
		panic(err)
	}

	fmt.Println(key)
	fmt.Println(prefix)
	fmt.Println(format.Verify(key, stored))
	fmt.Println(format.Verify("x5xjccjy8zk", stored))

	// Output:
	// x5xjccjy8zkp
	// x5xj
	// true
	// false
}

var errReaderErr = errors.New("read failed")

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errReaderErr
}

var _ io.Reader = errReader{}
