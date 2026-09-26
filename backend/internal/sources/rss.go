package sources

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

// ParseFeed разбирает RSS 2.0 или Atom. `now` подставляется, если у записи нет даты.
// Понимает BOM в начале файла (его отдаёт, например, сайт Банка России) и windows-1251.
func ParseFeed(r io.Reader, now time.Time) ([]RawPost, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("чтение ленты: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	data = bytes.TrimLeft(data, " \t\r\n")

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = charsetReader
	dec.Strict = false

	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("разбор ленты: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "rss":
			var doc rssDoc
			if err := dec.DecodeElement(&doc, &start); err != nil {
				return nil, fmt.Errorf("rss: %w", err)
			}
			return doc.posts(now), nil
		case "feed":
			var doc atomDoc
			if err := dec.DecodeElement(&doc, &start); err != nil {
				return nil, fmt.Errorf("atom: %w", err)
			}
			return doc.posts(now), nil
		default:
			return nil, errors.New("не RSS и не Atom: корневой элемент " + start.Name.Local)
		}
	}
}

func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "utf-8", "utf8", "":
		return input, nil
	case "windows-1251", "cp1251", "win-1251":
		return charmap.Windows1251.NewDecoder().Reader(input), nil
	default:
		return nil, fmt.Errorf("кодировка %q не поддерживается", label)
	}
}

type rssDoc struct {
	Items []struct {
		Title string `xml:"title"`
		// Ссылок может быть несколько (`<link>` и пустой `<atom:link>`): берём первую непустую.
		Links       []string `xml:"link"`
		GUID        string   `xml:"guid"`
		Description string   `xml:"description"`
		// Полный текст в Яндекс-формате: используется, если описание пустое.
		FullText string `xml:"full-text"`
		PubDate  string `xml:"pubDate"`
	} `xml:"channel>item"`
}

func (d rssDoc) posts(now time.Time) []RawPost {
	out := make([]RawPost, 0, len(d.Items))
	for _, it := range d.Items {
		link := ""
		for _, l := range it.Links {
			if l = strings.TrimSpace(l); l != "" {
				link = l
				break
			}
		}
		id := strings.TrimSpace(it.GUID)
		if id == "" {
			id = link
		}
		if id == "" {
			continue
		}
		out = append(out, RawPost{
			ExternalID:  id,
			URL:         link,
			PublishedAt: parseTime(it.PubDate, now),
			Text:        joinText(it.Title, firstNonEmpty(it.Description, it.FullText)),
		})
	}
	return out
}

type atomDoc struct {
	Entries []struct {
		ID      string `xml:"id"`
		Title   string `xml:"title"`
		Summary string `xml:"summary"`
		Content string `xml:"content"`
		Updated string `xml:"updated"`
		Publish string `xml:"published"`
		Links   []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
	} `xml:"entry"`
}

func (d atomDoc) posts(now time.Time) []RawPost {
	out := make([]RawPost, 0, len(d.Entries))
	for _, e := range d.Entries {
		link := ""
		for _, l := range e.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		id := strings.TrimSpace(e.ID)
		if id == "" {
			id = link
		}
		if id == "" {
			continue
		}
		body := e.Summary
		if body == "" {
			body = e.Content
		}
		when := e.Publish
		if when == "" {
			when = e.Updated
		}
		out = append(out, RawPost{
			ExternalID:  id,
			URL:         link,
			PublishedAt: parseTime(when, now),
			Text:        joinText(e.Title, body),
		})
	}
	return out
}

// joinText: заголовок и описание (в описании может быть HTML), дубль заголовка в описании убирается.
func joinText(title, description string) string {
	t := HTMLToText(title)
	d := HTMLToText(description)
	switch {
	case t == "":
		return truncate(d, MaxTextRunes)
	case d == "" || strings.HasPrefix(d, t):
		if d != "" && len(d) > len(t) {
			return truncate(d, MaxTextRunes)
		}
		return t
	default:
		return truncate(t+"\n"+d, MaxTextRunes)
	}
}

var timeLayouts = []string{
	time.RFC1123Z, time.RFC1123,
	"Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 -0700", "2 Jan 2006 15:04:05 MST",
	time.RFC3339, time.RFC3339Nano,
	"2006-01-02T15:04:05", "2006-01-02 15:04:05",
}

func parseTime(s string, fallback time.Time) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return fallback
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
