package static_serve

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the configuration for the static serve handler.
// It mirrors usecases/static_serve.Config.
type Config struct {
	Dir                string
	FileIgnorePatterns []string
}

func NewHandler(config *Config, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, path)
		for _, pattern := range config.FileIgnorePatterns {
			if strings.Contains(relPath, pattern) {
				http.NotFound(w, r)
				return
			}
		}

		fmt.Println("REL PATH: ", relPath)
		filePath, err := filepath.Abs(filepath.Join(config.Dir, relPath))
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
	}
}
