package input

import (
	"bufio"
	"fmt"
	"net/netip"
	"os"
	"strings"
)

func FromArg(s string) ([]netip.Addr, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("invalid IP address %q: %w", s, err)
	}
	return []netip.Addr{addr}, nil
}

// FromFile reads one IP per line. Blank lines and lines starting with '#' are
// skipped; any other unparseable line is an error with its line number.
func FromFile(path string) ([]netip.Addr, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var addrs []netip.Addr
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: invalid IP address %q: %w", path, line, raw, err)
		}
		addrs = append(addrs, addr)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%s: no IP addresses found", path)
	}
	return addrs, nil
}
