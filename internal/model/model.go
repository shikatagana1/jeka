package model

import (
	"encoding/json"
	"net/netip"
	"sort"
	"time"
)

type Kind int

const (
	KindCurrent Kind = iota
	KindHistory
)

type Record struct {
	Domain    string    `json:"domain"`
	FirstSeen time.Time `json:"first_seen,omitempty"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
	Source    string    `json:"source"`
	Kind      Kind      `json:"-"`
}

// omitempty does not work on time.Time, so drop zero timestamps by hand.
func (r Record) MarshalJSON() ([]byte, error) {
	type alias struct {
		Domain    string     `json:"domain"`
		FirstSeen *time.Time `json:"first_seen,omitempty"`
		LastSeen  *time.Time `json:"last_seen,omitempty"`
		Source    string     `json:"source"`
	}
	a := alias{Domain: r.Domain, Source: r.Source}
	if !r.FirstSeen.IsZero() {
		a.FirstSeen = &r.FirstSeen
	}
	if !r.LastSeen.IsZero() {
		a.LastSeen = &r.LastSeen
	}
	return json.Marshal(a)
}

type Result struct {
	IP      netip.Addr `json:"ip"`
	Current []string   `json:"current"`
	History []Record   `json:"history"`
	Last    string     `json:"last,omitempty"`
	Errors  []string   `json:"errors,omitempty"`
}

func Aggregate(ip netip.Addr, recs []Record, errs []string) Result {
	res := Result{IP: ip}
	if len(errs) > 0 {
		res.Errors = errs
	}

	currentSet := map[string]struct{}{}
	histByDomain := map[string]*Record{}

	for _, r := range recs {
		if r.Domain == "" {
			continue
		}
		switch r.Kind {
		case KindCurrent:
			currentSet[r.Domain] = struct{}{}
		case KindHistory:
			existing, ok := histByDomain[r.Domain]
			if !ok {
				rc := r
				histByDomain[r.Domain] = &rc
				continue
			}
			if !r.FirstSeen.IsZero() && (existing.FirstSeen.IsZero() || r.FirstSeen.Before(existing.FirstSeen)) {
				existing.FirstSeen = r.FirstSeen
			}
			if !r.LastSeen.IsZero() && r.LastSeen.After(existing.LastSeen) {
				existing.LastSeen = r.LastSeen
			}
			if existing.Source == "" {
				existing.Source = r.Source
			}
		}
	}

	current := make([]string, 0, len(currentSet))
	for d := range currentSet {
		current = append(current, d)
	}
	sort.Strings(current)
	res.Current = current

	history := make([]Record, 0, len(histByDomain))
	for _, r := range histByDomain {
		history = append(history, *r)
	}
	sort.Slice(history, func(i, j int) bool {
		a, b := history[i], history[j]
		az, bz := a.LastSeen.IsZero(), b.LastSeen.IsZero()
		if az != bz {
			return !az
		}
		if !az && !a.LastSeen.Equal(b.LastSeen) {
			return a.LastSeen.After(b.LastSeen)
		}
		return a.Domain < b.Domain
	})
	res.History = history

	res.Last = computeLast(recs, current)
	return res
}

func computeLast(recs []Record, sortedCurrent []string) string {
	var bestDomain string
	var bestTime time.Time
	found := false
	for _, r := range recs {
		if r.Domain == "" || r.LastSeen.IsZero() {
			continue
		}
		if !found || r.LastSeen.After(bestTime) ||
			(r.LastSeen.Equal(bestTime) && r.Domain < bestDomain) {
			bestDomain = r.Domain
			bestTime = r.LastSeen
			found = true
		}
	}
	if found {
		return bestDomain
	}
	if len(sortedCurrent) > 0 {
		return sortedCurrent[0]
	}
	return ""
}
