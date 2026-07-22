package provider

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"

	"github.com/shikatagana1/jeka/internal/model"
)

type livePTR struct {
	r *net.Resolver
}

func NewLivePTR(r *net.Resolver) Provider { return &livePTR{r: r} }

func (p *livePTR) Name() string { return "livePTR" }

func (p *livePTR) Available() bool { return true }

func (p *livePTR) Lookup(ctx context.Context, ip netip.Addr) ([]model.Record, error) {
	names, err := p.r.LookupAddr(ctx, ip.String())
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && (dnsErr.IsNotFound || dnsErr.Err == "no such host") {
			return nil, nil
		}
		return nil, err
	}
	recs := make([]model.Record, 0, len(names))
	for _, name := range names {
		name = strings.TrimSuffix(strings.TrimSpace(name), ".")
		if name == "" {
			continue
		}
		recs = append(recs, model.Record{
			Domain: name,
			Source: p.Name(),
			Kind:   model.KindCurrent,
		})
	}
	return recs, nil
}
