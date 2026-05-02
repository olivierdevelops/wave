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

// Config holds the configuration for the dynamic forward handler.
// It mirrors usecases/dynamic_forward.Config.
type Config struct {
	URLSource             string
	URLKey                string
	AllowedDomains        []string
	BlockPrivateIPs       bool
	IncludeHeaders        [][2]string
	AllowInsecureRequests bool
	Timeout               string
	StripPrefix           string
}

func NewHandler(config *Config) (http.HandlerFunc, error) {
	if config.URLSource == "" {
		config.URLSource = "query"
	}
	if config.URLKey == "" {
		config.URLKey = "url"
	}

	var timeout time.Duration
	if config.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout duration: %w", err)
		}
	}

	allowedMap := make(map[string]bool, len(config.AllowedDomains))
	for _, d := range config.AllowedDomains {
		allowedMap[strings.ToLower(strings.TrimSpace(d))] = true
	}

	return func(w http.ResponseWriter, r *http.Request) {
		targetStr, err := extractTargetURL(config, r)
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

		if config.BlockPrivateIPs {
			host := targetURL.Hostname()
			if isPrivateIP(host) {
				log.Printf("Blocked request to private IP: %s", host)
				http.Error(w, "Access to internal addresses denied", http.StatusForbidden)
				return
			}
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				reqPath := req.URL.Path
				if config.StripPrefix != "" {
					reqPath = strings.TrimPrefix(reqPath, config.StripPrefix)
				}

				finalPath, _ := url.JoinPath(targetURL.Path, reqPath)

				req.URL.Scheme = targetURL.Scheme
				req.URL.Host = targetURL.Host
				req.URL.Path = finalPath
				req.Host = targetURL.Host

				for _, item := range config.IncludeHeaders {
					if len(item) >= 2 {
						req.Header.Set(item[0], item[1])
					}
				}

				log.Printf("Dynamic forward: %s %s -> %s://%s%s", req.Method, req.URL.Path, targetURL.Scheme, targetURL.Host, req.URL.Path)
			},
			Transport: &http.Transport{
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: config.AllowInsecureRequests},
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

func extractTargetURL(config *Config, r *http.Request) (string, error) {
	path := r.URL.Path

	switch config.URLSource {
	case "query":
		target := r.URL.Query().Get(config.URLKey)
		if target == "" {
			return "", fmt.Errorf("missing required query parameter: %s", config.URLKey)
		}
		return target, nil

	case "header":
		target := r.Header.Get(config.URLKey)
		if target == "" {
			return "", fmt.Errorf("missing required header: %s", config.URLKey)
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
		return "", fmt.Errorf("unsupported url_source: %s", config.URLSource)
	}
}

func isPrivateIP(hostname string) bool {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return isPrivateIPLiteral(hostname)
	}
	for _, ip := range ips {
		if isPrivateIPLiteral(ip.String()) {
			return true
		}
	}
	return false
}

func isPrivateIPLiteral(ipStr string) bool {
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
