package apitest

import (
	"testing"

	"github.com/tkw1536/quickpid/internal/steptest"
)

// Flow describes an HTTP test flow in terms of deterministic randomness and steps.
type Flow struct {
	NamespaceIDs []string
	PIDs         []string
	Steps        []steptest.Step
}

func (f Flow) Run(t *testing.T, h *harness) {
	t.Helper()
	h.useFixedRandomness(t, f.NamespaceIDs, f.PIDs)
	h.runSteps(t, f.Steps)
}
