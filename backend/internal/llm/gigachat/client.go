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
	"shtil/backend/internal/llm/tasks"
)

//go:embed russian_trusted_root_ca.pem
var russianRootCA []byte

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

// Digest просит модель обработать сюжет. При некорректном ответе повторяет запрос (до MaxAttempts),
// но исправлять ответ догадками не пытается.
func (c *Client) Digest(ctx context.Context, in llm.StoryInput) (llm.Digest, llm.Usage, error) {
	if len(in.Posts) == 0 {
		return llm.Digest{}, llm.Usage{}, errors.New("нет постов")
	}
	reqBody, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: tasks.DigestSystem},
			{Role: "user", Content: tasks.DigestUser(in)},
		},
		Functions:    []chatFunction{toFunction(tasks.DigestSchema())},
		FunctionCall: map[string]any{"name": tasks.DigestFunction},
		Temperature:  0.1,
		MaxTokens:    tasks.DigestMaxTokens,
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

		args, err := functionArgs(resp, tasks.DigestFunction)
		if err == nil {
			var d llm.Digest
			if d, err = tasks.ParseDigest(args); err == nil {
				if err = tasks.FinishDigest(&d, in, c.cfg.Log.Info); err == nil {
					return d, total, nil
				}
			}
		}
		lastErr = err
		c.cfg.Log.Warn("модель вернула некорректный ответ", "attempt", attempt, "err", err)
	}
	return llm.Digest{}, total, fmt.Errorf("%w: %v", llm.ErrInvalid, lastErr)
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
		case resp.StatusCode == http.StatusTooManyRequests:
			lastErr = fmt.Errorf("gigachat chat: HTTP 429: %w", llm.ErrRateLimited)
		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("gigachat chat: HTTP %d", resp.StatusCode)
		case resp.StatusCode == http.StatusPaymentRequired:
			return chatResponse{}, errors.New("gigachat chat: HTTP 402, исчерпан лимит токенов тарифа")
		default:
			return chatResponse{}, fmt.Errorf("gigachat chat: HTTP %d", resp.StatusCode)
		}
	}
	return chatResponse{}, lastErr
}

func toFunction(s tasks.Schema) chatFunction {
	return chatFunction{Name: s.Name, Description: s.Description, Parameters: s.Parameters}
}

// functionArgs достаёт аргументы вызванной функции из ответа.
func functionArgs(resp chatResponse, name string) (json.RawMessage, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("пустой список choices")
	}
	fc := resp.Choices[0].Message.FunctionCall
	if fc == nil || fc.Name != name {
		return nil, errors.New("модель не вызвала функцию")
	}
	return fc.Arguments, nil
}
