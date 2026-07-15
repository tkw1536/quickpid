//spellchecker:words service
package service_test

//spellchecker:words testing github bicpid service
import (
	"testing"

	"github.com/tkw1536/bicpid/service"
)

func TestLimitsWithValidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input service.Limits
		want  service.Limits
	}{
		{
			name:  "default_limits_unchanged",
			input: service.DefaultLimits(),
			want:  service.DefaultLimits(),
		},
		{
			name: "negative_values_clamped",
			input: service.Limits{
				MaxBodyBytes:           -1,
				DefaultPageLimit:       -5,
				MaxPageLimit:           -2,
				MaxBatchItems:          -3,
				MaxAutocompleteUsers:   -4,
				MaxNamespaceIDAttempts: -6,
				MaxPIDAttempts:         -7,
				MaxAPIKeyAttempts:      -8,
			},
			want: service.Limits{
				MaxBodyBytes:           0,
				DefaultPageLimit:       1,
				MaxPageLimit:           0,
				MaxBatchItems:          0,
				MaxAutocompleteUsers:   1,
				MaxNamespaceIDAttempts: 1,
				MaxPIDAttempts:         1,
				MaxAPIKeyAttempts:      1,
			},
		},
		{
			name: "zero_values_for_allow_zero_fields",
			input: service.Limits{
				MaxBodyBytes:  0,
				MaxPageLimit:  0,
				MaxBatchItems: 0,
			},
			want: service.Limits{
				MaxBodyBytes:           0,
				DefaultPageLimit:       1,
				MaxPageLimit:           0,
				MaxBatchItems:          0,
				MaxAutocompleteUsers:   1,
				MaxNamespaceIDAttempts: 1,
				MaxPIDAttempts:         1,
				MaxAPIKeyAttempts:      1,
			},
		},
		{
			name: "positive_values_unchanged",
			input: service.Limits{
				MaxBodyBytes:           2048,
				DefaultPageLimit:       25,
				MaxPageLimit:           500,
				MaxBatchItems:          50,
				MaxAutocompleteUsers:   20,
				MaxNamespaceIDAttempts: 42,
				MaxPIDAttempts:         43,
				MaxAPIKeyAttempts:      44,
			},
			want: service.Limits{
				MaxBodyBytes:           2048,
				DefaultPageLimit:       25,
				MaxPageLimit:           500,
				MaxBatchItems:          50,
				MaxAutocompleteUsers:   20,
				MaxNamespaceIDAttempts: 42,
				MaxPIDAttempts:         43,
				MaxAPIKeyAttempts:      44,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.input.WithValidValues()
			if got != tt.want {
				t.Fatalf("WithValidValues() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLimitsWithValidValuesDoesNotMutateReceiver(t *testing.T) {
	t.Parallel()

	input := service.Limits{
		MaxBodyBytes:           -1,
		DefaultPageLimit:       -1,
		MaxPageLimit:           -1,
		MaxBatchItems:          -1,
		MaxAutocompleteUsers:   -1,
		MaxNamespaceIDAttempts: -1,
		MaxPIDAttempts:         -1,
		MaxAPIKeyAttempts:      -1,
	}
	wantInput := input

	_ = input.WithValidValues()
	if input != wantInput {
		t.Fatalf("WithValidValues() mutated receiver: got %+v, want %+v", input, wantInput)
	}
}
