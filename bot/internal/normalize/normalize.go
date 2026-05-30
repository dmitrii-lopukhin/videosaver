package normalize

import (
	"net/url"
	"strings"
)

// Normalize strips query params, fragments, trailing slashes, and lowercases
// the scheme and host. The path is preserved as-is.
func Normalize(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""
	u.Fragment = ""
	result := strings.TrimRight(u.String(), "/")
	return result, nil
}
