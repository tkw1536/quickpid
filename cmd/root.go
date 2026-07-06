package cmd

//spellchecker:words context errors log slog time github quickpid backend
import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

const rootUsername = "root"

const rootKeyID = "bootstrap"

func ensureRootUser(ctx context.Context, auth backend.AuthBackend, logger *slog.Logger) error {
	page, err := auth.ListUsers(ctx, api.ListUsersParams{Limit: 1})
	if err != nil {
		return err
	}
	if page.Total > 0 {
		// no users to create.
		return nil
	}

	now := time.Now
	_, err = auth.CreateUser(ctx, api.UserCreateRequest{
		Username:  rootUsername,
		Superuser: true,
	}, now)
	if errors.Is(err, backend.ErrDuplicateUsername) {
		return nil
	}
	if err != nil {
		return err
	}

	issued, err := auth.IssueKey(ctx, rootUsername, rootKeyID, api.IssueKeyRequest{
		Comment: "bootstrap",
	}, now)
	if err != nil {
		return err
	}

	logger.Warn(
		"created root superuser with bootstrap API key; store this key securely and revoke it after first use",
		slog.String("username", rootUsername),
		slog.String("key", issued.Key),
	)
	return nil
}
