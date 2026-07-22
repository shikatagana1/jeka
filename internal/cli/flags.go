package cli

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/shikatagana1/jeka/internal/output"
)

type Config struct {
	IP          string
	File        string
	Output      output.Format
	Concurrency int
	Resolver    string
	Timeout     time.Duration
}

const (
	DefaultConcurrency = 8
	DefaultTimeout     = 5 * time.Second
	DefaultOutput      = output.FormatText
)

func Parse(args []string) (*Config, error) {
	fs := flag.NewFlagSet("jeka", flag.ContinueOnError)

	var (
		file        string
		outputStr   string
		concurrency int
		resolver    string
		timeout     time.Duration
	)

	fs.StringVar(&file, "file", "", "file of IPs, one per line (# comments); mutually exclusive with positional IP")
	fs.StringVar(&file, "f", "", "shorthand for --file")
	fs.StringVar(&outputStr, "output", string(DefaultOutput), "output format: text|json|csv")
	fs.StringVar(&outputStr, "o", string(DefaultOutput), "shorthand for --output")
	fs.IntVar(&concurrency, "concurrency", DefaultConcurrency, "worker-pool size for file mode")
	fs.IntVar(&concurrency, "c", DefaultConcurrency, "shorthand for --concurrency")
	fs.StringVar(&resolver, "resolver", "", "custom DNS server for PTR lookups (host[:port]); system resolver if empty")
	fs.DurationVar(&timeout, "timeout", DefaultTimeout, "per-lookup timeout")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "jeka - reverse-DNS lookup (current PTR + passive-DNS history)\n\n")
		fmt.Fprintf(fs.Output(), "Usage:\n  jeka [flags] <ip>\n  jeka [flags] -f <file>\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), "\nOptional API keys (enable extra providers when set):\n")
		fmt.Fprintf(fs.Output(), "  SECURITYTRAILS_API_KEY   enables the securitytrails provider\n")
		fmt.Fprintf(fs.Output(), "  VIRUSTOTAL_API_KEY       enables the virustotal provider\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg := &Config{
		File:        file,
		Output:      output.Format(strings.ToLower(strings.TrimSpace(outputStr))),
		Concurrency: concurrency,
		Resolver:    resolver,
		Timeout:     timeout,
	}

	rest := fs.Args()
	if len(rest) > 1 {
		return nil, fmt.Errorf("too many positional arguments: expected a single IP, got %d", len(rest))
	}
	if len(rest) == 1 {
		cfg.IP = rest[0]
	}

	switch {
	case cfg.IP == "" && cfg.File == "":
		return nil, fmt.Errorf("no input: provide a single IP argument or --file")
	case cfg.IP != "" && cfg.File != "":
		return nil, fmt.Errorf("provide either a single IP or --file, not both")
	}

	switch cfg.Output {
	case output.FormatText, output.FormatJSON, output.FormatCSV:
	default:
		return nil, fmt.Errorf("invalid --output %q: must be text, json, or csv", cfg.Output)
	}

	if cfg.Concurrency <= 0 {
		return nil, fmt.Errorf("--concurrency must be > 0, got %d", cfg.Concurrency)
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("--timeout must be > 0, got %s", cfg.Timeout)
	}

	return cfg, nil
}
