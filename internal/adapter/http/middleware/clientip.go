package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ParseTrustedProxies reads TRUSTED_PROXIES: comma-separated CIDRs or single IPs of the
// load balancers in front of this service. Empty means nobody is trusted.
func ParseTrustedProxies(spec string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			if ip := net.ParseIP(part); ip != nil && ip.To4() != nil {
				part += "/32"
			} else {
				part += "/128"
			}
		}
		_, n, err := net.ParseCIDR(part)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// TrustedRealIP replaces chi's RealIP. X-Forwarded-For and X-Real-IP are written by
// whoever sends the request, so they only count when the connection comes from one of
// our own proxies; then the client is the right-most address that is not a proxy.
// Otherwise RemoteAddr stands, and the rate limiter cannot be dodged by sending a new
// X-Forwarded-For with every request.
func TrustedRealIP(trusted []*net.IPNet) func(http.Handler) http.Handler {
	isTrusted := func(ip net.IP) bool {
		for _, n := range trusted {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			peer := net.ParseIP(host)
			if peer != nil && len(trusted) > 0 && isTrusted(peer) {
				if client := clientFromHeaders(r, isTrusted); client != "" {
					r.RemoteAddr = client
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientFromHeaders(r *http.Request, isTrusted func(net.IP) bool) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		hops := strings.Split(xff, ",")
		for i := len(hops) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(hops[i]))
			if ip == nil {
				return ""
			}
			if !isTrusted(ip) {
				return ip.String()
			}
		}
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}
	return ""
}

// remoteIP is the address the rate limiter keys on (already resolved by TrustedRealIP).
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
