package legal

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const KremlinBase = "http://kremlin.ru"

// ErrNoText — закон не найден на kremlin.ru или у страницы нет текста.
var ErrNoText = errors.New("текст закона не найден")

type Kremlin struct {
	HTTP *http.Client
	Base string
	// Delay — пауза перед каждым запросом (вежливость к сайту; 0 в тестах).
	Delay time.Duration
}

func NewKremlin() *Kremlin {
	return &Kremlin{HTTP: &http.Client{Timeout: 40 * time.Second}, Base: KremlinBase, Delay: 1500 * time.Millisecond}
}

var (
	resultRe = regexp.MustCompile(`(?s)<a href="(/acts/bank/\d+)"[^>]*>\s*Федеральный закон от (\d{2}\.\d{2}\.\d{4})[^№]*№\s*([0-9]+-ФЗ)`)
	bodyRe   = regexp.MustCompile(`(?s)itemprop="articleBody"[^>]*>(.*?)</div>\s*</div>\s*</div>`)
	tagRe    = regexp.MustCompile(`(?s)<script.*?</script>|<style.*?</style>|<[^>]+>`)
	spaceRe  = regexp.MustCompile(`\s+`)
)

// maxSearchPages — сколько страниц выдачи (по 20 законов) просматриваем за один день подписания.
const maxSearchPages = 8

// Text ищет закон на kremlin.ru среди законов, подписанных в тот же день, и возвращает страницу и очищенный текст.
func (k *Kremlin) Text(ctx context.Context, d Doc) (pageURL, text string, err error) {
	day := d.Signed.Format("02.01.2006")
	path := ""
	for page := 1; page <= maxSearchPages && path == ""; page++ {
		q := url.Values{"date_since": {day}, "date_till": {day}, "type": {"5"}, "page": {fmt.Sprint(page)}}
		search, err := k.get(ctx, k.Base+"/acts/bank/search?"+q.Encode())
		if err != nil {
			return "", "", err
		}
		if path = findResult(search, d); path == "" && !hasResults(search) {
			break
		}
	}
	if path == "" {
		return "", "", fmt.Errorf("%w: %s от %s", ErrNoText, d.Number, day)
	}
	page, err := k.get(ctx, k.Base+path)
	if err != nil {
		return "", "", err
	}
	text = ExtractBody(page)
	if len([]rune(text)) < 200 {
		return "", "", fmt.Errorf("%w: пустая страница %s", ErrNoText, path)
	}
	return k.Base + path, text, nil
}

func hasResults(searchHTML string) bool { return strings.Contains(searchHTML, `href="/acts/bank/`) }

// findResult выбирает в выдаче поиска запись строго с тем же номером и датой.
func findResult(searchHTML string, d Doc) string {
	want := d.Signed.Format("02.01.2006")
	// На странице между словами стоят неразрывные пробелы, которые \s не считает пробелами.
	searchHTML = strings.ReplaceAll(html.UnescapeString(searchHTML), "\u00a0", " ")
	for _, m := range resultRe.FindAllStringSubmatch(searchHTML, -1) {
		if m[2] == want && m[3] == d.Number {
			return m[1]
		}
	}
	return ""
}

// ExtractBody достаёт из страницы текст документа без разметки.
func ExtractBody(pageHTML string) string {
	m := bodyRe.FindStringSubmatch(pageHTML)
	if m == nil {
		return ""
	}
	return Normalize(tagRe.ReplaceAllString(m[1], " "))
}

// Normalize сводит пробелы (в том числе неразрывные) к одному; по нему же проверяются цитаты.
func Normalize(s string) string {
	s = html.UnescapeString(s)
	s = strings.NewReplacer(" ", " ", " ", " ", "«", `"`, "»", `"`, "“", `"`, "”", `"`, "—", "-", "–", "-").Replace(s)
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

// get делает запрос с паузой; при 403/429 (ограничение частоты) ждёт и повторяет до трёх раз.
func (k *Kremlin) get(ctx context.Context, u string) (string, error) {
	wait := k.Delay
	var lastErr error
	for try := 0; try < 3; try++ {
		if err := sleep(ctx, wait); err != nil {
			return "", err
		}
		wait = 4 * (wait + time.Second) // 403 обычно проходит после паузы
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ShtilLawBot/1.0)")
		resp, err := k.HTTP.Do(req)
		if err != nil {
			return "", err
		}
		b, rerr := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
		resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusOK:
			return string(b), rerr
		case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
			lastErr = fmt.Errorf("kremlin.ru: HTTP %d", resp.StatusCode)
		default:
			return "", fmt.Errorf("kremlin.ru: HTTP %d", resp.StatusCode)
		}
	}
	return "", lastErr
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
