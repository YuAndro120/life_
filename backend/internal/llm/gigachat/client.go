// Package gigachat — клиент GigaChat API: OAuth-токен с кэшем, строгий JSON через вызов функции, повторы.
package gigachat

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"shtil/backend/internal/llm"
)

//go:embed russian_trusted_root_ca.pem
var russianRootCA []byte

//go:embed digest_system.txt
var systemPrompt string

type Config struct {
	// AuthKey — ключ авторизации (base64 от «client_id:secret») из личного кабинета Сбера. Только из переменной окружения.
	AuthKey string
	Scope   string // GIGACHAT_API_PERS (физлицо) по умолчанию
	Model   string // GigaChat-2 по умолчанию
	BaseURL string
	AuthURL string
	// MaxAttempts — сколько раз просить модель при некорректном ответе (по умолчанию 3).
	MaxAttempts int
	HTTP        *http.Client
	Log         *slog.Logger
	Now         func() time.Time
	// Sleep подменяется в тестах, чтобы повторы не ждали по-настоящему.
	Sleep func(time.Duration)
}

type Client struct {
	cfg Config

	mu      sync.Mutex
	token   string
	expires time.Time
}

func New(cfg Config) (*Client, error) {
	if cfg.AuthKey == "" {
		return nil, errors.New("gigachat: не задан ключ авторизации (GIGACHAT_AUTH_KEY)")
	}
	if cfg.Scope == "" {
		cfg.Scope = "GIGACHAT_API_PERS"
	}
	if cfg.Model == "" {
		cfg.Model = "GigaChat-2"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://gigachat.devices.sberbank.ru/api/v1"
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Sleep == nil {
		cfg.Sleep = time.Sleep
	}
	if cfg.HTTP == nil {
		h, err := defaultHTTPClient()
		if err != nil {
			return nil, err
		}
		cfg.HTTP = h
	}
	return &Client{cfg: cfg}, nil
}

// defaultHTTPClient доверяет системным сертификатам и корневому сертификату Минцифры (его требуют серверы Сбера).
func defaultHTTPClient() (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(russianRootCA) {
		return nil, errors.New("gigachat: не удалось загрузить корневой сертификат Минцифры")
	}
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}, nil
}

// RootCAPEM отдаёт встроенный корневой сертификат (для тестов и диагностики).
func RootCAPEM() []byte { return russianRootCA }

// --- авторизация ---

func (c *Client) accessToken(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && c.token != "" && c.cfg.Now().Add(60*time.Second).Before(c.expires) {
		return c.token, nil
	}
	body := url.Values{"scope": {c.cfg.Scope}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.AuthURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", uuid4())
	req.Header.Set("Authorization", "Basic "+c.cfg.AuthKey)

	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("gigachat oauth: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		// Тело ответа не логируем и не возвращаем: в нём могут быть данные ключа.
		return "", fmt.Errorf("gigachat oauth: HTTP %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.AccessToken == "" {
		return "", errors.New("gigachat oauth: неожиданный ответ")
	}
	c.token = out.AccessToken
	// expires_at в миллисекундах; на случай секунд различаем по величине.
	if out.ExpiresAt > 1e11 {
		c.expires = time.UnixMilli(out.ExpiresAt)
	} else {
		c.expires = time.Unix(out.ExpiresAt, 0)
	}
	return c.token, nil
}

func uuid4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// --- обработка сюжета ---

type chatRequest struct {
	Model        string         `json:"model"`
	Messages     []chatMessage  `json:"messages"`
	Functions    []chatFunction `json:"functions"`
	FunctionCall map[string]any `json:"function_call"`
	Temperature  float64        `json:"temperature"`
	MaxTokens    int            `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content      string `json:"content"`
			FunctionCall *struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function_call"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

const functionName = "publish_story"

func digestFunction() chatFunction {
	enum := func(vals []string) map[string]any { return map[string]any{"type": "string", "enum": vals} }
	return chatFunction{
		Name:        functionName,
		Description: "Записать нейтральный заголовок, пересказ и классификацию сюжета",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topic":         enum(llm.Topics),
				"info_type":     enum(llm.InfoTypes),
				"heaviness":     enum(llm.Heavinesss),
				"region_code":   map[string]any{"type": "string", "description": "только цифры кода региона России (например 78), без названия; или пустая строка"},
				"title":         map[string]any{"type": "string", "description": "нейтральный заголовок, 8–14 слов"},
				"summary":       map[string]any{"type": "string", "description": "2–3 предложения о том, что произошло"},
				"meaning":       map[string]any{"type": "string", "description": "что это значит для обычного человека, или пустая строка"},
				"is_newsworthy": map[string]any{"type": "boolean", "description": "true, если это новость для широкой аудитории; false для технических страниц, таблиц, регламентных объявлений"},
			},
			"required": []string{"topic", "info_type", "heaviness", "title", "summary", "is_newsworthy"},
		},
	}
}

// minRunesForMeaning — меньше этого объёма текста в постах вывод «что это значит» не строится.
const minRunesForMeaning = 300

func sourceRunes(in llm.StoryInput) int {
	n := 0
	for _, p := range in.Posts {
		n += len([]rune(p.Text))
	}
	return n
}

// Дешёвый предел на вход: не больше стольких постов и символов на пост уходит в модель.
const (
	maxPosts        = 6
	maxRunesPerPost = 1200
)

func userMessage(in llm.StoryInput) string {
	var b strings.Builder
	posts := in.Posts
	if len(posts) > maxPosts {
		posts = posts[:maxPosts]
	}
	for i, p := range posts {
		text := []rune(p.Text)
		if len(text) > maxRunesPerPost {
			text = text[:maxRunesPerPost]
		}
		fmt.Fprintf(&b, "Пост %d. Источник: %s (%s), %s\n%s\n\n", i+1, p.SourceTitle, p.SourceKind, p.PublishedAt.UTC().Format("2006-01-02 15:04"), string(text))
	}
	return strings.TrimSpace(b.String())
}

// Digest просит модель обработать сюжет. При некорректном ответе повторяет запрос (до MaxAttempts),
// но исправлять ответ догадками не пытается.
func (c *Client) Digest(ctx context.Context, in llm.StoryInput) (llm.Digest, llm.Usage, error) {
	if len(in.Posts) == 0 {
		return llm.Digest{}, llm.Usage{}, errors.New("нет постов")
	}
	reqBody, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage(in)},
		},
		Functions:    []chatFunction{digestFunction()},
		FunctionCall: map[string]any{"name": functionName},
		Temperature:  0.1,
		MaxTokens:    700,
	})
	if err != nil {
		return llm.Digest{}, llm.Usage{}, err
	}

	var total llm.Usage
	var lastErr error
	for attempt := 1; attempt <= c.cfg.MaxAttempts; attempt++ {
		resp, err := c.chat(ctx, reqBody)
		if err != nil {
			return llm.Digest{}, total, err // сетевые и HTTP-ошибки повторяются внутри chat
		}
		total.PromptTokens += resp.Usage.PromptTokens
		total.CompletionTokens += resp.Usage.CompletionTokens

		d, err := parseDigest(resp)
		if err == nil {
			if err = d.Validate(); err == nil {
				if reason := d.SanitizeMeaning(); reason != "" {
					c.cfg.Log.Info("«значит» отброшено", "причина", reason)
				}
				// Политика: если в источниках только заголовки, «что это значит» модель может лишь домыслить, поэтому его нет.
				if sourceRunes(in) < minRunesForMeaning && d.Meaning != "" {
					c.cfg.Log.Info("«значит» отброшено: в источниках только заголовок")
					d.Meaning = ""
				}
				return d, total, nil
			}
		}
		lastErr = err
		c.cfg.Log.Warn("модель вернула некорректный ответ", "attempt", attempt, "err", err)
	}
	return llm.Digest{}, total, fmt.Errorf("%w: %v", llm.ErrInvalid, lastErr)
}

func parseDigest(resp chatResponse) (llm.Digest, error) {
	if len(resp.Choices) == 0 {
		return llm.Digest{}, errors.New("пустой список choices")
	}
	fc := resp.Choices[0].Message.FunctionCall
	if fc == nil || fc.Name != functionName {
		return llm.Digest{}, errors.New("модель не вызвала функцию")
	}
	args := fc.Arguments
	// Аргументы приходят объектом; на случай строки с JSON внутри разворачиваем её.
	var asString string
	if json.Unmarshal(args, &asString) == nil {
		args = json.RawMessage(asString)
	}
	var raw struct {
		Topic      string `json:"topic"`
		InfoType   string `json:"info_type"`
		Heaviness  string `json:"heaviness"`
		RegionCode string `json:"region_code"`
		Title      string `json:"title"`
		Summary    string `json:"summary"`
		Meaning    string `json:"meaning"`
		Newsworthy *bool  `json:"is_newsworthy"`
	}
	dec := json.NewDecoder(bytes.NewReader(args))
	if err := dec.Decode(&raw); err != nil {
		return llm.Digest{}, fmt.Errorf("аргументы функции не JSON: %w", err)
	}
	if raw.Newsworthy == nil {
		return llm.Digest{}, errors.New("нет поля is_newsworthy")
	}
	return llm.Digest{
		Newsworthy: *raw.Newsworthy,
		Topic:      strings.TrimSpace(raw.Topic), InfoType: strings.TrimSpace(raw.InfoType), Heaviness: strings.TrimSpace(raw.Heaviness),
		RegionCode: strings.TrimSpace(raw.RegionCode), Title: strings.TrimSpace(raw.Title),
		Summary: strings.TrimSpace(raw.Summary), Meaning: strings.TrimSpace(raw.Meaning),
	}, nil
}

// chat отправляет запрос. Повторяет при 429 и 5xx с нарастающей паузой; при 401 один раз обновляет токен.
func (c *Client) chat(ctx context.Context, body []byte) (chatResponse, error) {
	refreshed := false
	var lastErr error
	for try := 0; try < 4; try++ {
		if try > 0 {
			c.cfg.Sleep(time.Duration(1<<uint(try-1)) * time.Second)
		}
		token, err := c.accessToken(ctx, false)
		if err != nil {
			return chatResponse{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return chatResponse{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.cfg.HTTP.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("gigachat chat: %w", err)
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusOK:
			var out chatResponse
			if err := json.Unmarshal(raw, &out); err != nil {
				return chatResponse{}, fmt.Errorf("gigachat chat: неожиданный ответ: %w", err)
			}
			return out, nil
		case resp.StatusCode == http.StatusUnauthorized && !refreshed:
			refreshed = true
			if _, err := c.accessToken(ctx, true); err != nil {
				return chatResponse{}, err
			}
			lastErr = errors.New("gigachat chat: HTTP 401, токен обновлён")
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = fmt.Errorf("gigachat chat: HTTP %d", resp.StatusCode)
		case resp.StatusCode == http.StatusPaymentRequired:
			return chatResponse{}, errors.New("gigachat chat: HTTP 402, исчерпан лимит токенов тарифа")
		default:
			return chatResponse{}, fmt.Errorf("gigachat chat: HTTP %d", resp.StatusCode)
		}
	}
	return chatResponse{}, lastErr
}
