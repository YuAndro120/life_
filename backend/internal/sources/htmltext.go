package sources

import (
	"io"
	"strings"

	xhtml "golang.org/x/net/html"
)

// HTMLToText превращает фрагмент HTML в обычный текст: теги убираются, <br> и блоки дают переносы,
// сущности раскрываются, пробелы схлопываются. Скрипты и стили отбрасываются.
func HTMLToText(fragment string) string {
	z := xhtml.NewTokenizer(strings.NewReader(fragment))
	var b strings.Builder
	skip := 0
	for {
		switch z.Next() {
		case xhtml.ErrorToken:
			if z.Err() != io.EOF {
				// Битую разметку не роняем: берём то, что успели разобрать.
			}
			return normalizeSpace(b.String())
		case xhtml.TextToken:
			if skip == 0 {
				b.WriteString(string(z.Text()))
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "script", "style":
				skip++
			case "br":
				b.WriteString("\n")
			}
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			switch string(name) {
			case "script", "style":
				if skip > 0 {
					skip--
				}
			case "p", "div", "li", "tr", "h1", "h2", "h3", "h4":
				b.WriteString("\n")
			}
		}
	}
}

// normalizeSpace схлопывает пробелы внутри строк и пустые строки, обрезает края.
func normalizeSpace(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	lines := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	out := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			if !blank && len(out) > 0 {
				out = append(out, "")
			}
			blank = true
			continue
		}
		blank = false
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
