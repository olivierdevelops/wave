package auth_logout

import (
	"encoding/json"
	htmltemplate "html/template"
	"log"
	"net/http"
	"strings"
)

type Config struct {
	For               string `yaml:"for,omitempty"`
	RedirectOnSuccess string `yaml:"redirect_on_success,omitempty"`
	RedirectOnFailure string `yaml:"redirect_on_failure,omitempty"`

	ErrorTemplate     string `yaml:"error_template,omitempty"`
	ErrorTemplateStr  string `yaml:"error_template_str,omitempty"`
	ErrorRedirect     string `yaml:"error_redirect,omitempty"`
	ErrorResponseType string `yaml:"error_response_type,omitempty"`

	CookieSecure   *bool  `yaml:"cookie_secure,omitempty"`
	CookieSameSite string `yaml:"cookie_same_site,omitempty"`
}

// LogoutResponse is the result of a logout attempt.
type LogoutResponse struct {
	Success    bool
	Location   string
	Name       string
	Value      string
	Message    string
	Error      string
	Code       string
	RedirectTo string
}

// LogoutFn is a function that performs logout.
// Inject this before using Config.CreateRoute.
var LogoutFn func(r *http.Request, authConfigName string) *LogoutResponse

// LogoutErrorContext is the data passed to error templates.
type LogoutErrorContext struct {
	Success bool          `json:"success"`
	Error   string        `json:"error"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Request *http.Request `json:"-"`
}

// CreateRoute implements servers.RouteConfig.
func (c *Config) CreateRoute(method, path string, data map[string]string) (http.HandlerFunc, error) {
	var errorTemplate *htmltemplate.Template
	var templateErr error

	if c.ErrorTemplate != "" {
		errorTemplate, templateErr = htmltemplate.ParseFiles(c.ErrorTemplate)
		if templateErr != nil {
			log.Printf("[WARN] Failed to parse error template file %s: %v", c.ErrorTemplate, templateErr)
		}
	} else if c.ErrorTemplateStr != "" {
		errorTemplate, templateErr = htmltemplate.New("error").Parse(c.ErrorTemplateStr)
		if templateErr != nil {
			log.Printf("[WARN] Failed to parse error template string: %v", templateErr)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var response *LogoutResponse
		if LogoutFn != nil {
			response = LogoutFn(r, c.For)
		} else {
			response = &LogoutResponse{
				Success: false,
				Error:   "logout not configured",
				Code:    "not_configured",
			}
		}

		if !response.Success {
			log.Printf("[LOGOUT ERROR]: %s (code: %s)", response.Error, response.Code)
			c.handleError(w, r, response, errorTemplate)
			return
		}

		log.Printf("[LOGOUT SUCCESS]: %s", response.Message)

		switch response.Location {
		case "cookie":
			c.clearCookie(w, r, response)
			c.redirectOnSuccess(w, r, response)

		case "header":
			w.Header().Set(response.Name, "")
			c.sendJSON(w, response)

		default:
			http.Error(w, "Unexpected error: invalid token location", http.StatusInternalServerError)
		}
	}, nil
}

func (c *Config) handleError(w http.ResponseWriter, r *http.Request, response *LogoutResponse, errorTemplate *htmltemplate.Template) {
	ctx := LogoutErrorContext{
		Success: false,
		Error:   response.Error,
		Code:    response.Code,
		Message: response.Message,
		Request: r,
	}

	responseType := c.ErrorResponseType
	if responseType == "" {
		if isBrowserRequest(r) {
			responseType = "redirect"
		} else {
			responseType = "json"
		}
	}

	switch responseType {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ctx)

	case "html":
		if errorTemplate != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			if err := errorTemplate.Execute(w, ctx); err != nil {
				log.Printf("[ERROR] Failed to execute error template: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		} else {
			c.renderBasicErrorHTML(w, ctx)
		}

	case "redirect":
		fallthrough
	default:
		redirectURL := c.ErrorRedirect
		if redirectURL == "" {
			redirectURL = c.RedirectOnFailure
		}
		if redirectURL == "" {
			redirectURL = r.Referer()
		}
		if redirectURL == "" {
			redirectURL = "/"
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func (c *Config) clearCookie(w http.ResponseWriter, r *http.Request, response *LogoutResponse) {
	secure := isSecureRequest(r)
	if c.CookieSecure != nil {
		secure = *c.CookieSecure
	}

	sameSite := parseSameSite(c.CookieSameSite, secure)

	cookie := &http.Cookie{
		Name:     response.Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	}

	http.SetCookie(w, cookie)
}

func (c *Config) redirectOnSuccess(w http.ResponseWriter, r *http.Request, response *LogoutResponse) {
	redirectURL := c.RedirectOnSuccess
	if redirectURL == "" && response.RedirectTo != "" {
		redirectURL = response.RedirectTo
	}
	if redirectURL == "" {
		redirectURL = "/"
	}

	log.Printf("[LOGOUT] Redirecting to: %s", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (c *Config) sendJSON(w http.ResponseWriter, response *LogoutResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Config) renderBasicErrorHTML(w http.ResponseWriter, ctx LogoutErrorContext) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Logout Error</title>
    <style>
        body { font-family: sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .error { background: #fee; border: 1px solid #fcc; border-radius: 4px; padding: 15px; margin: 20px 0; }
        .error-title { color: #c33; font-weight: bold; margin-bottom: 10px; }
        .error-message { color: #666; }
        .error-code { color: #999; font-size: 0.9em; margin-top: 10px; }
        .back-link { margin-top: 20px; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="error">
        <div class="error-title">Logout Failed</div>
        <div class="error-message">` + htmltemplate.HTMLEscapeString(ctx.Error) + `</div>
        <div class="error-code">Error Code: ` + htmltemplate.HTMLEscapeString(ctx.Code) + `</div>
    </div>
    <div class="back-link">
        <a href="javascript:history.back()">&#8592; Go Back</a>
    </div>
</body>
</html>`

	w.Write([]byte(html))
}

func isBrowserRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	userAgent := r.Header.Get("User-Agent")
	return strings.Contains(accept, "text/html") || strings.Contains(userAgent, "Mozilla")
}

func isSecureRequest(r *http.Request) bool {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return true
	}
	if r.TLS != nil {
		return true
	}
	if r.URL.Scheme == "https" {
		return true
	}
	return false
}

func parseSameSite(value string, secure bool) http.SameSite {
	switch value {
	case "None":
		return http.SameSiteNoneMode
	case "Lax", "":
		return http.SameSiteLaxMode
	case "Strict":
		return http.SameSiteStrictMode
	default:
		if secure {
			return http.SameSiteLaxMode
		}
		return http.SameSiteDefaultMode
	}
}
