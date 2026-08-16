//spellchecker:words gorm
package gorm

//spellchecker:words context errors time github quickpid backend internal apikey password gorm
import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/password"
	"gorm.io/gorm"
)

type apiKeyRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey"`
	KeyID    string `gorm:"column:key_id;type:text;not null;primaryKey"`

	Comment string `gorm:"column:comment;type:text;not null"`
	// Named DateCreated (not CreatedAt) so GORM does not auto-stamp wall clock.
	DateCreated time.Time  `gorm:"column:created_at;not null"`
	ExpiresAt   *time.Time `gorm:"column:expires_at"`

	Prefix string `gorm:"column:prefix;type:text;not null;index"`
	Digest []byte `gorm:"column:digest;not null"`

	UserScopeRows      []apiKeyUserScopeRow      `gorm:"foreignKey:Username,KeyID;references:Username,KeyID;constraint:OnDelete:CASCADE"`
	NamespaceScopeRows []apiKeyNamespaceScopeRow `gorm:"foreignKey:Username,KeyID;references:Username,KeyID;constraint:OnDelete:CASCADE"`
}

func (apiKeyRow) TableName() string { return "user_api_keys" }

type apiKeyUserScopeRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey"`
	KeyID    string `gorm:"column:key_id;type:text;not null;primaryKey"`
	Pos      int    `gorm:"column:pos;not null;primaryKey"`
	Scope    string `gorm:"column:scope;type:text;not null"`
}

func (apiKeyUserScopeRow) TableName() string { return "user_api_key_user_scopes" }

type apiKeyNamespaceScopeRow struct {
	Username    string `gorm:"column:username;type:text;not null;primaryKey"`
	KeyID       string `gorm:"column:key_id;type:text;not null;primaryKey"`
	Pos         int    `gorm:"column:pos;not null;primaryKey"`
	NamespaceID string `gorm:"column:namespace_id;type:text;not null"`
	Scope       string `gorm:"column:scope;type:text;not null"`
}

func (apiKeyNamespaceScopeRow) TableName() string { return "user_api_key_namespace_scopes" }

func (k apiKeyRow) toSpec() api.APIKeyInfo {
	expiresAt := k.ExpiresAt
	if expiresAt != nil {
		utc := expiresAt.UTC()
		expiresAt = &utc
	}
	info := api.APIKeyInfo{
		ID:              k.KeyID,
		Comment:         k.Comment,
		CreatedAt:       k.DateCreated.UTC(),
		ExpiresAt:       expiresAt,
		UserScopes:      userScopesFromRows(k.UserScopeRows),
		NamespaceScopes: namespaceScopesFromRows(k.NamespaceScopeRows),
	}
	info.NormalizeScopes()
	return info
}

func userScopeRowsFor(username, keyID string, scopes []api.UserScope) []apiKeyUserScopeRow {
	rows := make([]apiKeyUserScopeRow, len(scopes))
	for i, scope := range scopes {
		rows[i] = apiKeyUserScopeRow{
			Username: username,
			KeyID:    keyID,
			Pos:      i,
			Scope:    string(scope),
		}
	}
	return rows
}

func namespaceScopeRowsFor(username, keyID string, scopes []api.APIKeyNamespaceScope) []apiKeyNamespaceScopeRow {
	rows := make([]apiKeyNamespaceScopeRow, len(scopes))
	for i, grant := range scopes {
		rows[i] = apiKeyNamespaceScopeRow{
			Username:    username,
			KeyID:       keyID,
			Pos:         i,
			NamespaceID: grant.Namespace,
			Scope:       string(grant.Scope),
		}
	}
	return rows
}

func userScopesFromRows(rows []apiKeyUserScopeRow) []api.UserScope {
	if len(rows) == 0 {
		return []api.UserScope{}
	}
	scopes := make([]api.UserScope, len(rows))
	for i := range rows {
		scopes[i] = api.UserScope(rows[i].Scope)
	}
	return scopes
}

func namespaceScopesFromRows(rows []apiKeyNamespaceScopeRow) []api.APIKeyNamespaceScope {
	if len(rows) == 0 {
		return []api.APIKeyNamespaceScope{}
	}
	scopes := make([]api.APIKeyNamespaceScope, len(rows))
	for i := range rows {
		scopes[i] = api.APIKeyNamespaceScope{
			Namespace: rows[i].NamespaceID,
			Scope:     api.NamespaceScope(rows[i].Scope),
		}
	}
	return scopes
}

func orderByPos(db *gorm.DB) *gorm.DB {
	return db.Order("pos ASC")
}

func (s *GormBackend) SetPassword(ctx context.Context, username api.ValidUsername, newPassword *api.ValidPassword) (bool, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (bool, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return false, err
		}
		usernameString := username.String()
		if newPassword == nil {
			if err := tx.Where("username = ?", usernameString).Delete(&userPasswordRow{}).Error; err != nil {
				return false, err
			}
			return false, nil
		}
		hash, hashErr := password.Hash(newPassword.String())
		if hashErr != nil {
			return false, fmt.Errorf("failed to hash password: %w", hashErr)
		}
		row := userPasswordRow{Username: usernameString, Hash: hash}
		if err := tx.Save(&row).Error; err != nil {
			return false, err
		}
		return true, nil
	})
}

func (s *GormBackend) CheckPassword(ctx context.Context, username api.ValidUsername, candidate api.ValidPassword) (bool, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (bool, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return false, err
		}
		var row userPasswordRow
		if err := tx.First(&row, "username = ?", username.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		return password.Verify(candidate.String(), row.Hash), nil
	})
}

func (s *GormBackend) CreateKey(ctx context.Context, format apikey.Format, username api.ValidUsername, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		hashed, err := format.Hash(key)
		if err != nil {
			return nil, fmt.Errorf("failed to hash key: %w", err)
		}

		usernameString := username.String()
		row := apiKeyRow{
			Username:           usernameString,
			KeyID:              keyID,
			Comment:            req.Comment,
			DateCreated:        now().UTC(),
			ExpiresAt:          req.ExpiresAt,
			Prefix:             hashed.Prefix,
			Digest:             hashed.Digest,
			UserScopeRows:      userScopeRowsFor(usernameString, keyID, req.UserScopes),
			NamespaceScopeRows: namespaceScopeRowsFor(usernameString, keyID, req.NamespaceScopes),
		}
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrKeyCollision
			}
			return nil, err
		}

		info := row.toSpec()
		return &info, nil
	})
}

func (s *GormBackend) ListKeys(ctx context.Context, _ apikey.Format, username api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedAPIKeysResponse, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		q := tx.Model(&apiKeyRow{}).Where("username = ?", username.String())
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if int64(offset) >= total {
			return &api.PaginatedAPIKeysResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.APIKeyInfo{},
			}, nil
		}

		var rows []apiKeyRow
		if err := q.Preload("UserScopeRows", orderByPos).
			Preload("NamespaceScopeRows", orderByPos).
			Order("key_id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.APIKeyInfo, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedAPIKeysResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *GormBackend) RevokeKey(ctx context.Context, _ apikey.Format, username api.ValidUsername, keyID string) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}
		var n int64
		if err := tx.Model(&apiKeyRow{}).Where("username = ? AND key_id = ?", username.String(), keyID).Count(&n).Error; err != nil {
			return zero, err
		}
		if n == 0 {
			return zero, backend.ErrKeyNotFound
		}
		if err := cascadingDelete(tx, &apiKeyRow{Username: username.String(), KeyID: keyID}); err != nil {
			return zero, err
		}
		return zero, nil
	})
	return err
}

func (s *GormBackend) LookupUserByKey(ctx context.Context, format apikey.Format, key string) (string, *api.APIKeyInfo, error) {
	return withTx2(s.db.WithContext(ctx), func(tx *gorm.DB) (string, *api.APIKeyInfo, error) {
		lookupPrefix, err := format.Prefix(key)
		if err != nil {
			return "", nil, fmt.Errorf("%w: invalid key prefix", backend.ErrInvalidKey)
		}

		var rows []apiKeyRow
		if err := tx.Preload("UserScopeRows", orderByPos).
			Preload("NamespaceScopeRows", orderByPos).
			Where("prefix = ?", lookupPrefix).Find(&rows).Error; err != nil {
			return "", nil, err
		}
		for _, row := range rows {
			if format.Verify(key, apikey.Stored{Prefix: row.Prefix, Digest: row.Digest}) {
				info := row.toSpec()
				return row.Username, &info, nil
			}
		}
		return "", nil, fmt.Errorf("%w: no valid key found", backend.ErrInvalidKey)
	})
}
