package http

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
)

// trustedProxyCIDRs holds the CIDR ranges from which X-Real-IP / X-Forwarded-For
// headers are honoured. Anything from outside these ranges has its proxy
// headers ignored — `r.RemoteAddr` is used directly so an attacker who can
// reach the backend pod cannot spoof their source IP to bypass per-IP rate
// limits.
//
// Empty list (the dev default) means: trust nothing. The cluster Helm chart
// configures the in-cluster pod/service CIDRs in production.
var (
	trustedProxyCIDRs   []*net.IPNet
	trustedProxyCIDRsMu sync.RWMutex
)

// SetTrustedProxyCIDRs parses a comma-separated list of CIDR ranges and
// stores them for subsequent realIP() calls. Invalid entries are logged and
// skipped so a single typo doesn't disable the whole list.
//
// Idempotent and safe to call once at startup from cmd/server/main.go.
func SetTrustedProxyCIDRs(raw string) {
	trustedProxyCIDRsMu.Lock()
	defer trustedProxyCIDRsMu.Unlock()

	trustedProxyCIDRs = nil
	if raw == "" {
		slog.Info("trusted proxy CIDRs unset — proxy headers will be ignored, using RemoteAddr")
		return
	}

	parsed := make([]*net.IPNet, 0, 4)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			slog.Warn("invalid trusted proxy CIDR — entry skipped", "cidr", entry, "error", err)
			continue
		}
		parsed = append(parsed, cidr)
	}
	trustedProxyCIDRs = parsed
	slog.Info("trusted proxy CIDRs configured", "count", len(parsed))
}

// remoteAddrIP returns r.RemoteAddr stripped of its port. Handles both IPv4
// (`1.2.3.4:5678`) and IPv6 (`[::1]:5678`) by using net.SplitHostPort first,
// falling back to the raw RemoteAddr for direct unix sockets / unusual paths.
func remoteAddrIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isTrustedProxy reports whether the given IP falls inside any configured
// trusted-proxy CIDR. Returns false when the trusted list is empty (the
// dev/unconfigured default).
func isTrustedProxy(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	trustedProxyCIDRsMu.RLock()
	defer trustedProxyCIDRsMu.RUnlock()
	for _, cidr := range trustedProxyCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}
