package resolver

import (
	"context"
	"net"
	"time"
)

// New returns the system resolver when server is empty, otherwise one that
// dials the given DNS server (port 53 if none is specified).
func New(server string, timeout time.Duration) *net.Resolver {
	if server == "" {
		return net.DefaultResolver
	}

	if _, _, err := net.SplitHostPort(server); err != nil {
		server = net.JoinHostPort(server, "53")
	}

	d := &net.Dialer{Timeout: timeout}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return d.DialContext(ctx, network, server)
		},
	}
}
