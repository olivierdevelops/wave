package static_serve

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Dir                string   `yaml:"dir"`
	FileIgnorePatterns []string `yaml:"file_ignore_patterns,omitempty"`
}

// CreateRoute implements servers.RouteConfig.
func (c *Config) CreateRoute(method, path string, data map[string]string) (http.HandlerFunc, error) {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, path)
		for _, pattern := range c.FileIgnorePatterns {
			if strings.Contains(relPath, pattern) {
				http.NotFound(w, r)
				return
			}
		}

		fmt.Println("REL PATH: ", relPath)
		filePath, err := filepath.Abs(filepath.Join(c.Dir, relPath))
		if err != nil {
			fmt.Println("ERRR: ", err.Error())
			http.NotFound(w, r)
			return
		}
		fmt.Println("REL PATH: ", relPath, " => filePath: ", filePath)
		fmt.Println("looking for: ", filePath)

		info, err := os.Stat(filePath)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filePath)
	}, nil
}
