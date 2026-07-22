package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/shikatagana1/jeka/internal/model"
)

type hackerTarget struct {
	hc *http.Client
}

func NewHackerTarget(hc *http.Client) Provider { return &hackerTarget{hc: hc} }

func (p *hackerTarget) Name() string { return "hackertarget" }

func (p *hackerTarget) Available() bool { return true }

func (p *hackerTarget) Lookup(ctx context.Context, ip netip.Addr) ([]model.Record, error) {
	endpoint := "https://api.hackertarget.com/reversedns/?q=" + url.QueryEscape(ip.String())
	body, status, err := httpGet(ctx, p.hc, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", status)
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil, nil
	}
	// The free API reports problems as plaintext bodies, not status codes.
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(lower, "error"),
		strings.Contains(lower, "api count exceeded"),
		strings.Contains(lower, "rate limit"),
		strings.Contains(lower, "no dns"),
		strings.Contains(lower, "no records"):
		return nil, nil
	}

	var recs []model.Record
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Each line is "<ip> <hostname>".
		fields := strings.Fields(line)
		var domain string
		if len(fields) >= 2 {
			domain = fields[len(fields)-1]
		} else {
			domain = fields[0]
		}
		domain = strings.TrimSuffix(domain, ".")
		if domain == "" {
			continue
		}
		recs = append(recs, model.Record{
			Domain: domain,
			Source: p.Name(),
			Kind:   model.KindHistory,
		})
	}
	return recs, nil
}
