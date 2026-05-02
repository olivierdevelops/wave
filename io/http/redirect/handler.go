package redirect

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Config holds the configuration for the redirect handler.
// It mirrors usecases/redirect.Config.
type Config struct {
	RedirectURL string
	StatusCode  int
}

func NewHandler(config *Config) (http.HandlerFunc, error) {
	rawURL := strings.TrimSpace(config.RedirectURL)
	if rawURL == "" {
		return nil, fmt.Errorf("missing redirect URL")
	}

	target, err := url.Parse(rawURL)
	if err != nil || !target.IsAbs() {
		return nil, fmt.Errorf("invalid redirect URL")
	}

	code := config.StatusCode
	if code == 0 {
		code = http.StatusFound
	}

	if code < 300 || code > 399 {
		return nil, fmt.Errorf("invalid redirect status code: %d. Must match condition 300 > code < 399 ", code)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		u := *target
		u.RawQuery = r.URL.RawQuery
		http.Redirect(w, r, u.String(), code)
	}, nil
}
