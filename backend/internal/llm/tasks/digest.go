// Package tasks — то, что не зависит от провайдера модели: промпты, схемы функций, входные сообщения,
// разбор и проверка ответов. Клиенты (gigachat, openai) только доставляют запрос и ответ.
package tasks

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"shtil/backend/internal/llm"
)

//go:embed digest_system.txt
var DigestSystem string

const (
	DigestFunction = "publish_story"
	LawFunction    = "extract_law"

	// DigestMaxTokens и LawMaxTokens — потолок ответа модели.
	DigestMaxTokens = 700
	LawMaxTokens    = 900
)

// Schema — описание вызываемой функции (JSON Schema параметров).
type Schema struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// DigestSchema — строгая схема ответа по сюжету.
func DigestSchema() Schema {
	enum := func(vals []string) map[string]any { return map[string]any{"type": "string", "enum": vals} }
	return Schema{
		Name:        DigestFunction,
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

// MinRunesForMeaning — меньше этого объёма текста в постах вывод «что это значит» не строится.
const MinRunesForMeaning = 300

func SourceRunes(in llm.StoryInput) int {
	n := 0
	for _, p := range in.Posts {
		n += len([]rune(p.Text))
	}
	return n
}

// Дешёвый предел на вход: не больше стольких постов и символов на пост уходит в модель.
const (
	MaxPosts        = 6
	MaxRunesPerPost = 1200
)

func DigestUser(in llm.StoryInput) string {
	var b strings.Builder
	posts := in.Posts
	if len(posts) > MaxPosts {
		posts = posts[:MaxPosts]
	}
	for i, p := range posts {
		text := []rune(p.Text)
		if len(text) > MaxRunesPerPost {
			text = text[:MaxRunesPerPost]
		}
		lang := p.Lang
		if lang == "" {
			lang = "ru"
		}
		fmt.Fprintf(&b, "Пост %d. Источник: %s (%s), язык %s, %s\n%s\n\n", i+1, p.SourceTitle, p.SourceKind, lang, p.PublishedAt.UTC().Format("2006-01-02 15:04"), string(text))
	}
	return strings.TrimSpace(b.String())
}

// UnwrapArgs: аргументы функции приходят объектом; на случай строки с JSON внутри разворачиваем её.
func UnwrapArgs(args json.RawMessage) json.RawMessage {
	var asString string
	if json.Unmarshal(args, &asString) == nil {
		return json.RawMessage(asString)
	}
	return args
}

// ParseDigest разбирает аргументы функции publish_story.
func ParseDigest(args json.RawMessage) (llm.Digest, error) {
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
	if err := json.NewDecoder(bytes.NewReader(UnwrapArgs(args))).Decode(&raw); err != nil {
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

// FinishDigest проверяет разобранный ответ и применяет политику «значит»: возвращает ошибку при нарушении схемы.
func FinishDigest(d *llm.Digest, in llm.StoryInput, log func(msg string, args ...any)) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if reason := d.SanitizeMeaning(); reason != "" {
		log("«значит» отброшено", "причина", reason)
	}
	// Политика: если в источниках только заголовки, «что это значит» модель может лишь домыслить, поэтому его нет.
	if SourceRunes(in) < MinRunesForMeaning && d.Meaning != "" {
		log("«значит» отброшено: в источниках только заголовок")
		d.Meaning = ""
	}
	return nil
}
