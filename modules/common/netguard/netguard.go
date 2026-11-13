/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package netguard

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// Outbound URL guard (S2): every fetch coco makes on behalf of configured
// endpoints (remote parse backends today; scrapers, MCP client fetches
// next) passes here. Two modes:
//
//   - deny-list (default): private/loopback/link-local/unique-local
//     ranges, cloud metadata addresses and non-HTTP(S) schemes are
//     rejected; everything else passes.
//   - whitelist (SSRF_WHITELIST_ON=on): ONLY listed hosts pass — checked
//     BEFORE any DNS resolution, so a hostname resolving into a private
//     range cannot slip through a name-based check.
//
// The guard validates the ADDRESS, it does not follow redirects —
// redirect-per-hop revalidation is the HTTP client wrapper's duty
// (documented for the callers that add it).

const (
	// WhitelistEnv switches to whitelist-only mode ("on"/"true"/"1").
	WhitelistEnv = "SSRF_WHITELIST"
	// WhitlistExtraEnv appends hosts to the built-in allowlist.
	WhitelistExtraEnv = "SSRF_WHITELIST_EXTRA"
)

// cloudMetadataHosts are the instance-metadata endpoints that must never
// be reachable from user-configurable fetch targets.
var cloudMetadataHosts = map[string]bool{
	"169.254.169.254":          true, // AWS/GCP/Azure/OpenStack metadata
	"metadata.google.internal": true,
	"100.100.100.200":          true, // Alibaba metadata
}

// dangerousPorts must not be fetch targets (authentication databases,
// admin ports — SSRF's favorite pivots).
var dangerousPorts = map[string]bool{
	"22":    true, // ssh
	"23":    true, // telnet
	"3306":  true, // mysql
	"5432":  true, // postgres
	"6379":  true, // redis
	"9200":  true, // elasticsearch-ish
	"11211": true, // memcached
}

// ValidateURL checks one outbound URL. The error says why, so callers can
// surface an actionable message instead of a bare "denied".
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("netguard: unparsable URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("netguard: scheme %q not allowed (http/https only)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("netguard: URL carries no host")
	}
	if dangerousPorts[u.Port()] {
		return fmt.Errorf("netguard: port %s not fetchable", u.Port())
	}

	if whitelistMode() {
		return validateWhitelisted(host)
	}
	return validateAddress(host)
}

// validateAddress resolves nothing literal it doesn't have to: literal
// IPs are checked directly; hostnames resolve once and every resulting
// address must pass — a name that resolves anywhere private is private.
func validateAddress(host string) error {
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip, host)
	}
	if cloudMetadataHosts[strings.ToLower(host)] {
		return fmt.Errorf("netguard: cloud metadata address %q not fetchable", host)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// resolution failure is not an SSRF verdict — let the fetch fail
		// with its own error rather than inventing a block reason here
		return nil
	}
	for _, ip := range ips {
		if err := checkIP(ip, host); err != nil {
			return err
		}
	}
	return nil
}

func checkIP(ip net.IP, host string) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("netguard: %q resolves into a non-routable range (%s)", host, ip)
	}
	if ip.To4() != nil && ip.IsGlobalUnicast() {
		// unique-local fc00::/7 (IPv6 private) is not covered by IsPrivate on all Go versions
		if strings.HasPrefix(ip.String(), "fc") || strings.HasPrefix(ip.String(), "fd") {
			return fmt.Errorf("netguard: %q resolves into a unique-local range (%s)", host, ip)
		}
	}
	if cloudMetadataHosts[ip.String()] {
		return fmt.Errorf("netguard: cloud metadata address %s not fetchable", ip)
	}
	return nil
}

// validateWhitelisted accepts a host only when it appears in the
// configured allowlist — name-based, pre-DNS, exactly the WeKnora
// "whitelist-only mode" semantics.
func validateWhitelisted(host string) error {
	for _, allowed := range whitelistHosts() {
		if strings.EqualFold(strings.TrimSpace(allowed), host) {
			return nil
		}
	}
	return fmt.Errorf("netguard: host %q not in the SSRF whitelist", host)
}

func whitelistMode() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(WhitelistEnv))) {
	case "on", "true", "1":
		return true
	}
	return false
}

// whitelistHosts reads the extra allowlist; an EMPTY list in whitelist
// mode denies every host — fail-closed on purpose, the operator asked for
// whitelist-only and configured nothing.
func whitelistHosts() []string {
	out := []string{}
	for _, v := range strings.Split(os.Getenv(WhitelistExtraEnv), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
