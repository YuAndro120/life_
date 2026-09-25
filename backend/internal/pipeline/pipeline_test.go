package pipeline

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sort"
	"sync"
	"testing"
	"time"

	"shtil/backend/internal/cluster"
	"shtil/backend/internal/llm"
)

var t0 = time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)

type memPost struct {
	cluster.Doc
	story    int64
	source   string
	official bool
}

type memStory struct {
	id                    int64
	digest                *llm.Digest
	digestCount, attempts int
	lastPost              time.Time
	posts, sources        int
	official              bool
	published, skipped    bool
	digestErr             string
}

// memStore — хранилище в памяти со смыслом реального: считает счётчики сюжетов и правило публикации.
type memStore struct {
	mu      sync.Mutex
	posts   []*memPost
	stories map[int64]*memStory
	next    int64
}

func newMem() *memStore { return &memStore{stories: map[int64]*memStory{}, next: 100} }

func (m *memStore) add(id, src int64, minutes int, source string, official bool, text string) {
	m.posts = append(m.posts, &memPost{Doc: cluster.Doc{ID: id, SourceID: src, Text: text, At: t0.Add(time.Duration(minutes) * time.Minute)}, source: source, official: official})
}

func (m *memStore) UnclusteredPosts(_ context.Context, since time.Time) ([]cluster.Doc, error) {
	var out []cluster.Doc
	for _, p := range m.posts {
		if p.story == 0 && p.At.After(since) {
			out = append(out, p.Doc)
		}
	}
	return out, nil
}

func (m *memStore) OpenStories(_ context.Context, since time.Time) ([]cluster.Story, error) {
	byID := map[int64]*cluster.Story{}
	for _, p := range m.posts {
		if p.story != 0 {
			if byID[p.story] == nil {
				byID[p.story] = &cluster.Story{ID: p.story}
			}
			byID[p.story].Docs = append(byID[p.story].Docs, p.Doc)
		}
	}
	var out []cluster.Story
	for _, s := range byID {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *memStore) CreateStory(_ context.Context, _ time.Time) (int64, error) {
	m.next++
	m.stories[m.next] = &memStory{id: m.next}
	return m.next, nil
}

func (m *memStore) AttachPost(_ context.Context, storyID, postID int64) error {
	for _, p := range m.posts {
		if p.ID == postID {
			p.story = storyID
		}
	}
	return nil
}

func (m *memStore) Recount(_ context.Context, storyID int64) error {
	st := m.stories[storyID]
	srcs := map[int64]bool{}
	st.posts, st.official, st.lastPost = 0, false, time.Time{}
	for _, p := range m.posts {
		if p.story == storyID {
			st.posts++
			srcs[p.SourceID] = true
			st.official = st.official || p.official
			if p.At.After(st.lastPost) {
				st.lastPost = p.At
			}
		}
	}
	st.sources = len(srcs)
	return nil
}

func (m *memStore) StoriesForDigest(_ context.Context, quietBefore time.Time, maxAttempts, batch int) ([]DigestCandidate, error) {
	var ids []int64
	for id, st := range m.stories {
		if !st.lastPost.After(quietBefore) && (st.digest == nil || st.posts > st.digestCount) && st.attempts < maxAttempts {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) > batch {
		ids = ids[:batch]
	}
	var out []DigestCandidate
	for _, id := range ids {
		out = append(out, DigestCandidate{ID: id, PostCount: m.stories[id].posts})
	}
	return out, nil
}

func (m *memStore) StoryPosts(_ context.Context, storyID int64, limit int) ([]llm.Post, error) {
	var out []llm.Post
	for _, p := range m.posts {
		if p.story == storyID && len(out) < limit {
			out = append(out, llm.Post{SourceTitle: p.source, Text: p.Text, PublishedAt: p.At})
		}
	}
	return out, nil
}

func (m *memStore) SaveDigest(_ context.Context, id int64, d llm.Digest, _ time.Time) error {
	st := m.stories[id]
	st.digest, st.digestCount, st.attempts, st.digestErr = &d, st.posts, 0, ""
	if !d.Newsworthy {
		st.skipped = true
	}
	return nil
}

func (m *memStore) RecordDigestFailure(_ context.Context, id int64, msg string) error {
	m.stories[id].attempts++
	m.stories[id].digestErr = msg
	return nil
}

func (m *memStore) Publish(_ context.Context) (int64, int64, error) {
	var p, u int64
	for _, st := range m.stories {
		ok := st.digest != nil && !st.skipped && (st.sources >= 2 || st.official)
		if ok && !st.published {
			st.published, p = true, p+1
		}
		if !ok && st.published && !(st.sources >= 2 || st.official) {
			st.published, u = false, u+1
		}
	}
	return p, u, nil
}

type fakeLLM struct {
	calls  int
	fn     func(in llm.StoryInput, call int) (llm.Digest, error)
	tokens int
}

func (f *fakeLLM) Digest(_ context.Context, in llm.StoryInput) (llm.Digest, llm.Usage, error) {
	f.calls++
	d, err := f.fn(in, f.calls)
	return d, llm.Usage{PromptTokens: f.tokens, CompletionTokens: f.tokens / 4}, err
}

func okDigest(in llm.StoryInput, _ int) (llm.Digest, error) {
	return llm.Digest{Topic: "economy", InfoType: "fact", Heaviness: "neutral", Newsworthy: true, Title: "Заголовок из " + in.Posts[0].Text, Summary: "Пересказ события по нескольким источникам"}, nil
}

func worker(store *memStore, l llm.Client, now time.Time) *Worker {
	w := New(store, l, slog.New(slog.NewTextHandler(io.Discard, nil)))
	w.Now = func() time.Time { return now }
	return w
}

func fill(m *memStore) {
	m.add(1, 1, 0, "ТАСС", false, "Банк России сохранил ключевую ставку на уровне 16% годовых")
	m.add(2, 2, 3, "РИА", false, "ЦБ оставил ключевую ставку без изменений — 16% годовых")
	m.add(3, 3, 30, "Минфин", true, "Минфин предложил изменить порядок уплаты авансовых платежей для ИП на УСН")
	m.add(4, 1, 45, "ТАСС", false, "Сборная России сыграла вничью с Сербией в товарищеском матче")
}

func TestClustersDigestsAndPublishes(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: okDigest, tokens: 400}
	sum, err := worker(m, l, t0.Add(2*time.Hour)).RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.NewPosts != 4 || sum.NewStories != 3 || sum.Attached != 1 {
		t.Errorf("склейка: %+v", sum)
	}
	if sum.Digested != 3 || l.calls != 3 || sum.PromptTokens != 1200 {
		t.Errorf("пересказ: %+v, вызовов %d", sum, l.calls)
	}
	// Опубликованы: ставка (2 источника) и Минфин (официальный). Матч (1 источник, не официальный) остаётся черновиком.
	if sum.Published != 2 {
		t.Errorf("опубликовано %d, ожидалось 2", sum.Published)
	}
	pub := 0
	for _, st := range m.stories {
		if st.published {
			pub++
		}
	}
	if pub != 2 || len(m.stories) != 3 {
		t.Errorf("сюжетов %d, опубликовано %d", len(m.stories), pub)
	}
}

func TestDebounceWaitsForQuietPeriod(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: okDigest}
	// Последний пост в 09:45; через 5 минут после него (<10) сюжет матча ещё «шумит».
	sum, _ := worker(m, l, t0.Add(50*time.Minute)).RunOnce(context.Background())
	if sum.Digested != 2 || l.calls != 2 {
		t.Errorf("пересказаны должны быть только сюжеты, затихшие >10 минут: %+v", sum)
	}
	sum, _ = worker(m, l, t0.Add(60*time.Minute)).RunOnce(context.Background())
	if sum.Digested != 1 {
		t.Errorf("после паузы должен обработаться третий: %+v", sum)
	}
}

func TestSecondRunDoesNotRedoWork(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: okDigest}
	w := worker(m, l, t0.Add(2*time.Hour))
	w.RunOnce(context.Background())
	sum, _ := w.RunOnce(context.Background())
	if sum.NewPosts != 0 || sum.Digested != 0 || l.calls != 3 {
		t.Errorf("повторный проход ничего не должен делать: %+v, вызовов %d", sum, l.calls)
	}
}

func TestNewPostTriggersReDigestAndJoinsExistingStory(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: okDigest}
	w := worker(m, l, t0.Add(2*time.Hour))
	w.RunOnce(context.Background())
	m.add(5, 3, 100, "Интерфакс", false, "Центробанк сохранил ключевую ставку 16% и сообщил о замедлении инфляции")
	w2 := worker(m, l, t0.Add(3*time.Hour))
	sum, _ := w2.RunOnce(context.Background())
	if sum.NewStories != 0 || sum.Attached != 1 {
		t.Errorf("пост должен присоединиться к сюжету о ставке: %+v", sum)
	}
	if sum.Digested != 1 || l.calls != 4 {
		t.Errorf("сюжет с новым постом должен быть пересказан заново: %+v, вызовов %d", sum, l.calls)
	}
}

func TestInvalidAnswerCountsAttemptsThenGivesUp(t *testing.T) {
	m := newMem()
	m.add(1, 1, 0, "Минфин", true, "Минфин предложил изменить порядок уплаты авансовых платежей для ИП на УСН")
	l := &fakeLLM{fn: func(llm.StoryInput, int) (llm.Digest, error) { return llm.Digest{}, llm.ErrInvalid }}
	w := worker(m, l, t0.Add(time.Hour))
	for i := 0; i < 5; i++ {
		w.RunOnce(context.Background())
	}
	if l.calls != 3 {
		t.Errorf("после %d неудачных попыток сюжет должен быть оставлен, вызовов %d", w.MaxAttempts, l.calls)
	}
	for _, st := range m.stories {
		if st.published || st.digest != nil || st.digestErr == "" {
			t.Errorf("сюжет без пересказа не должен публиковаться: %+v", st)
		}
	}
}

func TestOneBadStoryDoesNotBlockOthers(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: func(in llm.StoryInput, call int) (llm.Digest, error) {
		if call == 1 {
			return llm.Digest{}, llm.ErrInvalid
		}
		return okDigest(in, call)
	}}
	sum, _ := worker(m, l, t0.Add(2*time.Hour)).RunOnce(context.Background())
	if sum.Failed != 1 || sum.Digested != 2 {
		t.Errorf("%+v", sum)
	}
}

func TestNetworkErrorStopsRunWithoutBurningAttempts(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: func(llm.StoryInput, int) (llm.Digest, error) {
		return llm.Digest{}, errors.New("gigachat chat: HTTP 402, исчерпан лимит")
	}}
	sum, err := worker(m, l, t0.Add(2*time.Hour)).RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if l.calls != 1 || sum.Stopped == "" {
		t.Errorf("после ошибки сети/лимита проход должен остановиться: вызовов %d, %+v", l.calls, sum)
	}
	for _, st := range m.stories {
		if st.attempts != 0 {
			t.Error("сетевая ошибка не должна расходовать попытки сюжета")
		}
	}
}

func TestBatchLimitsSpend(t *testing.T) {
	m := newMem()
	fill(m)
	l := &fakeLLM{fn: okDigest}
	w := worker(m, l, t0.Add(2*time.Hour))
	w.Batch = 2
	sum, _ := w.RunOnce(context.Background())
	if sum.Digested != 2 || l.calls != 2 {
		t.Errorf("за проход обрабатывается не больше Batch сюжетов: %+v", sum)
	}
}

func TestNoLLMStillClustersButNeverPublishes(t *testing.T) {
	m := newMem()
	fill(m)
	sum, err := worker(m, nil, t0.Add(2*time.Hour)).RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.NewStories != 3 || sum.Published != 0 {
		t.Errorf("без модели сюжеты создаются, но не публикуются: %+v", sum)
	}
}

func TestOldPostsOutsideWindowAreIgnored(t *testing.T) {
	m := newMem()
	m.add(1, 1, -60*40, "ТАСС", false, "Банк России сохранил ключевую ставку на уровне 16% годовых")
	sum, _ := worker(m, &fakeLLM{fn: okDigest}, t0).RunOnce(context.Background())
	if sum.NewPosts != 0 {
		t.Errorf("пост старше окна не должен обрабатываться: %+v", sum)
	}
}

func TestNotNewsworthyStoryIsNeverPublished(t *testing.T) {
	m := newMem()
	m.add(1, 1, 0, "Банк России", true, "Финальная таксономия XBRL Банка России (версия 8.1.0.3)")
	l := &fakeLLM{fn: func(in llm.StoryInput, _ int) (llm.Digest, error) {
		d, _ := okDigest(in, 0)
		d.Newsworthy = false
		return d, nil
	}}
	sum, _ := worker(m, l, t0.Add(time.Hour)).RunOnce(context.Background())
	if sum.Digested != 1 || sum.Published != 0 {
		t.Errorf("техническая страница официального источника не должна публиковаться: %+v", sum)
	}
	for _, st := range m.stories {
		if !st.skipped || st.published {
			t.Errorf("%+v", st)
		}
	}
}
