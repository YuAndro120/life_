// Package sources получает и разбирает посты из внешних источников: RSS/Atom и публичное превью Telegram.
package sources

import "time"

// RawPost — пост в едином виде, независимо от источника.
type RawPost struct {
	ExternalID  string
	URL         string
	PublishedAt time.Time
	Text        string
}

// MaxTextRunes ограничивает длину сохраняемого текста: для эмбеддинга и саммари хватает начала.
const MaxTextRunes = 8000

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
