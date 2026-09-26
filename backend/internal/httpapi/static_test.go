package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func webServer(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>Штиль</html>"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "sw.js"), []byte("const BUILD = '__BUILD__'; const ASSETS = JSON.parse('__ASSETS__');"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "manifest.webmanifest"), []byte("{}"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "js"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "js", "app.js"), []byte("export {}"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "test"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "test", "core.test.js"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "dev-server.mjs"), []byte("x"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "fonts"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "fonts", "a.ttf"), []byte("font"), 0o644))
	return NewServer(&fakeStore{}).WithWeb(dir).Handler()
}

func TestWebServesIndexAndAssetsWithSecurityHeaders(t *testing.T) {
	h := webServer(t)
	root := get(h, "/")
	if root.Code != 200 || !strings.Contains(root.Body.String(), "Штиль") {
		t.Fatalf("/: %d %q", root.Code, root.Body.String())
	}
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "X-Frame-Options": "DENY", "Cache-Control": "no-cache",
	} {
		if got := root.Header().Get(header); got != want {
			t.Errorf("%s: %q, ждали %q", header, got, want)
		}
	}
	if csp := root.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP: %q", csp)
	}
	if ct := get(h, "/js/app.js").Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Errorf("js: %q", ct)
	}
	if ct := get(h, "/manifest.webmanifest").Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/manifest+json") {
		t.Errorf("manifest: %q", ct)
	}
	if cc := get(h, "/fonts/a.ttf").Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=86400") {
		t.Errorf("шрифты должны кэшироваться: %q", cc)
	}
}

func TestWebDoesNotListDirectoriesOrEscapeTheRoot(t *testing.T) {
	h := webServer(t)
	for _, p := range []string{"/js/", "/fonts/", "/test/core.test.js", "/dev-server.mjs", "/nope.html", "/../etc/passwd", "/..%2f..%2fetc/passwd"} {
		if code := get(h, p).Code; code == 200 {
			t.Errorf("%s: неожиданно 200", p)
		}
	}
}

func TestWebDoesNotShadowAPIAndIsOptional(t *testing.T) {
	h := webServer(t)
	if code := get(h, "/v1/health").Code; code != 200 {
		t.Fatalf("/v1/health: %d", code)
	}
	if code := get(h, "/v1/unknown").Code; code != 404 {
		t.Fatalf("неизвестный маршрут API должен быть 404, а не страницей: %d", code)
	}
	api := NewServer(&fakeStore{}).WithWeb("").Handler()
	if code := get(api, "/").Code; code != 404 {
		t.Fatalf("без веба корень — 404: %d", code)
	}
	if code := get(NewServer(&fakeStore{}).WithWeb("/нет/такого/каталога").Handler(), "/").Code; code != 404 {
		t.Fatalf("несуществующий каталог отключает веб: %d", code)
	}
}

func TestServiceWorkerGetsVersionAndAssetList(t *testing.T) {
	h := webServer(t)
	rec := get(h, "/sw.js")
	body := rec.Body.String()
	if rec.Code != 200 || strings.Contains(body, "__BUILD__") || strings.Contains(body, "__ASSETS__") {
		t.Fatalf("метки не подставлены: %d %q", rec.Code, body)
	}
	for _, want := range []string{`"/index.html"`, `"/js/app.js"`, `"/fonts/a.ttf"`, `"/manifest.webmanifest"`} {
		if !strings.Contains(body, want) {
			t.Errorf("в списке нет %s: %s", want, body)
		}
	}
	for _, bad := range []string{"test/core.test.js", "dev-server.mjs", `"/sw.js"`} {
		if strings.Contains(body, bad) {
			t.Errorf("служебный файл в списке: %s", bad)
		}
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("sw.js должен всегда сверяться с сервером: %q", cc)
	}
	if rec.Header().Get("Service-Worker-Allowed") != "/" {
		t.Errorf("нет Service-Worker-Allowed")
	}
}

func TestServiceWorkerVersionChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "v1")
	write("sw.js", "const BUILD = '__BUILD__'; const ASSETS = JSON.parse('__ASSETS__');")
	first := string(buildServiceWorker(dir))
	write("index.html", "v2")
	second := string(buildServiceWorker(dir))
	if first == second {
		t.Fatal("версия воркера должна меняться вместе с файлами")
	}
	if again := string(buildServiceWorker(dir)); again != second {
		t.Fatal("версия должна быть стабильной для одного и того же содержимого")
	}
}
