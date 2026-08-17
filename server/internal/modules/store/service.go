package store

import (
	"context"
	"math"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// AssetResolver resolves an asset id to a public URL. Implemented by asset.Service.
type AssetResolver interface {
	PublicURLByID(ctx context.Context, id int64) (string, error)
}

// Service provides store read operations for the mini program.
type Service struct {
	repo   Repository
	assets AssetResolver
}

// NewService builds the store service.
func NewService(repo Repository, assets AssetResolver) *Service {
	return &Service{repo: repo, assets: assets}
}

// Geo is an optional caller location used to compute store distance.
type Geo struct {
	Lat *float64
	Lng *float64
}

// ListStores returns active stores with optional distance sorting/annotation.
func (s *Service) ListStores(ctx context.Context, page httpx.Page, geo Geo) ([]StoreView, int64, error) {
	stores, total, err := s.repo.ListActiveStores(ctx, page.Limit(), page.Offset())
	if err != nil {
		return nil, 0, err
	}
	views := make([]StoreView, 0, len(stores))
	for _, st := range stores {
		views = append(views, s.storeView(ctx, st, geo))
	}
	return views, total, nil
}

// GetStore returns a single store view.
func (s *Service) GetStore(ctx context.Context, id int64, geo Geo) (StoreView, error) {
	st, err := s.repo.GetStore(ctx, id)
	if err != nil {
		return StoreView{}, err
	}
	return s.storeView(ctx, st, geo), nil
}

func (s *Service) storeView(ctx context.Context, st Store, geo Geo) StoreView {
	view := StoreView{
		ID:                       st.ID,
		Name:                     st.Name,
		Phone:                    st.Phone,
		CustomerServiceQRAssetID: st.CustomerServiceQRAssetID,
		Address:                  st.Address,
		Latitude:                 st.Latitude,
		Longitude:                st.Longitude,
		BusinessHours:            st.BusinessHours,
		Status:                   st.Status,
	}
	if st.LogoAssetID != nil {
		if url, err := s.assets.PublicURLByID(ctx, *st.LogoAssetID); err == nil {
			view.LogoURL = url
		}
	}
	if st.CustomerServiceQRAssetID != nil {
		if url, err := s.assets.PublicURLByID(ctx, *st.CustomerServiceQRAssetID); err == nil {
			view.CustomerServiceQRURL = url
		}
	}
	if geo.Lat != nil && geo.Lng != nil && st.Latitude != nil && st.Longitude != nil {
		d := int64(haversineMeters(*geo.Lat, *geo.Lng, *st.Latitude, *st.Longitude))
		view.DistanceMeters = &d
	}
	return view
}

// haversineMeters returns the great-circle distance between two points.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := rad(lat2 - lat1)
	dLng := rad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
