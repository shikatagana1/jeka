package model

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func mustIP(s string) netip.Addr {
	a, err := netip.ParseAddr(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestAggregateLastByLastSeen(t *testing.T) {
	ip := mustIP("1.2.3.4")
	t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	recs := []Record{
		{Domain: "old.example.com", LastSeen: t1, Source: "mnemonic", Kind: KindHistory},
		{Domain: "new.example.com", LastSeen: t2, Source: "mnemonic", Kind: KindHistory},
		{Domain: "ptr.example.com", Source: "livePTR", Kind: KindCurrent},
	}
	res := Aggregate(ip, recs, nil)

	if res.Last != "new.example.com" {
		t.Fatalf("Last = %q, want new.example.com", res.Last)
	}
	if len(res.Current) != 1 || res.Current[0] != "ptr.example.com" {
		t.Fatalf("Current = %v, want [ptr.example.com]", res.Current)
	}
	// History sorted newest-LastSeen first.
	if res.History[0].Domain != "new.example.com" {
		t.Fatalf("History[0] = %q, want new.example.com", res.History[0].Domain)
	}
}

func TestAggregateFallbackToCurrent(t *testing.T) {
	ip := mustIP("1.2.3.4")
	recs := []Record{
		{Domain: "b.example.com", Source: "livePTR", Kind: KindCurrent},
		{Domain: "a.example.com", Source: "livePTR", Kind: KindCurrent},
		{Domain: "hist.example.com", Source: "hackertarget", Kind: KindHistory}, // no timestamp
	}
	res := Aggregate(ip, recs, nil)
	if res.Last != "a.example.com" { // first sorted current
		t.Fatalf("Last = %q, want a.example.com", res.Last)
	}
}

func TestAggregateEmpty(t *testing.T) {
	res := Aggregate(mustIP("1.2.3.4"), nil, []string{"x: boom"})
	if res.Last != "" {
		t.Fatalf("Last = %q, want empty", res.Last)
	}
	if len(res.Errors) != 1 {
		t.Fatalf("Errors = %v", res.Errors)
	}
}

func TestAggregateMergeDuplicate(t *testing.T) {
	ip := mustIP("1.2.3.4")
	early := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	recs := []Record{
		{Domain: "dup.example.com", FirstSeen: late, LastSeen: late, Source: "mnemonic", Kind: KindHistory},
		{Domain: "dup.example.com", FirstSeen: early, LastSeen: early, Source: "securitytrails", Kind: KindHistory},
	}
	res := Aggregate(ip, recs, nil)
	if len(res.History) != 1 {
		t.Fatalf("History len = %d, want 1 (merged)", len(res.History))
	}
	h := res.History[0]
	if !h.FirstSeen.Equal(early) {
		t.Fatalf("merged FirstSeen = %v, want %v", h.FirstSeen, early)
	}
	if !h.LastSeen.Equal(late) {
		t.Fatalf("merged LastSeen = %v, want %v", h.LastSeen, late)
	}
}

func TestRecordJSONOmitsZeroTimes(t *testing.T) {
	b, err := json.Marshal(Record{Domain: "x.com", Source: "livePTR"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "first_seen") || strings.Contains(s, "last_seen") {
		t.Fatalf("zero times not omitted: %s", s)
	}
}
