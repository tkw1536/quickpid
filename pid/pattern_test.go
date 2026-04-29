package pid_test

import (
	"testing"

	"github.com/tkw1536/quickpid/api/pid"
)

func TestPatternValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern pid.Pattern
		wantErr string
	}{
		{
			name:    "ok_singleAsterisk",
			pattern: "*",
			wantErr: "",
		},
		{
			name:    "ok_mixedAllowedChars",
			pattern: "*-_*--__*",
			wantErr: "",
		},
		{
			name:    "missingAsterisk_empty",
			pattern: "",
			wantErr: "pattern must contain '*'",
		},
		{
			name:    "missingAsterisk_onlyDashesAndUnderscores",
			pattern: "---___",
			wantErr: "pattern must contain '*'",
		},
		{
			name:    "invalidCharacters_letter",
			pattern: "**a-**",
			wantErr: "pattern contains invalid characters",
		},
		{
			name:    "invalidCharacters_space",
			pattern: "* *",
			wantErr: "pattern contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.pattern.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate(%q): got err %v, want nil", string(tt.pattern), err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate(%q): got nil, want error %q", string(tt.pattern), tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Validate(%q): got err %q want %q", string(tt.pattern), err.Error(), tt.wantErr)
			}
		})
	}
}
