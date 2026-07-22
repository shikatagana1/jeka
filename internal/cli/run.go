package cli

import (
	"context"
	"net/http"
	"net/netip"
	"os"
	"sync"

	"github.com/shikatagana1/jeka/internal/input"
	"github.com/shikatagana1/jeka/internal/model"
	"github.com/shikatagana1/jeka/internal/output"
	"github.com/shikatagana1/jeka/internal/provider"
	"github.com/shikatagana1/jeka/internal/resolver"
)

func Run(cfg *Config) error {
	res := resolver.New(cfg.Resolver, cfg.Timeout)

	hc := &http.Client{Timeout: cfg.Timeout}
	registry := provider.NewRegistry(
		provider.NewLivePTR(res),
		provider.NewHackerTarget(hc),
		provider.NewMnemonic(hc),
		provider.NewSecurityTrails(hc),
		provider.NewVirusTotal(hc),
	)

	ips, err := loadIPs(cfg)
	if err != nil {
		return err
	}

	results := make([]model.Result, len(ips))
	workers := cfg.Concurrency
	if workers > len(ips) {
		workers = len(ips)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				ip := ips[idx]
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
				recs, errs := registry.Query(ctx, ip)
				cancel()
				results[idx] = model.Aggregate(ip, recs, errs)
			}
		}()
	}
	for i := range ips {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	w, err := output.NewWriter(cfg.Output, os.Stdout)
	if err != nil {
		return err
	}
	return w.Write(results)
}

func loadIPs(cfg *Config) ([]netip.Addr, error) {
	if cfg.File != "" {
		return input.FromFile(cfg.File)
	}
	return input.FromArg(cfg.IP)
}
