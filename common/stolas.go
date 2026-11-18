package common

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"
)

// StolasCanvasConfiguration enables credential delegation only for a configured
// HTTPS canvas origin and a sufficiently long server-to-server secret.
func StolasCanvasConfiguration() (origin, secret string, enabled bool) {
	origin = strings.TrimRight(strings.TrimSpace(os.Getenv("STOLAS_CANVAS_URL")), "/")
	secret = strings.TrimSpace(os.Getenv("STOLAS_CANVAS_CLIENT_SECRET"))
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Path != "" || strings.ContainsAny(origin, "?#") || utf8.RuneCountInString(secret) < 32 {
		return "", "", false
	}
	return origin, secret, true
}

// IsStolasCanvasClient authenticates a server request without trusting client IP
// or user-supplied forwarding headers. Incomplete configuration never opts in.
func IsStolasCanvasClient(provided string) bool {
	_, secret, enabled := StolasCanvasConfiguration()
	if !enabled || provided == "" {
		return false
	}
	providedHash, expectedHash := sha256.Sum256([]byte(provided)), sha256.Sum256([]byte(secret))
	return subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1
}
