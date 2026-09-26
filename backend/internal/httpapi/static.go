package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func init() {
	// В образе distroless нет системной таблицы типов: регистрируем нужные явно.
	for ext, typ := range map[string]string{
		".webmanifest": "application/manifest+json", ".ttf": "font/ttf", ".js": "text/javascript; charset=utf-8",
		".css": "text/css; charset=utf-8", ".html": "text/html; charset=utf-8", ".png": "image/png", ".svg": "image/svg+xml",
	} {
		_ = mime.AddExtensionType(ext, typ)
	}
}

// WithWeb включает раздачу веб-приложения (PWA) из каталога dir на тех же адресах, где нет маршрутов API.
// Пустой dir или отсутствующий каталог оставляют только API.
func (s *Server) WithWeb(dir string) *Server {
	if dir == "" {
		return s
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return s
	}
	s.webDir = dir
	s.swBody = buildServiceWorker(dir)
	return s
}

// skipInAssets — служебные файлы, которые не входят в кэш оболочки и не влияют на версию.
func skipInAssets(rel string) bool {
	base := path.Base(rel)
	return rel == "sw.js" || strings.HasPrefix(rel, "test/") || strings.HasPrefix(rel, "e2e/") || strings.HasPrefix(rel, "tools/") ||
		strings.HasPrefix(base, ".") || strings.HasSuffix(base, ".mjs") || strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".txt") ||
		strings.HasSuffix(base, ".map")
}

// buildServiceWorker подставляет в sw.js версию (хэш содержимого файлов) и список файлов оболочки.
// Код изменился — версия другая — браузер ставит новый воркер и обновляет кэш; ничего вручную поднимать не нужно.
func buildServiceWorker(dir string) []byte {
	tpl, err := os.ReadFile(filepath.Join(dir, "sw.js"))
	if err != nil {
		return nil
	}
	var assets []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if !skipInAssets(rel) {
			assets = append(assets, "/"+rel)
		}
		return nil
	})
	sort.Strings(assets)
	h := sha256.New()
	for _, a := range assets {
		body, _ := os.ReadFile(filepath.Join(dir, strings.TrimPrefix(a, "/")))
		h.Write([]byte(a))
		h.Write(body)
	}
	h.Write(tpl)
	list, _ := json.Marshal(append([]string{"/"}, assets...))
	out := strings.ReplaceAll(string(tpl), "__BUILD__", hex.EncodeToString(h.Sum(nil))[:12])
	return []byte(strings.ReplaceAll(out, "__ASSETS__", string(list)))
}

// noListing не даёт FileServer показывать содержимое каталогов: отдаётся только index.html.
type noListing struct{ fs http.FileSystem }

func (n noListing) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if st.IsDir() {
		index, err := n.fs.Open(path.Join(name, "index.html"))
		if err != nil {
			_ = f.Close()
			return nil, fs.ErrNotExist
		}
		_ = index.Close()
	}
	return f, nil
}

const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
	"font-src 'self'; connect-src 'self'; manifest-src 'self'; worker-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func (s *Server) web() http.Handler {
	files := http.FileServer(noListing{http.Dir(s.webDir)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Служебные файлы разработки (тесты, локальный сервер) наружу не отдаём.
		if strings.HasPrefix(r.URL.Path, "/test") || strings.HasSuffix(r.URL.Path, ".mjs") || strings.HasPrefix(r.URL.Path, "/.") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/sw.js" && s.swBody != nil {
			h := w.Header()
			h.Set("Content-Type", "text/javascript; charset=utf-8")
			h.Set("Cache-Control", "no-cache")
			h.Set("Service-Worker-Allowed", "/")
			h.Set("X-Content-Type-Options", "nosniff")
			_, _ = w.Write(s.swBody)
			return
		}
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=()")
		switch strings.ToLower(path.Ext(r.URL.Path)) {
		case ".ttf", ".woff2", ".png":
			h.Set("Cache-Control", "public, max-age=86400")
		default:
			h.Set("Cache-Control", "no-cache") // всегда сверяем с сервером (ETag), чтобы обновления доходили сразу
		}
		files.ServeHTTP(w, r)
	})
}
