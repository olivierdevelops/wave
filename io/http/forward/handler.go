package forward

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Config holds the configuration for the forward handler.
// It mirrors usecases/forward.Config.
type Config struct {
	ForwardURL            string
	IncludeHeaders        [][2]string
	AllowInsecureRequests bool
	Timeout               string
	StripPrefix           string
	URLParams             []string
}

func NewHandler(config *Config, path string) (http.HandlerFunc, error) {
	forwardURL := strings.TrimSpace(config.ForwardURL)
	if forwardURL == "" {
		return nil, fmt.Errorf("missing forward url")
	}

	targetURL, err := url.Parse(strings.TrimSuffix(forwardURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid forward URL: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			prefix := path
			targetURLPath := targetURL.Path
			if targetURLPath == "" {
				targetURLPath = "/"
			}

			log.Printf("targetURLPath='%s' req.URL.Path='%s' prefix='%s' strings.TrimPrefix(req.URL.Path, prefix)='%s'", targetURLPath, req.URL.Path, prefix, strings.TrimPrefix(req.URL.Path, prefix))

			urlPath, _ := url.JoinPath(targetURLPath, strings.TrimPrefix(req.URL.Path, prefix))

			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = urlPath
			req.Host = targetURL.Host

			for _, item := range config.IncludeHeaders {
				if len(item) >= 2 {
					req.Header.Set(item[0], item[1])
				}
			}
			log.Printf("Forwarding %s to: %s%s", req.Method, targetURL.Host, req.URL.Path)
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
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
		FlushInterval: -1,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}, nil
}

// flushingWriter is an auto-flushing writer.
type flushingWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func (fw *flushingWriter) Write(p []byte) (n int, err error) {
	n, err = fw.writer.Write(p)
	if err == nil {
		fw.flusher.Flush()
	}
	return n, err
}

// isClientClosed detects client disconnect errors.
func isClientClosed(err error) bool {
	if err == nil {
		return false
	}
	str := err.Error()
	return strings.Contains(str, "broken pipe") ||
		strings.Contains(str, "connection reset") ||
		strings.Contains(str, "request canceled") ||
		err == context.Canceled
}
