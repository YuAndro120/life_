// Package catalog читает каталог источников (seeds/sources.yaml) и применяет главное правило проекта:
// источник без даты проверки по реестрам нежелательных организаций и иноагентов не активируется.
package catalog

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Entry struct {
	Kind      string `yaml:"kind"`
	Handle    string `yaml:"handle"`
	URL       string `yaml:"url"`
	Title     string `yaml:"title"`
	TopicHint string `yaml:"topic_hint"`
	// Country — страна издания (RU, US, GB, EU): по ней пользователь выбирает, откуда читать новости.
	Country string `yaml:"country"`
	// Region — код субъекта РФ для региональных источников (двузначный номер); пусто у федеральных.
	Region string `yaml:"region"`
	// Lang — язык постов (ru по умолчанию). Пересказ всегда по-русски.
	Lang string `yaml:"lang"`
	// Active — желание владельца включить источник. Фактически он включится только при legal_checked.
	Active bool `yaml:"active"`
	// LegalChecked — дата (ГГГГ-ММ-ДД), когда источник проверен по реестрам нежелательных организаций и иноагентов.
	LegalChecked string `yaml:"legal_checked"`
	Note         string `yaml:"note"`
}

type file struct {
	Sources []Entry `yaml:"sources"`
}

var (
	kinds     = map[string]bool{"rss": true, "tg": true, "gov": true, "site": true}
	handleRe  = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	countryRe = regexp.MustCompile(`^[A-Z]{2}$`)
	regionRe  = regexp.MustCompile(`^[0-9]{2}$`)
	langRe    = regexp.MustCompile(`^[a-z]{2}$`)
	topics    = map[string]bool{
		"economy": true, "finance": true, "law": true, "tech_ai": true, "city": true, "health": true,
		"education": true, "transport": true, "housing": true, "science": true, "culture": true,
		"sport": true, "showbiz": true, "space": true, "crypto": true, "politics": true, "crime": true,
		"incidents": true, "disasters": true,
	}
)

func Load(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func Parse(r io.Reader) ([]Entry, error) {
	var doc file
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("разбор каталога: %w", err)
	}
	seen := map[string]bool{}
	for i := range doc.Sources {
		e := &doc.Sources[i]
		if e.Lang == "" {
			e.Lang = "ru"
		}
		if e.Kind == "tg" && e.URL == "" && e.Handle != "" {
			e.URL = "https://t.me/s/" + e.Handle
		}
		if err := e.Validate(); err != nil {
			return nil, fmt.Errorf("источник %d (%s): %w", i+1, e.Handle, err)
		}
		key := e.Kind + "/" + e.Handle
		if seen[key] {
			return nil, fmt.Errorf("источник %s указан дважды", key)
		}
		seen[key] = true
	}
	return doc.Sources, nil
}

func (e Entry) Validate() error {
	if !kinds[e.Kind] {
		return fmt.Errorf("неизвестный kind %q", e.Kind)
	}
	if !handleRe.MatchString(e.Handle) {
		return fmt.Errorf("handle %q: допустимы латиница, цифры и _ . -", e.Handle)
	}
	if strings.TrimSpace(e.Title) == "" {
		return fmt.Errorf("не задан title")
	}
	u, err := url.Parse(e.URL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("url %q должен быть http(s)", e.URL)
	}
	if e.Kind == "tg" && u.Host != "t.me" {
		return fmt.Errorf("для tg ожидается t.me, получено %q", u.Host)
	}
	if e.TopicHint != "" && !topics[e.TopicHint] {
		return fmt.Errorf("неизвестный topic_hint %q", e.TopicHint)
	}
	if e.Country != "" && !countryRe.MatchString(e.Country) {
		return fmt.Errorf("country %q: ожидается двухбуквенный код страны", e.Country)
	}
	if e.Region != "" && !regionRe.MatchString(e.Region) {
		return fmt.Errorf("region %q: ожидается двузначный код региона", e.Region)
	}
	if !langRe.MatchString(e.Lang) {
		return fmt.Errorf("lang %q: ожидается двухбуквенный код языка", e.Lang)
	}
	if e.LegalChecked != "" {
		if _, err := time.Parse("2006-01-02", e.LegalChecked); err != nil {
			return fmt.Errorf("legal_checked %q: ожидается ГГГГ-ММ-ДД", e.LegalChecked)
		}
	}
	return nil
}

// LegalCheckedAt возвращает дату проверки по реестрам, если она указана.
func (e Entry) LegalCheckedAt() (time.Time, bool) {
	if e.LegalChecked == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", e.LegalChecked)
	return t, err == nil
}

// EffectiveActive: источник включается, только если владелец его включил И указана дата проверки по реестрам.
// Вторая строка — причина, если источник остался выключенным.
func (e Entry) EffectiveActive() (bool, string) {
	if !e.Active {
		return false, "выключен в каталоге"
	}
	if _, ok := e.LegalCheckedAt(); !ok {
		return false, "нет legal_checked: сначала проверить по реестрам нежелательных организаций и иноагентов"
	}
	return true, ""
}
