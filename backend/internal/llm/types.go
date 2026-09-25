// Package llm описывает обращение к языковой модели: вход (посты сюжета), строгий выход и его проверку.
package llm

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Post struct {
	SourceTitle string
	SourceKind  string
	Lang        string // ru, en …
	URL         string
	PublishedAt time.Time
	Text        string
}

type StoryInput struct {
	Posts []Post
}

// Digest — результат обработки сюжета: классификация и нейтральный пересказ.
type Digest struct {
	Topic      string
	InfoType   string
	Heaviness  string
	RegionCode string // пусто, если событие не региональное
	Title      string
	Summary    string
	Meaning    string // пусто, если вывод не следует из фактов
	// Newsworthy — новость для широкой аудитории. false у технических страниц, таблиц и регламентных объявлений.
	Newsworthy bool
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Client — обработчик сюжетов. Реализации: GigaChat, подделка для тестов.
type Client interface {
	Digest(ctx context.Context, in StoryInput) (Digest, Usage, error)
}

// ErrRateLimited — модель отвечает 429 (слишком частые запросы); вызывающий может подождать и повторить.
var ErrRateLimited = errors.New("модель ограничила частоту запросов")

// ErrInvalid — модель не дала корректный ответ после всех попыток. Ответ не «чинится» догадками.
var ErrInvalid = errors.New("модель вернула некорректный ответ")

var (
	Topics = []string{
		"economy", "finance", "law", "tech_ai", "city", "health", "education", "transport", "housing",
		"science", "space", "culture", "sport", "showbiz", "crypto", "politics", "crime", "incidents", "disasters",
	}
	InfoTypes  = []string{"fact", "official", "opinion", "forecast", "rumor"}
	Heavinesss = []string{"neutral", "tense", "heavy"}
)

var (
	regionRe    = regexp.MustCompile(`^\d{2,3}$`)
	speculation = regexp.MustCompile(`(?i)(^|[^\p{L}])(может|могут|мог[а-я]*|возможно|вероятно|скорее всего|предположительно|видимо|наверное)([^\p{L}]|$)`)
	clickbait   = regexp.MustCompile(`(?i)(^|[^\p{L}])(шок|шокирующ|срочно|сенсаци|ужас|невероятн|вы не поверите|молния|скандал|взрывн|немедленно)`)
	titleShort  = 25
)

// Validate проверяет ответ модели по правилам проекта. Возвращает ошибку с причиной, чтобы её можно было залогировать.
func (d Digest) Validate() error {
	if !contains(Topics, d.Topic) {
		return fmt.Errorf("topic %q не из списка", d.Topic)
	}
	if !contains(InfoTypes, d.InfoType) {
		return fmt.Errorf("info_type %q не из списка", d.InfoType)
	}
	if !contains(Heavinesss, d.Heaviness) {
		return fmt.Errorf("heaviness %q не из списка", d.Heaviness)
	}
	if d.RegionCode != "" && !regionRe.MatchString(d.RegionCode) {
		return fmt.Errorf("region_code %q: ожидается 2–3 цифры или пусто", d.RegionCode)
	}
	if n := utf8.RuneCountInString(d.Title); n < titleShort || n > 160 {
		return fmt.Errorf("длина заголовка %d вне диапазона %d–160", n, titleShort)
	}
	if strings.ContainsAny(d.Title, "!") || strings.HasSuffix(strings.TrimSpace(d.Title), ".") {
		return errors.New("заголовок с восклицательным знаком или точкой в конце")
	}
	if clickbait.MatchString(d.Title) {
		return errors.New("в заголовке кликбейт")
	}
	if shoutingWords(d.Title) >= 2 {
		return errors.New("в заголовке слова ЗАГЛАВНЫМИ буквами")
	}
	if n := utf8.RuneCountInString(d.Summary); n < 40 || n > 700 {
		return fmt.Errorf("длина пересказа %d вне диапазона 40–700", n)
	}
	for name, field := range map[string]string{"заголовок": d.Title, "пересказ": d.Summary, "«значит»": d.Meaning} {
		if w := mixedScriptWord(field); w != "" {
			return fmt.Errorf("%s: в слове %q смешаны кириллица и латиница", name, w)
		}
	}
	if strings.EqualFold(strings.TrimSpace(d.Summary), strings.TrimSpace(d.Title)) {
		return errors.New("пересказ совпадает с заголовком")
	}
	return nil
}

// SanitizeMeaning убирает необязательное поле «значит», если оно нарушает правила: содержит предположения
// (может/возможно/вероятно) или слишком длинное. Остальной ответ остаётся как есть: ради необязательного
// поля не стоит терять сюжет и платить за повторные запросы. Возвращает причину или пустую строку.
func (d *Digest) SanitizeMeaning() string {
	switch {
	case d.Meaning == "":
		return ""
	case speculation.MatchString(d.Meaning):
		d.Meaning = ""
		return "«значит» содержало предположения (может/возможно/вероятно)"
	case utf8.RuneCountInString(d.Meaning) > 320:
		d.Meaning = ""
		return "«значит» длиннее 320 символов"
	}
	return ""
}

// mixedScriptWord возвращает первое слово, в котором перемешаны кириллица и латиница («Тommo», «Mосква»):
// так проявляются сбои перевода имён. Слова целиком латиницей (NASA, SpaceX) допустимы.
func mixedScriptWord(s string) string {
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && r != '-' }) {
		var cyr, lat bool
		for _, r := range w {
			switch {
			case unicode.Is(unicode.Cyrillic, r):
				cyr = true
			case unicode.Is(unicode.Latin, r):
				lat = true
			}
		}
		if cyr && lat {
			return w
		}
	}
	return ""
}

// shoutingWords считает слова длиннее трёх букв, написанные целиком заглавными (аббревиатуры до 3 букв — норма: ЦБ, МВД).
func shoutingWords(s string) int {
	n := 0
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if utf8.RuneCountInString(w) > 4 && w == strings.ToUpper(w) {
			n++
		}
	}
	return n
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
