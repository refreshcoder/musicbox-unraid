package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type StaticDir string

func (d StaticDir) Handler() http.Handler {
	root := string(d)
	fs := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
			http.NotFound(w, r)
			return
		}

		p := path.Clean(r.URL.Path)
		if p == "/" || strings.HasPrefix(p, "/assets/") {
			fs.ServeHTTP(w, r)
			return
		}

		target := filepath.Join(root, strings.TrimPrefix(p, "/"))
		if st, err := os.Stat(target); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fs.ServeHTTP(w, r2)
	})
}

