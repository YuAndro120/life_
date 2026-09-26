package originality

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

func post(id, src int64, minutes int, text string) Post {
	return Post{ID: id, SourceID: src, At: t0.Add(time.Duration(minutes) * time.Minute), Text: text}
}

const original = "Банк России принял решение сохранить ключевую ставку на уровне 16 процентов годовых. Совет директоров отметил замедление инфляции."

func TestNearVerbatimRewriteIsDerived(t *testing.T) {
	got := DerivedFrom([]Post{
		post(1, 10, 0, original),
		post(2, 20, 15, "Срочно: Банк России принял решение сохранить ключевую ставку на уровне 16 процентов годовых, сообщают агентства."),
	})
	if got[2] != 1 || len(got) != 1 {
		t.Fatalf("ждали {2:1}, получили %v", got)
	}
}

func TestIndependentReportsAreNotDerived(t *testing.T) {
	got := DerivedFrom([]Post{
		post(1, 10, 0, original),
		post(2, 20, 15, "Регулятор оставил стоимость заимствований прежней. Аналитики ожидали такого шага после августовских данных по инфляции в стране."),
	})
	if len(got) != 0 {
		t.Fatalf("независимые сообщения не должны считаться пересказом: %v", got)
	}
}

func TestSameSourceIsNeverDerived(t *testing.T) {
	got := DerivedFrom([]Post{post(1, 10, 0, original), post(2, 10, 5, original)})
	if len(got) != 0 {
		t.Fatalf("посты одного источника не сравниваются: %v", got)
	}
}

func TestEarlierPostIsTheOriginalRegardlessOfInputOrder(t *testing.T) {
	got := DerivedFrom([]Post{
		post(2, 20, 15, original+" Подробности позже."),
		post(1, 10, 0, original),
	})
	if got[2] != 1 || len(got) != 1 {
		t.Fatalf("получили %v", got)
	}
}

func TestChainPointsToTheFirstOriginal(t *testing.T) {
	got := DerivedFrom([]Post{
		post(1, 10, 0, original),
		post(2, 20, 10, original+" Комментарий эксперта."),
		post(3, 30, 20, original+" Комментарий эксперта. Ещё одно мнение."),
	})
	if got[2] != 1 || got[3] != 1 {
		t.Fatalf("оба должны ссылаться на первый пост: %v", got)
	}
}

func TestShortTextsAreNotJudged(t *testing.T) {
	got := DerivedFrom([]Post{post(1, 10, 0, "Путин заявил"), post(2, 20, 5, "Путин заявил")})
	if len(got) != 0 {
		t.Fatalf("слишком короткие тексты не сравниваются: %v", got)
	}
}
