package validation

import "regexp"

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidEmail returns true if the string looks like an email address
func ValidEmail(e string) bool {
	return emailRe.MatchString(e)
}

// ValidPort returns true if p is a valid TCP/UDP port
func ValidPort(p int) bool {
	return p > 0 && p <= 65535
}

// ValidProxyType validates supported proxy types
func ValidProxyType(t string) bool {
	switch t {
	case "tcp", "http", "https", "udp", "stcp", "xtcp", "tcpmux":
		return true
	default:
		return false
	}
}
