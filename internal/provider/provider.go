package provider

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/shikatagana1/jeka/internal/model"
)

type Provider interface {
	Name() string
	Available() bool
	Lookup(ctx context.Context, ip netip.Addr) ([]model.Record, error)
}

type Registry struct {
	providers []Provider
}

func NewRegistry(ps ...Provider) *Registry {
	return &Registry{providers: ps}
}

// Query runs every available provider and merges their records. A provider
// error is recorded in errs and does not abort the lookup.
func (r *Registry) Query(ctx context.Context, ip netip.Addr) (recs []model.Record, errs []string) {
	for _, p := range r.providers {
		if !p.Available() {
			continue
		}
		rs, err := p.Lookup(ctx, ip)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.Name(), err))
			continue
		}
		recs = append(recs, rs...)
	}
	return recs, errs
}
