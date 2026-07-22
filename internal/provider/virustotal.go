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

const EnvVirusTotalKey = "VIRUSTOTAL_API_KEY"

type virusTotal struct {
	hc  *http.Client
	key string
}

func NewVirusTotal(hc *http.Client) Provider {
	return &virusTotal{hc: hc, key: os.Getenv(EnvVirusTotalKey)}
}

func (p *virusTotal) Name() string { return "virustotal" }

func (p *virusTotal) Available() bool { return p.key != "" }

type vtResolutions struct {
	Data []struct {
		Attributes struct {
			HostName string `json:"host_name"`
			Date     int64  `json:"date"`
		} `json:"attributes"`
	} `json:"data"`
}

func (p *virusTotal) Lookup(ctx context.Context, ip netip.Addr) ([]model.Record, error) {
	endpoint := "https://www.virustotal.com/api/v3/ip_addresses/" +
		url.PathEscape(ip.String()) + "/resolutions?limit=40"
	body, status, err := httpGet(ctx, p.hc, endpoint, map[string]string{"x-apikey": p.key})
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

	var vr vtResolutions
	if err := json.Unmarshal(body, &vr); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	var recs []model.Record
	for _, d := range vr.Data {
		domain := strings.TrimSuffix(strings.TrimSpace(d.Attributes.HostName), ".")
		if domain == "" {
			continue
		}
		var last time.Time
		if d.Attributes.Date > 0 {
			last = time.Unix(d.Attributes.Date, 0).UTC()
		}
		recs = append(recs, model.Record{
			Domain:   domain,
			LastSeen: last,
			Source:   p.Name(),
			Kind:     model.KindHistory,
		})
	}
	return recs, nil
}
