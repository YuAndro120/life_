// Package cluster склеивает посты об одном событии без эмбеддингов: TF-IDF по основам слов и косинусная близость.
// Для новостей на одном языке этого хватает: агентства повторяют имена, организации и числа.
package cluster

import (
	"math"
	"sort"
	"time"

	"shtil/backend/internal/text"
)

// Параметры подбираются на реальных данных (`worker debug-clusters`), значения по умолчанию — стартовые.
type Config struct {
	// Threshold — минимальная косинусная близость поста к центру сюжета.
	Threshold float64
	// Window — насколько давно последний пост сюжета, чтобы к нему ещё можно было присоединиться.
	Window time.Duration
	// LeadRunes — сколько первых символов поста учитывается (заголовок и начало).
	LeadRunes int
}

func DefaultConfig() Config {
	return Config{Threshold: 0.30, Window: 36 * time.Hour, LeadRunes: 600}
}

type Doc struct {
	ID       int64
	SourceID int64
	Text     string
	At       time.Time
}

// Story — существующий или новый сюжет со своими документами.
type Story struct {
	ID   int64 // у новых сюжетов отрицательный временный ID
	Docs []Doc
}

type Assignment struct {
	DocID    int64
	StoryID  int64
	NewStory bool
	Score    float64
}

type vector map[string]float64

// Assign относит каждый новый пост к лучшему сюжету или создаёт новый. Посты обрабатываются по времени.
// existing не изменяется. Возвращает решения в порядке обработки и итоговое состояние сюжетов.
func Assign(cfg Config, existing []Story, fresh []Doc) ([]Assignment, []Story) {
	stories := make([]Story, len(existing))
	for i, s := range existing {
		stories[i] = Story{ID: s.ID, Docs: append([]Doc(nil), s.Docs...)}
	}
	docs := append([]Doc(nil), fresh...)
	sort.SliceStable(docs, func(i, j int) bool { return docs[i].At.Before(docs[j].At) })

	// IDF считаем по всему корпусу окна: и по старым, и по новым постам.
	var all []Doc
	for _, s := range stories {
		all = append(all, s.Docs...)
	}
	all = append(all, docs...)
	idf := buildIDF(cfg, all)

	centroids := make([]vector, len(stories))
	for i, s := range stories {
		centroids[i] = centroid(cfg, idf, s.Docs)
	}

	var out []Assignment
	nextTemp := int64(-1)
	for _, d := range docs {
		v := vectorize(cfg, idf, d.Text)
		best, bestScore := -1, 0.0
		for i, s := range stories {
			if d.At.Sub(lastAt(s)) > cfg.Window {
				continue
			}
			if score := cosine(v, centroids[i]); score > bestScore {
				best, bestScore = i, score
			}
		}
		if best >= 0 && bestScore >= cfg.Threshold {
			stories[best].Docs = append(stories[best].Docs, d)
			centroids[best] = centroid(cfg, idf, stories[best].Docs)
			out = append(out, Assignment{DocID: d.ID, StoryID: stories[best].ID, Score: bestScore})
			continue
		}
		stories = append(stories, Story{ID: nextTemp, Docs: []Doc{d}})
		centroids = append(centroids, v)
		out = append(out, Assignment{DocID: d.ID, StoryID: nextTemp, NewStory: true, Score: bestScore})
		nextTemp--
	}
	return out, stories
}

func lastAt(s Story) time.Time {
	var t time.Time
	for _, d := range s.Docs {
		if d.At.After(t) {
			t = d.At
		}
	}
	return t
}

func lead(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func buildIDF(cfg Config, docs []Doc) map[string]float64 {
	df := map[string]int{}
	for _, d := range docs {
		seen := map[string]bool{}
		for _, t := range text.Terms(lead(d.Text, cfg.LeadRunes)) {
			if !seen[t.Stem] {
				seen[t.Stem] = true
				df[t.Stem]++
			}
		}
	}
	n := float64(len(docs))
	idf := make(map[string]float64, len(df))
	for term, c := range df {
		idf[term] = math.Log((n+1)/(float64(c)+1)) + 1
	}
	return idf
}

func vectorize(cfg Config, idf map[string]float64, s string) vector {
	v := vector{}
	for _, t := range text.Terms(lead(s, cfg.LeadRunes)) {
		w := idf[t.Stem]
		if w == 0 {
			w = 1
		}
		v[t.Stem] += t.Boost * w
	}
	for k, x := range v {
		v[k] = math.Sqrt(x) // повтор слова даёт убывающую отдачу
	}
	return normalize(v)
}

func centroid(cfg Config, idf map[string]float64, docs []Doc) vector {
	sum := vector{}
	for _, d := range docs {
		for k, x := range vectorize(cfg, idf, d.Text) {
			sum[k] += x
		}
	}
	return normalize(sum)
}

func normalize(v vector) vector {
	var n float64
	for _, x := range v {
		n += x * x
	}
	n = math.Sqrt(n)
	if n == 0 {
		return v
	}
	for k := range v {
		v[k] /= n
	}
	return v
}

func cosine(a, b vector) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for k, x := range a {
		dot += x * b[k]
	}
	return dot
}
