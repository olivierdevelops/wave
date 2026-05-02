package serve_content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds the configuration for the serve content handler.
// It mirrors usecases/serve_content.Config.
type Config struct {
	StatusCode          *int
	Headers             [][2]string
	Body                string
	PrintRequest        bool
	TimeoutMilliseconds int
}

func NewHandler(config *Config) (http.HandlerFunc, error) {
	if config.Body == "" {
		return nil, fmt.Errorf("missing body")
	}

	statusCode := 200
	if config.StatusCode != nil {
		statusCode = *config.StatusCode
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if config.PrintRequest {
			printRequest(r)
		}

		if config.TimeoutMilliseconds > 0 {
			time.Sleep(time.Duration(config.TimeoutMilliseconds) * time.Millisecond)
		}

		for _, item := range config.Headers {
			w.Header().Add(item[0], item[1])
		}
		w.WriteHeader(statusCode)
		w.Write([]byte(config.Body))
	}, nil
}

func printRequest(r *http.Request) {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔════════════════════════════════════════════════════════════════\n")
	sb.WriteString("║ REQUEST DETAILS\n")
	sb.WriteString("╚════════════════════════════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Method: %s\n", r.Method))
	sb.WriteString(fmt.Sprintf("URL:    %s\n", r.URL.String()))
	sb.WriteString(fmt.Sprintf("Path:   %s\n", r.URL.Path))
	sb.WriteString(fmt.Sprintf("Host:   %s\n", r.Host))
	sb.WriteString(fmt.Sprintf("Remote: %s\n", r.RemoteAddr))
	sb.WriteString("\n")

	if len(r.Header) > 0 {
		sb.WriteString("┌─ HEADERS ─────────────────────────────────────────────────────\n")
		for key, values := range r.Header {
			for _, value := range values {
				sb.WriteString(fmt.Sprintf("│ %s: %s\n", key, value))
			}
		}
		sb.WriteString("└───────────────────────────────────────────────────────────────\n\n")
	}

	if len(r.URL.Query()) > 0 {
		sb.WriteString("┌─ QUERY PARAMETERS ────────────────────────────────────────────\n")
		for key, values := range r.URL.Query() {
			for _, value := range values {
				sb.WriteString(fmt.Sprintf("│ %s = %s\n", key, value))
			}
		}
		sb.WriteString("└───────────────────────────────────────────────────────────────\n\n")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error reading body: %v\n", err))
	} else {
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		if len(body) > 0 {
			sb.WriteString("┌─ BODY ────────────────────────────────────────────────────────\n")

			contentType := r.Header.Get("Content-Type")
			isJSON := strings.Contains(strings.ToLower(contentType), "application/json") ||
				(len(body) > 0 && body[0] == '{') ||
				(len(body) > 0 && body[0] == '[')

			if isJSON {
				var prettyJSON bytes.Buffer
				if err := json.Indent(&prettyJSON, body, "│ ", "  "); err != nil {
					sb.WriteString(fmt.Sprintf("│ %s\n", string(body)))
				} else {
					sb.WriteString(prettyJSON.String())
					sb.WriteString("\n")
				}
			} else {
				lines := strings.Split(string(body), "\n")
				for _, line := range lines {
					sb.WriteString(fmt.Sprintf("│ %s\n", line))
				}
			}

			sb.WriteString("└───────────────────────────────────────────────────────────────\n")
		} else {
			sb.WriteString("(empty body)\n")
		}
	}

	sb.WriteString("\n")

	fmt.Print(sb.String())
}
