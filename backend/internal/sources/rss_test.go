package sources

import (
	"strings"
	"testing"
	"time"
)

func TestFeedWithEmptyAtomLinkAndYandexFullText(t *testing.T) {
	const feed = `<?xml version="1.0"?><rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:yandex="http://news.yandex.ru" version="2.0"><channel><title>t</title>
<atom:link rel="self" href="https://example.org/rss"/>
<item><title>Заголовок новости</title><link>https://example.org/news/1</link><atom:link rel="alternate" href="https://example.org/news/1"/>
<description><![CDATA[]]></description><pubDate>Sat, 26 Sep 2026 13:03:00 +0300</pubDate>
<yandex:full-text>&lt;p&gt;Полный текст новости о важном событии.&lt;/p&gt;</yandex:full-text></item></channel></rss>`
	posts, err := ParseFeed(strings.NewReader(feed), time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC))
	if err != nil || len(posts) != 1 {
		t.Fatalf("постов %d, ошибка %v", len(posts), err)
	}
	if posts[0].URL != "https://example.org/news/1" || !strings.Contains(posts[0].Text, "Полный текст новости") {
		t.Fatalf("получили %+v", posts[0])
	}
}
