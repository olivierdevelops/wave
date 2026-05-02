package dynamic_forward

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	URLSource             string      `yaml:"url_source,omitempty" json:"url_source,omitempty"`
	URLKey                string      `yaml:"url_key,omitempty" json:"url_key,omitempty"`
	AllowedDomains        []string    `yaml:"allowed_domains,omitempty" json:"allowed_domains,omitempty"`
	BlockPrivateIPs       bool        `yaml:"block_private_ips,omitempty" json:"block_private_ips,omitempty"`
	IncludeHeaders        [][2]string `yaml:"include_headers,omitempty" json:"include_headers,omitempty"`
	AllowInsecureRequests bool        `yaml:"allow_insecure_requests,omitempty" json:"allow_insecure_requests,omitempty"`
	Timeout               string      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	StripPrefix           string      `yaml:"strip_prefix,omitempty" json:"strip_prefix,omitempty"`
}

// CreateRoute implements dynamic forwarding with SSRF protection.
func (c *Config) CreateRoute(method, path string, data map[string]string) (http.HandlerFunc, error) {
	if c.URLSource == "" {
		c.URLSource = "query"
	}
	if c.URLKey == "" {
		c.URLKey = "url"
	}

	var timeout time.Duration
	if c.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(c.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout duration: %w", err)
		}
	}

	allowedMap := make(map[string]bool, len(c.AllowedDomains))
	for _, d := range c.AllowedDomains {
		allowedMap[strings.ToLower(strings.TrimSpace(d))] = true
	}

	return func(w http.ResponseWriter, r *http.Request) {
		targetStr, err := c.extractTargetURL(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		targetURL, err := url.Parse(targetStr)
		if err != nil {
			log.Printf("Invalid target URL '%s': %v", targetStr, err)
			http.Error(w, "Invalid target URL", http.StatusBadRequest)
			return
		}

		if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
			http.Error(w, "Only http/https schemes allowed", http.StatusBadRequest)
			return
		}

		if len(allowedMap) > 0 {
			host := strings.ToLower(targetURL.Hostname())
			if !allowedMap[host] {
				log.Printf("Blocked request to disallowed domain: %s", host)
				http.Error(w, "Domain not allowed", http.StatusForbidden)
				return
			}
		}

		if c.BlockPrivateIPs {
			host := targetURL.Hostname()
			if c.isPrivateIP(host) {
				log.Printf("Blocked request to private IP: %s", host)
				http.Error(w, "Access to internal addresses denied", http.StatusForbidden)
				return
			}
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				reqPath := req.URL.Path
				if c.StripPrefix != "" {
					reqPath = strings.TrimPrefix(reqPath, c.StripPrefix)
				}

				finalPath, _ := url.JoinPath(targetURL.Path, reqPath)

				req.URL.Scheme = targetURL.Scheme
				req.URL.Host = targetURL.Host
				req.URL.Path = finalPath
				req.Host = targetURL.Host

				for _, item := range c.IncludeHeaders {
					if len(item) >= 2 {
						req.Header.Set(item[0], item[1])
					}
				}

				log.Printf("Dynamic forward: %s %s -> %s://%s%s", req.Method, req.URL.Path, targetURL.Scheme, targetURL.Host, req.URL.Path)
			},
			Transport: &http.Transport{
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: c.AllowInsecureRequests},
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				ForceAttemptHTTP2:  true,
				DisableCompression: true,
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				if r.Context().Err() != nil {
					return
				}
				log.Printf("Dynamic proxy error: %v", err)
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
			},
			FlushInterval: -1,
		}

		if timeout > 0 {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			r = r.WithContext(ctx)
		}

		proxy.ServeHTTP(w, r)
	}, nil
}

func (c *Config) extractTargetURL(r *http.Request) (string, error) {
	path := r.URL.Path

	switch c.URLSource {
	case "query":
		target := r.URL.Query().Get(c.URLKey)
		if target == "" {
			return "", fmt.Errorf("missing required query parameter: %s", c.URLKey)
		}
		return target, nil

	case "header":
		target := r.Header.Get(c.URLKey)
		if target == "" {
			return "", fmt.Errorf("missing required header: %s", c.URLKey)
		}
		return target, nil

	case "path":
		prefix := strings.TrimSuffix(path, "/*")
		if prefix == "" {
			prefix = path
		}
		target := strings.TrimPrefix(r.URL.Path, prefix)
		target = strings.TrimPrefix(target, "/")
		if target == "" {
			return "", fmt.Errorf("missing target URL in path")
		}
		return target, nil

	default:
		return "", fmt.Errorf("unsupported url_source: %s", c.URLSource)
	}
}

func (c *Config) isPrivateIP(hostname string) bool {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return c.isPrivateIPLiteral(hostname)
	}
	for _, ip := range ips {
		if c.isPrivateIPLiteral(ip.String()) {
			return true
		}
	}
	return false
}

func (c *Config) isPrivateIPLiteral(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
		ipInCIDR(ip, "10.0.0.0/8") || ipInCIDR(ip, "172.16.0.0/12") ||
		ipInCIDR(ip, "192.168.0.0/16") || ipInCIDR(ip, "127.0.0.0/8") ||
		ipInCIDR(ip, "169.254.0.0/16") || ipInCIDR(ip, "224.0.0.0/4")
}

func ipInCIDR(ip net.IP, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return network.Contains(ip)
}
