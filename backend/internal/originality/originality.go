// Package originality находит в сюжете посты-пересказы: пост, почти дословно повторяющий более ранний пост другого источника,
// не считается независимым источником. Сравнение по общим последовательностям слов (шинглам).
//
// Ограничение: ловится только близкое к дословному заимствование. Пересказ своими словами и ссылки вида «сообщает ТАСС»
// не распознаются, поэтому оценка независимости остаётся оптимистичной (нижняя граница числа пересказов).
package originality

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

// Post — пост сюжета.
type Post struct {
	ID       int64
	SourceID int64
	At       time.Time
	Text     string
}

const (
	shingleSize = 4
	// minShingles — при меньшем числе шинглов (очень короткий текст) сходство ненадёжно, пересказом не считаем.
	minShingles = 4
	// Threshold — доля общих шинглов от меньшего из двух текстов, начиная с которой пост считается пересказом.
	Threshold = 0.3
)

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

func shingles(text string) map[string]struct{} {
	words := wordRe.FindAllString(strings.ReplaceAll(strings.ToLower(text), "ё", "е"), -1)
	out := make(map[string]struct{})
	for i := 0; i+shingleSize <= len(words); i++ {
		out[strings.Join(words[i:i+shingleSize], " ")] = struct{}{}
	}
	return out
}

func overlap(a, b map[string]struct{}) float64 {
	if len(a) < minShingles || len(b) < minShingles {
		return 0
	}
	small, big := a, b
	if len(b) < len(a) {
		small, big = b, a
	}
	common := 0
	for k := range small {
		if _, ok := big[k]; ok {
			common++
		}
	}
	return float64(common) / float64(len(small))
}

// DerivedFrom возвращает для каждого поста-пересказа ID более раннего поста другого источника, который он повторяет.
// Посты без пересказа в результат не входят. Порядок: по времени, затем по ID; ищется самый ранний подходящий пост.
func DerivedFrom(posts []Post) map[int64]int64 {
	sorted := append([]Post(nil), posts...)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].At.Equal(sorted[j].At) {
			return sorted[i].At.Before(sorted[j].At)
		}
		return sorted[i].ID < sorted[j].ID
	})
	sets := make([]map[string]struct{}, len(sorted))
	for i, p := range sorted {
		sets[i] = shingles(p.Text)
	}
	out := make(map[int64]int64)
	for i := range sorted {
		for j := 0; j < i; j++ {
			if sorted[j].SourceID == sorted[i].SourceID {
				continue
			}
			if _, isCopy := out[sorted[j].ID]; isCopy {
				continue // ссылаемся на самый первый в цепочке, а не на копию копии
			}
			if overlap(sets[i], sets[j]) >= Threshold {
				out[sorted[i].ID] = sorted[j].ID
				break
			}
		}
	}
	return out
}
