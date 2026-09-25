// Package openai — клиент для сервисов с OpenAI-совместимым API (SpeShu.AI и подобные): ключ в заголовке,
// строгий JSON через вызов функции (tools), повторы. Промпты и проверка ответов общие с GigaChat (пакет tasks).
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"shtil/backend/internal/llm"
	"shtil/backend/internal/llm/tasks"
)

const DefaultBaseURL = "https://speshu.ai/api/v1"

type Config struct {
	// APIKey — только из переменной окружения.
	APIKey  string
	BaseURL string
	// Model — идентификатор вида «google/gemini-3.1-flash-lite».
	Model       string
	MaxAttempts int
	HTTP        *http.Client
	Log         *slog.Logger
	Sleep       func(time.Duration)
}

type Client struct{ cfg Config }

func New(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("openai: не задан ключ API")
	}
	if cfg.Model == "" {
		return nil, errors.New("openai: не задана модель")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 90 * time.Second}
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.Sleep == nil {
		cfg.Sleep = time.Sleep
	}
	return &Client{cfg: cfg}, nil
}

// WithModel возвращает копию клиента с другой моделью (например, более сильной для законов).
func (c *Client) WithModel(model string) *Client {
	cfg := c.cfg
	cfg.Model = model
	return &Client{cfg: cfg}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type tool struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type request struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Tools       []tool    `json:"tools"`
	ToolChoice  any       `json:"tool_choice"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (c *Client) body(system, user string, s tasks.Schema, maxTokens int) ([]byte, error) {
	return json.Marshal(request{
		Model:    c.cfg.Model,
		Messages: []message{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Tools:    []tool{{Type: "function", Function: function{Name: s.Name, Description: s.Description, Parameters: s.Parameters}}},
		ToolChoice: map[string]any{
			"type": "function", "function": map[string]any{"name": s.Name},
		},
		Temperature: 0.1,
		MaxTokens:   maxTokens,
	})
}

// Digest обрабатывает сюжет; при некорректном ответе повторяет запрос, но ответ не «чинит».
func (c *Client) Digest(ctx context.Context, in llm.StoryInput) (llm.Digest, llm.Usage, error) {
	if len(in.Posts) == 0 {
		return llm.Digest{}, llm.Usage{}, errors.New("нет постов")
	}
	body, err := c.body(tasks.DigestSystem, tasks.DigestUser(in), tasks.DigestSchema(), tasks.DigestMaxTokens)
	if err != nil {
		return llm.Digest{}, llm.Usage{}, err
	}
	var d llm.Digest
	usage, err := c.attempts(ctx, body, tasks.DigestFunction, "сюжет", func(args json.RawMessage) error {
		var perr error
		if d, perr = tasks.ParseDigest(args); perr != nil {
			return perr
		}
		return tasks.FinishDigest(&d, in, c.cfg.Log.Info)
	})
	if err != nil {
		return llm.Digest{}, usage, err
	}
	return d, usage, nil
}

// ExtractLaw извлекает структуру закона; поля подтверждаются номерами фрагментов текста.
func (c *Client) ExtractLaw(ctx context.Context, in llm.LawInput) (llm.LawDraft, llm.Usage, error) {
	if len(in.Fragments) == 0 {
		return llm.LawDraft{}, llm.Usage{}, errors.New("нет текста закона")
	}
	user, sent := tasks.LawUser(in)
	body, err := c.body(tasks.LawSystem, user, tasks.LawSchema(), tasks.LawMaxTokens)
	if err != nil {
		return llm.LawDraft{}, llm.Usage{}, err
	}
	var d llm.LawDraft
	usage, err := c.attempts(ctx, body, tasks.LawFunction, "закон", func(args json.RawMessage) error {
		var perr error
		if d, perr = tasks.ParseLaw(args, sent); perr != nil {
			return perr
		}
		return d.Validate()
	})
	if err != nil {
		return llm.LawDraft{}, usage, err
	}
	return d, usage, nil
}

func (c *Client) attempts(ctx context.Context, body []byte, fn, what string, check func(json.RawMessage) error) (llm.Usage, error) {
	var total llm.Usage
	var lastErr error
	for attempt := 1; attempt <= c.cfg.MaxAttempts; attempt++ {
		resp, err := c.post(ctx, body)
		if err != nil {
			return total, err
		}
		total.PromptTokens += resp.Usage.PromptTokens
		total.CompletionTokens += resp.Usage.CompletionTokens

		args, err := toolArgs(resp, fn)
		if err == nil {
			if err = check(args); err == nil {
				return total, nil
			}
		}
		lastErr = err
		c.cfg.Log.Warn("модель вернула некорректный ответ", "что", what, "attempt", attempt, "err", err)
	}
	return total, fmt.Errorf("%w: %v", llm.ErrInvalid, lastErr)
}

func toolArgs(r response, name string) (json.RawMessage, error) {
	if len(r.Choices) == 0 {
		return nil, errors.New("пустой список choices")
	}
	for _, tc := range r.Choices[0].Message.ToolCalls {
		if tc.Function.Name == name {
			return tc.Function.Arguments, nil
		}
	}
	return nil, errors.New("модель не вызвала функцию")
}

// post отправляет запрос. 429 и 5xx повторяются с нарастающей паузой; 401/402 — сразу ошибка с понятным текстом.
func (c *Client) post(ctx context.Context, body []byte) (response, error) {
	var lastErr error
	for try := 0; try < 4; try++ {
		if try > 0 {
			c.cfg.Sleep(time.Duration(1<<uint(try-1)) * time.Second)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return response{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
		resp, err := c.cfg.HTTP.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("openai chat: %w", err)
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			var out response
			if err := json.Unmarshal(raw, &out); err != nil {
				return response{}, fmt.Errorf("openai chat: неожиданный ответ: %w", err)
			}
			return out, nil
		case resp.StatusCode == http.StatusTooManyRequests:
			lastErr = fmt.Errorf("openai chat: HTTP 429: %w", llm.ErrRateLimited)
		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("openai chat: HTTP %d", resp.StatusCode)
		case resp.StatusCode == http.StatusUnauthorized:
			return response{}, errors.New("openai chat: HTTP 401, неверный ключ API")
		case resp.StatusCode == http.StatusPaymentRequired:
			return response{}, errors.New("openai chat: HTTP 402, на балансе не хватает средств")
		default:
			return response{}, fmt.Errorf("openai chat: HTTP %d: %s", resp.StatusCode, snippet(raw))
		}
	}
	return response{}, lastErr
}

func snippet(b []byte) string {
	if r := []rune(string(b)); len(r) > 200 {
		return string(r[:200]) + "…"
	}
	return string(b)
}
