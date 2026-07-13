//spellchecker:words servertest
package servertest

//spellchecker:words context fmt slices testing time github bicpid backend internal apikey httpfixture server service
import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
	"github.com/tkw1536/bicpid/internal/httpfixture"
	"github.com/tkw1536/bicpid/pid"
	"github.com/tkw1536/bicpid/server"
	"github.com/tkw1536/bicpid/service"
)

type stepConfig struct {
	NamespaceIDs []string  `json:"namespaceIDs,omitzero"`
	PIDs         []string  `json:"pids,omitzero"`
	APIKeyIDs    []string  `json:"apiKeyIDs,omitzero"`
	APIKeys      []string  `json:"apiKeys,omitzero"`
	Now          time.Time `json:"now,omitzero"`
	InfoEnabled  bool      `json:"infoEnabled"`
	Anonymous    bool      `json:"anonymous,omitzero"`

	// EnsureRootUser runs [service.Service.EnsureRootUser] before this step when the store is empty.
	// Use with apiKeyIDs/apiKeys/now so the bootstrap key is deterministic.
	EnsureRootUser bool `json:"ensureRootUser,omitzero"`
}

// flow describes an HTTP test flow in terms.
type flow struct {

	// General metadata about the flow.
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`

	// Steps are the HTTP tests to run against the server.
	Steps []struct {
		Name string `json:"name"`

		Config stepConfig `json:"config"`

		Limits service.Limits `json:"limits,omitzero"`

		httpfixture.Fixture
	} `json:"steps"`
}

func (f flow) Run(t *testing.T, s backend.Store) {
	t.Helper()

	var runtime testRuntime
	svc := service.New(s, &runtime, service.Options{})
	handler := server.NewServer(server.Options{}, svc, nil)

	for _, step := range f.Steps {
		handler.SetOptions(server.Options{
			Limits:      step.Limits,
			InfoEnabled: step.Config.InfoEnabled,
			Anonymous:   step.Config.Anonymous,
		})

		runtime.now = step.Config.Now
		runtime.namespaceIDs = slices.Clone(step.Config.NamespaceIDs)
		runtime.pids = slices.Clone(step.Config.PIDs)
		runtime.apiKeyIDs = slices.Clone(step.Config.APIKeyIDs)
		runtime.apiKeys = slices.Clone(step.Config.APIKeys)

		t.Run(step.Name, func(t *testing.T) {
			runtime.t = t

			if err := applyStepSetup(context.Background(), svc, step.Config); err != nil {
				t.Fatalf("applyStepSetup() error = %v", err)
			}

			step.Fixture.Run(t, handler)

			if !runtime.now.IsZero() && !runtime.usedNow {
				t.Errorf("now: %s was not used (this is an error in the testcases themselves)", runtime.now)
			}
			if runtime.namespaceIDs != nil && !runtime.usedNamespaceIDs {
				t.Errorf("namespaceIDs: %v was not used (this is an error in the testcases themselves)", runtime.namespaceIDs)
			}
			if runtime.pids != nil && !runtime.usedPIDs {
				t.Errorf("pids: %v was not used (this is an error in the testcases themselves)", runtime.pids)
			}
			if runtime.apiKeyIDs != nil && !runtime.usedAPIKeyIDs {
				t.Errorf("apiKeyIDs: %v was not used (this is an error in the testcases themselves)", runtime.apiKeyIDs)
			}
			if runtime.apiKeys != nil && !runtime.usedAPIKeys {
				t.Errorf("apiKeys: %v was not used (this is an error in the testcases themselves)", runtime.apiKeys)
			}

			runtime.t = nil
		})
	}
}

func applyStepSetup(ctx context.Context, svc *service.Service, cfg stepConfig) error {
	if cfg.EnsureRootUser {
		return svc.EnsureRootUser(ctx, slog.New(slog.DiscardHandler))
	}
	return nil
}

// testRuntime is a [service.Runtime] used during testing.
type testRuntime struct {
	t *testing.T

	now     time.Time
	usedNow bool

	namespaceIDs     []string
	usedNamespaceIDs bool

	pids     []string
	usedPIDs bool

	apiKeyIDs     []string
	usedAPIKeyIDs bool

	apiKeys     []string
	usedAPIKeys bool
}

func (r *testRuntime) NewNamespaceID() (string, error) {
	r.usedNamespaceIDs = true
	if r.namespaceIDs == nil {
		r.t.Fatalf("namespaceIDs: is not set (this is an error in the testcases themselves)")
	}
	if len(r.namespaceIDs) == 0 {
		return "", fmt.Errorf("no more namespace IDs configured")
	}
	id := r.namespaceIDs[0]
	r.namespaceIDs = r.namespaceIDs[1:]
	return id, nil
}

func (r *testRuntime) NewAPIKeyID() (string, error) {
	r.usedAPIKeyIDs = true
	if r.apiKeyIDs == nil {
		r.t.Fatalf("apiKeyIDs: is not set (this is an error in the testcases themselves)")
	}
	if len(r.apiKeyIDs) == 0 {
		return "", fmt.Errorf("no more API key IDs configured")
	}
	id := r.apiKeyIDs[0]
	r.apiKeyIDs = r.apiKeyIDs[1:]
	return id, nil
}

func (r *testRuntime) NewAPIKey(_ apikey.Format) (string, error) {
	r.usedAPIKeys = true
	if r.apiKeys == nil {
		r.t.Fatalf("apiKeys: is not set (this is an error in the testcases themselves)")
	}
	if len(r.apiKeys) == 0 {
		return "", fmt.Errorf("no more API keys configured")
	}
	key := r.apiKeys[0]
	r.apiKeys = r.apiKeys[1:]
	return key, nil
}

func (r *testRuntime) NewPID(format pid.Format) (string, error) {
	r.usedPIDs = true
	if r.pids == nil {
		r.t.Fatalf("pids: is not set (this is an error in the testcases themselves)")
	}
	if len(r.pids) == 0 {
		return "", fmt.Errorf("no more PIDs configured")
	}
	id := r.pids[0]
	r.pids = r.pids[1:]
	return id, nil
}

func (r *testRuntime) Now() time.Time {
	r.usedNow = true
	if r.now.IsZero() {
		r.t.Fatalf("now: is not set (this is an error in the testcases themselves)")
	}
	return r.now
}
