package legal

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Живая проверка обоих источников: SHTIL_LIVE=1 go test ./internal/legal -run Live -v
func TestLiveLawsAndText(t *testing.T) {
	if os.Getenv("SHTIL_LIVE") == "" {
		t.Skip("нужен SHTIL_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	docs, err := NewPravo().Laws(ctx, time.Now().AddDate(0, -2, 0), 5)
	if err != nil || len(docs) == 0 {
		t.Fatalf("список: %v, %d", err, len(docs))
	}
	found := 0
	for _, d := range docs {
		u, text, err := NewKremlin().Text(ctx, d)
		if err != nil {
			t.Logf("%s: %v", d.Number, err)
			continue
		}
		found++
		t.Logf("%s (%s) %s: %d знаков, принят: %v\n   %s", d.Number, d.Signed.Format("2006-01-02"), u, len([]rune(text)), PassedDate(text), truncate(text, 160))
		if !strings.Contains(text, "Федеральный закон") && !strings.Contains(text, "ФЕДЕРАЛЬНЫЙ ЗАКОН") {
			t.Errorf("%s: в тексте нет слов «Федеральный закон»", d.Number)
		}
	}
	if found == 0 {
		t.Fatal("ни у одного закона не нашёлся текст")
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
