//spellchecker:words backend authentication
package authentication

//spellchecker:words crypto github quickpid internal apikey
import (
	"crypto/rand"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/internal/apikey"
)

func generateAndCreateKey(format apikey.Format, create func(rawKey string) (*api.APIKeyInfo, error)) (*api.IssueKeyResponse, error) {
	rawKey, err := format.Generate(rand.Reader)
	if err != nil {
		return nil, err
	}
	info, err := create(rawKey)
	if err != nil {
		return nil, err
	}
	return &api.IssueKeyResponse{APIKeyInfo: *info, Key: rawKey}, nil
}
