package serve_file

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GetUserFn is a function that retrieves the current user from the request context.
type GetUserFn func(r *http.Request) interface{}

// SSRScriptPath optionally injects a script tag before </body> in HTML responses.
var SSRScriptPath = ""

// Config holds the configuration for the serve file handler.
// It mirrors usecases/serve_file.Config.
type Config struct {
	CatchAll   bool
	FilePath   string
	IsTemplate bool
}

func NewHandler(config *Config, path string, getUser GetUserFn) (http.HandlerFunc, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("missing 'filepath' for '%s': ", config.FilePath)
	}

	config.FilePath, _ = filepath.Abs(config.FilePath)

	fmt.Println("looking for: ", config.FilePath)

	var tmpl *template.Template
	var err error
	if config.IsTemplate {
		tmpl, err = template.ParseFiles(config.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template '%s': %w", config.FilePath, err)
		}
	}

	if config.CatchAll {
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		return func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("looking for: ", config.FilePath)

			if config.IsTemplate {
				serveTemplateWithData(w, r, tmpl, nil, getUser)
			} else {
				serveFile(w, r, config.FilePath, false)
			}
		}, nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("looking for: ", config.FilePath)

		if path == "/" && r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if config.IsTemplate {
			serveTemplateWithData(w, r, tmpl, nil, getUser)
		} else {
			serveFile(w, r, config.FilePath, false)
		}
	}, nil
}

func serveFile(w http.ResponseWriter, r *http.Request, path string, isTemplate bool) {
	if isTemplate {
		serveTemplateFile(w, r, path, nil)
	} else if strings.HasSuffix(strings.ToLower(path), ".html") {
		serveHTMLWithScriptForSSE(w, r, path)
	} else {
		http.ServeFile(w, r, path)
	}
}

func serveTemplateFile(w http.ResponseWriter, r *http.Request, filePath string, getUser GetUserFn) {
	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	serveTemplateWithData(w, r, tmpl, nil, getUser)
}

func serveTemplateWithData(w http.ResponseWriter, r *http.Request, tmpl *template.Template, data map[string]string, getUser GetUserFn) {
	templateData := buildTemplateData(r, data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	funcMap := template.FuncMap{}
	if getUser != nil {
		fn := getUser
		funcMap["GetUser"] = func() interface{} {
			return fn(r)
		}
	}

	var buf strings.Builder
	if err := tmpl.Funcs(funcMap).Execute(&buf, templateData); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}

	htmlContent := buf.String()

	if SSRScriptPath != "" && strings.Contains(strings.ToLower(htmlContent), "</body>") {
		scriptTag := fmt.Sprintf(`<script src="%s"></script>\n`, SSRScriptPath)
		htmlContent = strings.Replace(htmlContent, "</body>", scriptTag+"</body>", 1)
	}

	w.Write([]byte(htmlContent))
}

func buildTemplateData(r *http.Request, data map[string]string) map[string]interface{} {
	templateData := make(map[string]interface{})

	if data != nil {
		for k, v := range data {
			templateData[k] = v
		}
	}

	templateData["Request"] = map[string]interface{}{
		"Method":     r.Method,
		"URL":        r.URL.String(),
		"Path":       r.URL.Path,
		"Host":       r.Host,
		"RemoteAddr": r.RemoteAddr,
		"UserAgent":  r.UserAgent(),
	}

	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	templateData["Query"] = query

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	templateData["Headers"] = headers

	return templateData
}

func serveHTMLWithScriptForSSE(w http.ResponseWriter, r *http.Request, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	htmlContent := string(content)

	if SSRScriptPath != "" && strings.Contains(strings.ToLower(htmlContent), "</body>") {
		scriptTag := fmt.Sprintf(`<script src="%s"></script>\n`, SSRScriptPath)
		htmlContent = strings.Replace(htmlContent, "</body>", scriptTag+"</body>", 1)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	http.ServeContent(w, r, filepath.Base(filePath), getModTime(filePath), strings.NewReader(htmlContent))
}

func getModTime(filePath string) time.Time {
	if info, err := os.Stat(filePath); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}
