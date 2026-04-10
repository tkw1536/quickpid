package gormstore

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/tkw1536/quickpid/api"
	"gorm.io/gorm"
)

// Store implements api.Resolver using GORM. It is safe for concurrent use when backed by a
// database that serializes transactions appropriately (e.g. SQLite) or supports row locking.
type Store struct {
	db             *gorm.DB
	pidGen         func() (string, error)
	maxPIDAttempts int
}

// NewResolver returns an api.Resolver backed by db. The caller must open db with any GORM dialector.
// pidGen is used when an initial PID collides (unique index); allocation is retried at most
// maxPIDAttempts times per row (including the first try).
func NewResolver(db *gorm.DB, pidGen func() (string, error), maxPIDAttempts int) api.Resolver {
	return &Store{db: db, pidGen: pidGen, maxPIDAttempts: maxPIDAttempts}
}

var _ api.Resolver = (*Store)(nil)

func (s *Store) ListNamespaces(_ context.Context) ([]api.NamespaceResponse, error) {
	var rows []Namespace
	if err := s.db.Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]api.NamespaceResponse, len(rows))
	for i := range rows {
		out[i] = namespaceToAPI(&rows[i])
	}
	return out, nil
}

func (s *Store) CreateNamespace(_ context.Context, req api.NamespaceCreateRequest) (*api.NamespaceResponse, error) {
	var existing Namespace
	err := s.db.Where("name = ?", req.Name).First(&existing).Error
	if err == nil {
		return nil, api.ErrNamespaceAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now().UTC()
	ns := Namespace{
		Name:        req.Name,
		DateCreated: now,
	}
	if err := s.db.Create(&ns).Error; err != nil {
		return nil, err
	}
	resp := namespaceToAPI(&ns)
	return &resp, nil
}

func (s *Store) ListResources(_ context.Context, params api.ListResourcesParams) ([]api.ResourceResponse, error) {
	var ns Namespace
	if err := s.db.Where("name = ?", params.Namespace).First(&ns).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.ErrNamespaceNotFound
		}
		return nil, err
	}
	q := s.db.Where("namespace_id = ?", ns.ID)
	if params.Tag != "" {
		q = q.Where("tag = ?", params.Tag)
	}
	var rows []Resource
	if err := q.Order("pid").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]api.ResourceResponse, len(rows))
	for i := range rows {
		out[i] = resourceToAPI(&rows[i])
	}
	return out, nil
}

func (s *Store) CreateResource(ctx context.Context, namespace string, req api.ResourceCreateRequest, pid string) (*api.ResourceResponse, error) {
	var ns Namespace
	if err := s.db.WithContext(ctx).Where("name = ?", namespace).First(&ns).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.ErrNamespaceNotFound
		}
		return nil, err
	}

	candidate := pid
	for attempt := 0; attempt < s.maxPIDAttempts; attempt++ {
		if attempt > 0 {
			var err error
			candidate, err = s.pidGen()
			if err != nil {
				return nil, err
			}
		}
		now := time.Now().UTC()
		row := Resource{
			NamespaceID:  ns.ID,
			PID:          candidate,
			URL:          req.URL,
			IdInTarget:   req.IdInTarget,
			DateCreated:  now,
			DateUpdated:  now,
			TargetSystem: req.TargetSystem,
			Tag:          req.Tag,
			Deleted:      false,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			if isDuplicateKey(err) {
				continue
			}
			return nil, err
		}
		r := resourceToAPI(&row)
		return &r, nil
	}
	return nil, api.ErrPIDAllocationFailed
}

func (s *Store) BatchCreateResources(ctx context.Context, namespace string, reqs []api.ResourceCreateRequest, pids []string) ([]api.ResourceResponse, error) {
	if len(pids) != len(reqs) {
		return nil, errors.New("gormstore: len(pids) must equal len(reqs)")
	}
	if len(reqs) == 0 {
		return nil, nil
	}

	var out []api.ResourceResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ns Namespace
		if err := tx.Where("name = ?", namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return api.ErrNamespaceNotFound
			}
			return err
		}
		out = make([]api.ResourceResponse, 0, len(reqs))
		for i, req := range reqs {
			candidate := pids[i]
			var inserted bool
			for attempt := 0; attempt < s.maxPIDAttempts; attempt++ {
				if attempt > 0 {
					var err error
					candidate, err = s.pidGen()
					if err != nil {
						return err
					}
				}
				now := time.Now().UTC()
				row := Resource{
					NamespaceID:  ns.ID,
					PID:          candidate,
					URL:          req.URL,
					IdInTarget:   req.IdInTarget,
					DateCreated:  now,
					DateUpdated:  now,
					TargetSystem: req.TargetSystem,
					Tag:          req.Tag,
					Deleted:      false,
				}
				if err := tx.Create(&row).Error; err != nil {
					if isDuplicateKey(err) {
						continue
					}
					return err
				}
				out = append(out, resourceToAPI(&row))
				inserted = true
				break
			}
			if !inserted {
				return api.ErrPIDAllocationFailed
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "duplicate key")
}

func (s *Store) GetResource(_ context.Context, namespace, pid string) (*api.ResourceResponse, error) {
	var ns Namespace
	if err := s.db.Where("name = ?", namespace).First(&ns).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.ErrNamespaceNotFound
		}
		return nil, err
	}
	var row Resource
	if err := s.db.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.ErrResourceNotFound
		}
		return nil, err
	}
	r := resourceToAPI(&row)
	return &r, nil
}

func (s *Store) UpdateResource(_ context.Context, namespace, pid string, req api.ResourceUpdateRequest) (*api.ResourceResponse, error) {
	var ns Namespace
	if err := s.db.Where("name = ?", namespace).First(&ns).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.ErrNamespaceNotFound
		}
		return nil, err
	}
	var row Resource
	if err := s.db.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, api.ErrResourceNotFound
		}
		return nil, err
	}
	now := time.Now().UTC()
	row.URL = req.URL
	row.IdInTarget = req.IdInTarget
	row.TargetSystem = req.TargetSystem
	row.Tag = req.Tag
	row.Deleted = req.Deleted
	row.DateUpdated = now
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	r := resourceToAPI(&row)
	return &r, nil
}

func namespaceToAPI(n *Namespace) api.NamespaceResponse {
	return api.NamespaceResponse{
		Name:        n.Name,
		DateCreated: n.DateCreated.UTC().Format(time.RFC3339),
	}
}

func resourceToAPI(r *Resource) api.ResourceResponse {
	return api.ResourceResponse{
		PID:          r.PID,
		URL:          r.URL,
		IdInTarget:   r.IdInTarget,
		DateCreated:  r.DateCreated.UTC().Format(time.RFC3339),
		DateUpdated:  r.DateUpdated.UTC().Format(time.RFC3339),
		TargetSystem: r.TargetSystem,
		Tag:          r.Tag,
		Deleted:      r.Deleted,
	}
}
