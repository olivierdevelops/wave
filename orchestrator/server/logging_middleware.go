package servers

import (
	infrahttp "easyserver/infra/http"
	"net/http"
)

// loggingMiddleware wraps the infra implementation so existing callers in this
// package continue to compile without change.
func loggingMiddleware(next http.Handler) http.Handler {
	return infrahttp.LoggingMiddleware(next)
}
