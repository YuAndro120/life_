package sources

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ParseTelegramPreview разбирает публичное веб-превью канала (https://t.me/s/<канал>).
// Посты без текста (только медиа) пропускаются: обрабатывать нечего.
func ParseTelegramPreview(r io.Reader, channel string, now time.Time) ([]RawPost, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("разбор превью Telegram: %w", err)
	}
	var out []RawPost
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "tgme_widget_message") && attr(n, "data-post") != "" {
			if p, ok := messageToPost(n, channel, now); ok {
				out = append(out, p)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out, nil
}

func messageToPost(msg *html.Node, channel string, now time.Time) (RawPost, bool) {
	dataPost := attr(msg, "data-post") // «канал/123»
	_, idStr, found := strings.Cut(dataPost, "/")
	if !found {
		return RawPost{}, false
	}
	if _, err := strconv.Atoi(idStr); err != nil {
		return RawPost{}, false
	}

	var textNode, timeNode *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "div" && hasClass(n, "tgme_widget_message_text") && textNode == nil {
				textNode = n
			}
			if n.Data == "time" && timeNode == nil {
				timeNode = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(msg)
	if textNode == nil {
		return RawPost{}, false
	}

	var inner strings.Builder
	for c := textNode.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&inner, c); err != nil {
			return RawPost{}, false
		}
	}
	text := HTMLToText(inner.String())
	if text == "" {
		return RawPost{}, false
	}

	when := now
	if timeNode != nil {
		when = parseTime(attr(timeNode, "datetime"), now)
	}
	return RawPost{
		ExternalID:  idStr,
		URL:         "https://t.me/" + channel + "/" + idStr,
		PublishedAt: when,
		Text:        truncate(text, MaxTextRunes),
	}, true
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
