package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"shtil/backend/internal/llm"
)

func reply(args string) string {
	quoted, _ := json.Marshal(args)
	return fmt.Sprintf(`{"choices":[{"message":{"content":"","tool_calls":[{"function":{"name":"publish_story","arguments":%s}}]}}],"usage":{"prompt_tokens":100,"completion_tokens":30}}`, quoted)
}

const goodArgs = `{"topic":"economy","info_type":"official","heaviness":"neutral","region_code":"","title":"Банк России сохранил ключевую ставку на уровне 16 процентов годовых","summary":"Совет директоров Банка России принял решение не менять ключевую ставку. Регулятор отметил замедление инфляции.","meaning":"","is_newsworthy":true}`

func newClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := New(Config{APIKey: "k", BaseURL: srv.URL, Model: "test/model", Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Sleep: func(time.Duration) {}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func story() llm.StoryInput {
	return llm.StoryInput{Posts: []llm.Post{{SourceTitle: "ЦБ", SourceKind: "gov", Lang: "ru", Text: "Банк России сохранил ключевую ставку.", PublishedAt: time.Now()}}}
}

func TestDigestSendsToolRequestAndParsesResult(t *testing.T) {
	var got request
	var auth string
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		io.WriteString(w, reply(goodArgs))
	})
	d, u, err := c.Digest(context.Background(), story())
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer k" || got.Model != "test/model" || len(got.Tools) != 1 || got.Tools[0].Function.Name != "publish_story" {
		t.Fatalf("запрос: auth=%q model=%q tools=%+v", auth, got.Model, got.Tools)
	}
	if d.Topic != "economy" || !d.Newsworthy || u.PromptTokens != 100 || u.CompletionTokens != 30 {
		t.Fatalf("ответ: %+v %+v", d, u)
	}
}

func TestDigestRetriesInvalidThenGivesUp(t *testing.T) {
	var calls int32
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		io.WriteString(w, reply(`{"topic":"nope","info_type":"official","heaviness":"neutral","title":"x","summary":"y","is_newsworthy":true}`))
	})
	_, u, err := c.Digest(context.Background(), story())
	if !errors.Is(err, llm.ErrInvalid) || atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("err=%v, вызовов=%d", err, calls)
	}
	if u.PromptTokens != 300 {
		t.Fatalf("токены за все попытки: %d", u.PromptTokens)
	}
}

func TestRetriesServerErrorsAndReportsRateLimit(t *testing.T) {
	var calls int32
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			http.Error(w, "busy", 503)
			return
		}
		io.WriteString(w, reply(goodArgs))
	})
	if _, _, err := c.Digest(context.Background(), story()); err != nil {
		t.Fatal(err)
	}
	limited := newClient(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "slow down", 429) })
	if _, _, err := limited.Digest(context.Background(), story()); !errors.Is(err, llm.ErrRateLimited) {
		t.Fatalf("ждали ErrRateLimited, получили %v", err)
	}
}

func TestAuthAndBalanceErrorsAreFatalAndClear(t *testing.T) {
	for code, want := range map[int]string{401: "ключ", 402: "средств"} {
		c := newClient(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", code) })
		_, _, err := c.Digest(context.Background(), story())
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("HTTP %d: %v", code, err)
		}
	}
}

func TestExtractLawUsesFragmentsAndModelOverride(t *testing.T) {
	var got request
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		args, _ := json.Marshal(map[string]any{
			"relevant": true, "title": "Меняется срок уведомлений об авансовых платежах для ИП",
			"what_changed": "Срок уведомления сокращается до 20 числа.", "who_affected": "ИП на упрощённой системе.",
			"audience_tags": "work:ip_usn", "effective_at": "2026-10-01",
			"what_changed_ref": 1, "who_affected_ref": 1, "effective_ref": 2,
		})
		quoted, _ := json.Marshal(string(args))
		fmt.Fprintf(w, `{"choices":[{"message":{"tool_calls":[{"function":{"name":"extract_law","arguments":%s}}]}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`, quoted)
	})
	c = c.WithModel("anthropic/claude-sonnet-5")
	d, _, err := c.ExtractLaw(context.Background(), llm.LawInput{Title: "Закон", Number: "1-ФЗ", Fragments: []llm.Fragment{
		{N: 1, Text: "Срок уведомления ИП об авансовых платежах сокращается до 20 числа."},
		{N: 2, Text: "Настоящий Федеральный закон вступает в силу с 1 октября 2026 года."},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "anthropic/claude-sonnet-5" || !strings.Contains(got.Messages[1].Content, "[2] Настоящий") {
		t.Fatalf("запрос: %q / %q", got.Model, got.Messages[1].Content)
	}
	if !strings.Contains(d.Quotes.EffectiveDay, "1 октября 2026") || d.EffectiveAt != "2026-10-01" {
		t.Fatalf("черновик: %+v", d)
	}
}

func TestNewRequiresKeyAndModel(t *testing.T) {
	if _, err := New(Config{Model: "m"}); err == nil {
		t.Fatal("нужен ключ")
	}
	if _, err := New(Config{APIKey: "k"}); err == nil {
		t.Fatal("нужна модель")
	}
}
