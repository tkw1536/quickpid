//spellchecker:words pidtest
package pidtest

//spellchecker:words bytes context errors slog slices testing time github quickpid backend internal apikey httpfixture server service
import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/httpfixture"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/server"
	"github.com/tkw1536/quickpid/service"
)

type stepConfig struct {
	NamespaceIDs []string         `json:"namespaceIDs,omitzero"`
	PIDs         []string         `json:"pids,omitzero"`
	APIKeyIDs    []string         `json:"apiKeyIDs,omitzero"`
	APIKeys      []string         `json:"apiKeys,omitzero"`
	Now          time.Time        `json:"now,omitzero"`
	InfoEnabled  bool             `json:"infoEnabled"`
	Anonymous    bool             `json:"anonymous,omitzero"`
	Meta         api.MetaResponse `json:"meta,omitzero"`

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

	// Buffer all log output, and dump it to the console if the test fails.
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		_, _ = t.Output().Write([]byte("--- FAILED TEST LOG\n"))
		_, _ = t.Output().Write(buf.Bytes())
		_, _ = t.Output().Write([]byte("--- END FAILED TEST LOG\n"))
	})

	handler := server.NewServer(server.Options{}, svc, logger)

	for _, step := range f.Steps {
		handler.SetOptions(server.Options{
			Limits:      effectiveFlowLimits(step.Limits),
			InfoEnabled: step.Config.InfoEnabled,
			Anonymous:   step.Config.Anonymous,
			Meta:        step.Config.Meta,
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

			step.Run(t, handler)

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

func effectiveFlowLimits(overrides service.Limits) service.Limits {
	limits := service.DefaultLimits()

	if overrides.MaxBodyBytes != 0 {
		limits.MaxBodyBytes = overrides.MaxBodyBytes
	}
	if overrides.DefaultPageLimit != 0 {
		limits.DefaultPageLimit = overrides.DefaultPageLimit
	}
	if overrides.MaxPageLimit != 0 {
		limits.MaxPageLimit = overrides.MaxPageLimit
	}
	if overrides.MaxBatchItems != 0 {
		limits.MaxBatchItems = overrides.MaxBatchItems
	}
	if overrides.MaxAutocompleteUsers != 0 {
		limits.MaxAutocompleteUsers = overrides.MaxAutocompleteUsers
	}
	if overrides.MaxNamespaceIDAttempts != 0 {
		limits.MaxNamespaceIDAttempts = overrides.MaxNamespaceIDAttempts
	}
	if overrides.MaxPIDAttempts != 0 {
		limits.MaxPIDAttempts = overrides.MaxPIDAttempts
	}
	if overrides.MaxAPIKeyAttempts != 0 {
		limits.MaxAPIKeyAttempts = overrides.MaxAPIKeyAttempts
	}

	return limits
}

func applyStepSetup(ctx context.Context, svc *service.Service, cfg stepConfig) error {
	if !cfg.EnsureRootUser {
		return nil
	}
	if err := svc.EnsureRootUser(ctx, slog.New(slog.DiscardHandler)); err != nil {
		return fmt.Errorf("failed to create root user: %w", err)
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

var errNoMoreNamespaceIDs = errors.New("no more namespace IDs configured")

func (r *testRuntime) NewNamespaceID() (string, error) {
	r.usedNamespaceIDs = true
	if r.namespaceIDs == nil {
		r.t.Fatalf("namespaceIDs: is not set (this is an error in the testcases themselves)")
	}
	if len(r.namespaceIDs) == 0 {
		return "", errNoMoreNamespaceIDs
	}
	id := r.namespaceIDs[0]
	r.namespaceIDs = r.namespaceIDs[1:]
	return id, nil
}

var errNoMoreAPIKeyIDs = errors.New("no more API key IDs configured")

func (r *testRuntime) NewAPIKeyID() (string, error) {
	r.usedAPIKeyIDs = true
	if r.apiKeyIDs == nil {
		r.t.Fatalf("apiKeyIDs: is not set (this is an error in the testcases themselves)")
	}
	if len(r.apiKeyIDs) == 0 {
		return "", errNoMoreAPIKeyIDs
	}
	id := r.apiKeyIDs[0]
	r.apiKeyIDs = r.apiKeyIDs[1:]
	return id, nil
}

var errNoMoreAPIKeys = errors.New("no more API keys configured")

func (r *testRuntime) NewAPIKey(_ apikey.Format) (string, error) {
	r.usedAPIKeys = true
	if r.apiKeys == nil {
		r.t.Fatalf("apiKeys: is not set (this is an error in the testcases themselves)")
	}
	if len(r.apiKeys) == 0 {
		return "", errNoMoreAPIKeys
	}
	key := r.apiKeys[0]
	r.apiKeys = r.apiKeys[1:]
	return key, nil
}

var errNoMorePIDs = errors.New("no more PIDs configured")

func (r *testRuntime) NewPID(format pid.Format) (string, error) {
	r.usedPIDs = true
	if r.pids == nil {
		r.t.Fatalf("pids: is not set (this is an error in the testcases themselves)")
	}
	if len(r.pids) == 0 {
		return "", errNoMorePIDs
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
