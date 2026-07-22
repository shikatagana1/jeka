package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/shikatagana1/jeka/internal/model"
)

type mnemonic struct {
	hc *http.Client
}

func NewMnemonic(hc *http.Client) Provider { return &mnemonic{hc: hc} }

func (p *mnemonic) Name() string { return "mnemonic" }

func (p *mnemonic) Available() bool { return true }

type mnemonicResponse struct {
	ResponseCode int    `json:"responseCode"`
	Message      string `json:"message"`
	Data         []struct {
		Query     string `json:"query"`
		Answer    string `json:"answer"`
		RRType    string `json:"rrtype"`
		FirstSeen int64  `json:"firstSeenTimestamp"`
		LastSeen  int64  `json:"lastSeenTimestamp"`
	} `json:"data"`
}

func (p *mnemonic) Lookup(ctx context.Context, ip netip.Addr) ([]model.Record, error) {
	endpoint := "https://api.mnemonic.no/pdns/v3/" + url.PathEscape(ip.String())
	body, status, err := httpGet(ctx, p.hc, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", status)
	}

	var mr mnemonicResponse
	if err := json.Unmarshal(body, &mr); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	target := ip.String()
	var recs []model.Record
	for _, d := range mr.Data {
		// We want records where the IP is the answer and the domain is the query.
		if strings.TrimSuffix(d.Answer, ".") != target {
			continue
		}
		switch strings.ToUpper(d.RRType) {
		case "A", "AAAA", "PTR":
		default:
			continue
		}
		domain := strings.TrimSuffix(strings.TrimSpace(d.Query), ".")
		if domain == "" {
			continue
		}
		recs = append(recs, model.Record{
			Domain:    domain,
			FirstSeen: millisToTime(d.FirstSeen),
			LastSeen:  millisToTime(d.LastSeen),
			Source:    p.Name(),
			Kind:      model.KindHistory,
		})
	}
	return recs, nil
}

func millisToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}
