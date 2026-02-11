package httputil

import (
	"net"
	"net/http"
	"strings"
)

func GetClientIP(r *http.Request, trustedProxyCIDRs []string) string {
	remoteIP := peerIP(r.RemoteAddr)
	if len(trustedProxyCIDRs) == 0 {
		return remoteIP
	}
	if !isIPInCIDRs(remoteIP, trustedProxyCIDRs) {
		return remoteIP
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return remoteIP
}

func peerIP(remoteAddr string) string {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return ip
}

func isIPInCIDRs(ipStr string, cidrs []string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, cidrStr := range cidrs {
		cidrStr = strings.TrimSpace(cidrStr)
		if cidrStr == "" {
			continue
		}
		_, network, err := net.ParseCIDR(cidrStr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
