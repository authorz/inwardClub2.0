package payment

import (
	"context"
	"time"
)

// CollectionExpiryService runs the offline-collection:expire scheduled sweep
// (spec §11, §9.3.5): pending aggregate-collection orders past their validity
// window are closed along with their payment and business orders. It only
// transitions DB state, so the worker builds it from just the store repository
// (no acquirer needed — expiry never calls the acquirer).
type CollectionExpiryService struct {
	repo StoreRepository
	now  func() time.Time
}

// NewCollectionExpiryService builds the offline-collection expiry sweep service.
func NewCollectionExpiryService(repo StoreRepository) *CollectionExpiryService {
	return &CollectionExpiryService{repo: repo, now: time.Now}
}

// SweepExpired closes every pending collection order whose validity window has
// ended. It returns the number of collection orders expired.
func (s *CollectionExpiryService) SweepExpired(ctx context.Context) (int64, error) {
	return s.repo.ExpireCollections(ctx, s.now().UTC())
}
