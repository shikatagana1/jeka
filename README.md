# jeka

A reverse-DNS lookup CLI. Give it an IP address (or a file of them) and it reports:

- the current PTR domain(s) the IP resolves to (live DNS),
- the historical domains the IP has pointed to, from passive-DNS providers, with
  first/last-seen timestamps where the source provides them, and
- the single most recent ("last") domain across all sources.

Providers are queried concurrently. If one is down, rate-limited, or missing an API key,
the run continues and the failure is reported per IP rather than aborting.

No third-party dependencies. Module `github.com/shikatagana1/jeka`, Go 1.26.

---

## Install

With the Go toolchain:

```sh
go install github.com/shikatagana1/jeka@latest
```

Or build from a clone:

```sh
git clone https://github.com/shikatagana1/jeka.git
cd jeka
go build -o jeka .
```

Or run without installing:

```sh
go run . 8.8.8.8
```

## Usage

```
jeka [flags] <ip>
jeka [flags] -f <file>
```

### Flags

| Flag | Alias | Type | Default | Meaning |
|------|-------|------|---------|---------|
| `--file` | `-f` | string | `""` | File of IPs (one per line; `#` comments and blank lines ignored). Mutually exclusive with the positional IP. |
| `--output` | `-o` | string | `text` | Output format: `text`, `json`, or `csv`. |
| `--concurrency` | `-c` | int | `8` | Worker-pool size for file mode. |
| `--resolver` | | string | `""` | Custom DNS server for PTR lookups (`host` or `host:port`, port defaults to 53). System resolver if empty. |
| `--timeout` | | duration | `5s` | Per-lookup timeout (applies to each IP's provider fan-out). |
| *(positional)* | | string | | A single IP address (IPv4 or IPv6), used when `--file` is not given. |

Exit codes: `0` success, `1` runtime error, `2` usage / flag error.

## Examples

Single IP (text):

```sh
jeka 8.8.8.8
```

File of IPs, JSON output, 16 workers:

```sh
jeka -c 16 -o json -f targets.txt
```

CSV output for a single IPv6 address:

```sh
jeka -o csv 2001:4860:4860::8888
```

Use a custom resolver and a longer timeout for the live PTR lookup:

```sh
jeka --resolver 1.1.1.1 --timeout 10s 8.8.8.8
```

Input file format (`targets.txt`):

```
# one IP per line; blank lines and #-comment lines are ignored
8.8.8.8
1.1.1.1
2606:4700:4700::1111
```

### Sample output

```
== 1.1.1.1 ==
Current:
  * one.one.one.one
History:
  DOMAIN                FIRST SEEN            LAST SEEN             SOURCE
  example-cdn.net       2021-03-04T00:00:00Z  2024-11-18T00:00:00Z  mnemonic
  old-host.example      2019-08-12T00:00:00Z  2020-01-30T00:00:00Z  mnemonic
  legacy.example.org    -                     -                     hackertarget
Last: one.one.one.one
```

The `Last:` line (also marked with `*` in the current block) is the most recent domain
across all sources. See [How it works](#how-it-works) below.

## Output formats

- `text`: a human-readable block per IP with the IP header, the current PTR domain(s), a
  history table (domain / first seen / last seen / source), and the highlighted `Last:`
  domain. The last domain is marked with `*`.
- `json`: a `[]Result` array via `json.MarshalIndent`. Zero-valued timestamps are omitted
  rather than serialized as `0001-01-01T00:00:00Z`.
- `csv`: flat rows with header `ip,domain,first_seen,last_seen,source,kind,is_last`, one
  row per record. `kind` is `current` or `history`; `is_last` marks the row whose domain
  equals the computed last domain. Timestamps are RFC3339 (empty when unknown).

## Providers

Data is gathered from several sources. If a provider fails or is unavailable, the run
continues and the failure is recorded in the per-IP `errors` list.

| Provider | Kind | API key | Notes |
|----------|------|---------|-------|
| `livePTR` | current | no | Live PTR via the configured resolver (`net.Resolver.LookupAddr`). |
| `hackertarget` | history | no | `api.hackertarget.com` reverse-DNS (free tier, rate-limited; no timestamps). |
| `mnemonic` | history | no | Mnemonic PassiveDNS v3 (`api.mnemonic.no`); can provide first/last-seen timestamps. |
| `securitytrails` | history | **yes** | Enabled only when `SECURITYTRAILS_API_KEY` is set. |
| `virustotal` | history | **yes** | Enabled only when `VIRUSTOTAL_API_KEY` is set. |

### API-key environment variables

Key-gated providers are **skipped (not errored)** when their key is absent:

```sh
export SECURITYTRAILS_API_KEY=...   # enables the securitytrails provider
export VIRUSTOTAL_API_KEY=...       # enables the virustotal provider
```

## How it works

For each IP, jeka fans out to every *available* provider under a shared, timeout-bounded
`*http.Client`. Each provider returns a set of `Record`s (domain + optional first/last
seen + source + kind). The results are then aggregated:

1. Current domains come from `livePTR`, deduped and sorted.
2. History domains are merged across providers: duplicate domains keep the earliest
   `FirstSeen` and the latest `LastSeen`, then are sorted by `LastSeen` descending.
3. The last domain is the one with the maximum `LastSeen` across all records (lexical
   tie-break). If no record carries a timestamp, it falls back to the first current
   domain; if there are none, it is empty.

File mode runs this pipeline across an ordered worker pool (`-c`), so output always
matches input order regardless of concurrency.

### Project layout

```
jeka/
├── main.go                     # entrypoint: parse flags, run, set exit code
└── internal/
    ├── cli/                    # flag parsing and the run pipeline
    ├── input/                  # single-IP and file input parsing
    ├── resolver/               # net.Resolver construction (custom server support)
    ├── model/                  # Record, Result, and the Aggregate() logic
    ├── provider/               # Provider interface and one file per source
    └── output/                 # text / json / csv writers
```

Adding a new passive-DNS source is a matter of implementing the `Provider` interface
(`Name`, `Available`, `Lookup`) and registering it in `internal/cli/run.go`.

## Caveats

- **Free-tier history is often timestamp-less.** Mnemonic's unauthenticated (TLP-white)
  responses return zeroed first/last-seen fields, so those rows show `-` and history sort
  falls back to lexical order. Supplying a SecurityTrails or VirusTotal API key yields
  richer, timestamped history.
- The free `hackertarget` and `mnemonic` endpoints are rate-limited; large batches may see
  occasional soft failures, which degrade gracefully rather than aborting the run.
- `securitytrails` and `virustotal` are implemented against their documented API shapes but
  are gated off unless the corresponding env key is set.

## Development

```sh
go build ./...
go vet ./...
go test ./...
go test -race ./...
```

## License

Released under the MIT License. See [LICENSE](LICENSE).
