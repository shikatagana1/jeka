package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shikatagana1/jeka/internal/model"
)

const EnvSecurityTrailsKey = "SECURITYTRAILS_API_KEY"

type securityTrails struct {
	hc  *http.Client
	key string
}

func NewSecurityTrails(hc *http.Client) Provider {
	return &securityTrails{hc: hc, key: os.Getenv(EnvSecurityTrailsKey)}
}

func (p *securityTrails) Name() string { return "securitytrails" }

func (p *securityTrails) Available() bool { return p.key != "" }

type stResponse struct {
	Records []struct {
		Hostname  string `json:"hostname"`
		FirstSeen string `json:"first_seen"`
		LastSeen  string `json:"last_seen"`
	} `json:"records"`
}

func (p *securityTrails) Lookup(ctx context.Context, ip netip.Addr) ([]model.Record, error) {
	endpoint := "https://api.securitytrails.com/v1/ips/" + url.PathEscape(ip.String()) + "/dns"
	body, status, err := httpGet(ctx, p.hc, endpoint, map[string]string{"APIKEY": p.key})
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, nil
	case http.StatusForbidden, http.StatusUnauthorized:
		return nil, fmt.Errorf("auth/plan error (status %d)", status)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited (status %d)", status)
	default:
		return nil, fmt.Errorf("unexpected status %d", status)
	}

	var sr stResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	var recs []model.Record
	for _, r := range sr.Records {
		domain := strings.TrimSuffix(strings.TrimSpace(r.Hostname), ".")
		if domain == "" {
			continue
		}
		recs = append(recs, model.Record{
			Domain:    domain,
			FirstSeen: parseDate(r.FirstSeen),
			LastSeen:  parseDate(r.LastSeen),
			Source:    p.Name(),
			Kind:      model.KindHistory,
		})
	}
	return recs, nil
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
