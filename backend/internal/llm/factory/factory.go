// Package factory выбирает провайдера модели по переменным окружения.
//
//	LLM_PROVIDER=gigachat (по умолчанию) — GIGACHAT_AUTH_KEY, GIGACHAT_SCOPE, GIGACHAT_MODEL
//	LLM_PROVIDER=speshu                  — SPESHU_API_KEY, SPESHU_MODEL (пересказ), SPESHU_LAW_MODEL (законы), SPESHU_BASE_URL
package factory

import (
	"fmt"
	"log/slog"
	"os"

	"shtil/backend/internal/llm"
	"shtil/backend/internal/llm/gigachat"
	"shtil/backend/internal/llm/openai"
)

const (
	DefaultDigestModel = "google/gemini-3.1-flash-lite"
	DefaultLawModel    = "anthropic/claude-sonnet-5"
)

// Both — то, что умеет провайдер: пересказ сюжетов и разбор законов.
type Both interface {
	llm.Client
	llm.LawExtractor
}

func Provider() string {
	if p := os.Getenv("LLM_PROVIDER"); p != "" {
		return p
	}
	return "gigachat"
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// New возвращает клиента выбранного провайдера. Если ключа нет, возвращает nil без ошибки
// (воркер в этом случае только склеивает посты). lawModel != "" переопределяет модель для законов.
func New(log *slog.Logger, lawModel string) (digest llm.Client, laws llm.LawExtractor, name string, err error) {
	switch p := Provider(); p {
	case "gigachat":
		key := os.Getenv("GIGACHAT_AUTH_KEY")
		if key == "" {
			return nil, nil, p, nil
		}
		mk := func(model string) (*gigachat.Client, error) {
			return gigachat.New(gigachat.Config{AuthKey: key, Scope: os.Getenv("GIGACHAT_SCOPE"), Model: model, Log: log})
		}
		d, err := mk(os.Getenv("GIGACHAT_MODEL"))
		if err != nil {
			return nil, nil, p, err
		}
		if lawModel == "" {
			lawModel = env("GIGACHAT_LAW_MODEL", "GigaChat-2-Pro")
		}
		l, err := mk(lawModel)
		if err != nil {
			return nil, nil, p, err
		}
		return d, l, p + " " + d.Model() + " / законы " + lawModel, nil
	case "speshu":
		key := os.Getenv("SPESHU_API_KEY")
		if key == "" {
			return nil, nil, p, nil
		}
		base, err := openai.New(openai.Config{APIKey: key, BaseURL: os.Getenv("SPESHU_BASE_URL"), Model: env("SPESHU_MODEL", DefaultDigestModel), Log: log})
		if err != nil {
			return nil, nil, p, err
		}
		if lawModel == "" {
			lawModel = env("SPESHU_LAW_MODEL", DefaultLawModel)
		}
		return base, base.WithModel(lawModel), fmt.Sprintf("speshu %s / законы %s", env("SPESHU_MODEL", DefaultDigestModel), lawModel), nil
	default:
		return nil, nil, p, fmt.Errorf("неизвестный LLM_PROVIDER %q (gigachat или speshu)", p)
	}
}
