package serve_dependencies

import (
	"embed"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed dependencies/*
var dependenciesDir embed.FS

type Config struct{}

// CreateRoute implements servers.RouteConfig.
func (c *Config) CreateRoute(method, path string, data map[string]string) (http.HandlerFunc, error) {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, path)

		filePath := filepath.Join("dependencies", relPath)
		fmt.Println("LOOKING FOR: ", filePath)

		f, err := dependenciesDir.Open(filePath)
		if err != nil {
			fmt.Println("ERROR: ", err.Error())
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			http.NotFound(w, r)
			return
		}

		ext := filepath.Ext(filePath)
		contentType := mime.TypeByExtension(ext)
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		fmt.Printf("filePath='%s'\ncontentType='%s'\n\n\n\n", filePath, contentType)

		w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
		io.Copy(w, f)
	}, nil
}

func (c *Config) Validate() error {
	return nil
}
