package shorturl

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidateURL checks the service's scheme and size policy without rewriting input.
// DNS names must be ASCII labels (punycode allowed), with no trailing dot.
// Canonical IPv4 and bracketed IPv6 literals are accepted, including private IPs;
// IPv6 zones are rejected. Explicit ports must be canonical decimal 1..65535.
// This is syntax validation, not a public-network or destination trust check.
func (s *Service) ValidateURL(input string) error {
	if len(input) == 0 || len(input) > s.cfg.MaxURLBytes || !utf8.ValidString(input) || strings.TrimSpace(input) != input {
		return ErrInvalidURL
	}
	for _, r := range input {
		if unicode.IsControl(r) || r == '\\' {
			return ErrInvalidURL
		}
	}
	// Check escapes in every component, including RawQuery, which url.Parse leaves unchecked.
	for i := 0; i < len(input); i++ {
		if input[i] != '%' {
			continue
		}
		if i+2 >= len(input) {
			return ErrInvalidURL
		}
		b, err := strconv.ParseUint(input[i+1:i+3], 16, 8)
		if err != nil || b < 32 || b == 127 || b == '\\' {
			return ErrInvalidURL
		}
		i += 2
	}
	u, err := url.Parse(input)
	if err != nil || u.Opaque != "" || u.User != nil || u.Host == "" || !u.IsAbs() {
		return ErrInvalidURL
	}
	allowedScheme := u.Scheme == "https" || s.cfg.AllowHTTP && u.Scheme == "http"
	if !allowedScheme {
		return ErrInvalidURL
	}
	host := u.Hostname()
	if host == "" {
		return ErrInvalidURL
	}
	if strings.HasPrefix(u.Host, "[") {
		ip, err := netip.ParseAddr(host)
		if err != nil || !ip.Is6() || ip.Zone() != "" {
			return ErrInvalidURL
		}
		if u.Host != "["+host+"]" && u.Host != "["+host+"]:"+u.Port() {
			return ErrInvalidURL
		}
	} else {
		if strings.ContainsAny(host, ":[]") || !validDNSOrIPv4(host) {
			return ErrInvalidURL
		}
	}
	port := u.Port()
	if strings.HasSuffix(u.Host, ":") {
		return ErrInvalidURL
	}
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
			return ErrInvalidURL
		}
	}
	return nil
}

func validDNSOrIPv4(host string) bool {
	if len(host) > 253 || strings.HasSuffix(host, ".") {
		return false
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Is4() && ip.String() == host
	}
	labels := strings.Split(host, ".")
	// Reject numeric/hex final labels to avoid alternative browser IPv4 interpretations.
	last := strings.ToLower(labels[len(labels)-1])
	numeric := true
	for _, c := range last {
		if c < '0' || c > '9' {
			numeric = false
		}
	}
	if numeric || strings.HasPrefix(last, "0x") {
		return false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			validCharacter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-'
			if !validCharacter {
				return false
			}
		}
	}
	return true
}
