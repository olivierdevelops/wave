package redirect

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	RedirectURL string `yaml:"redirect_url,omitempty" json:"redirect_url"`
	StatusCode  int    `yaml:"status_code,omitempty" json:"status_code"`
}

// CreateRoute implements servers.RouteConfig.
func (c *Config) CreateRoute(method, path string, data map[string]string) (http.HandlerFunc, error) {
	rawURL := strings.TrimSpace(c.RedirectURL)
	if rawURL == "" {
		return nil, fmt.Errorf("missing redirect URL")
	}

	target, err := url.Parse(rawURL)
	if err != nil || !target.IsAbs() {
		return nil, fmt.Errorf("invalid redirect URL")
	}

	code := c.StatusCode
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
