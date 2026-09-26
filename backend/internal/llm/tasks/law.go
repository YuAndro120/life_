package tasks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"shtil/backend/internal/llm"
)

var LawSystem = `Ты извлекаешь структуру из текста российского федерального закона для приложения-дайджеста. Ты ничего не пересказываешь от себя и не додумываешь.

Правила:
1. Используй только текст закона. Если чего-то в тексте нет, оставь поле пустым. Никаких предположений и оценок.
2. relevant = true, только если закон заметно касается обычных граждан или индивидуальных предпринимателей: налоги, выплаты, права, штрафы, жильё, транспорт, работа, образование, здоровье, персональные данные, услуги. Ратификации, межгосударственные соглашения, внутренние процедуры органов власти, награды, бюджетные технические правки, узкоотраслевые нормы для организаций — false.
3. title: нейтральный заголовок 6–14 слов о сути изменения, без «шок», «срочно», восклицаний и точки в конце. Не копируй название закона «О внесении изменений…», а назови, что именно меняется.
4. what_changed: 1–3 коротких предложения, что именно меняется. Только факты из текста.
5. who_affected: кого касается, одним предложением. Только то, что следует из текста.
6. actions: до 3 действий, которые человек должен или может сделать, ТОЛЬКО если закон прямо их предписывает (подать, уведомить, зарегистрировать до срока). Иначе пустой список.
7. audience_tags: ОБЯЗАТЕЛЬНОЕ поле, строка с 1–4 тегами через запятую, только из списка: ` + strings.Join(llm.AudienceTags, ", ") + `. Пример: "work:ip_usn, work:employee". "all", если касается всех граждан. Пустым быть не может. Теги industry:* ставь, если закон касается работников или предприятий отрасли; sells:* — если закон задаёт правила продажи конкретных товаров или услуг (например, маркировка, алкоголь, маркетплейсы). Узкий закон не помечай тегом all.
8. effective_at: дата вступления в силу в формате YYYY-MM-DD, только если в тексте она указана календарной датой (например «вступает в силу с 1 марта 2027 года»). Если сказано «по истечении 10 дней после опубликования» или дата разная для разных статей, оставь пустую строку.
9. Текст разбит на пронумерованные фрагменты [N]. Не копируй текст, а укажи номера фрагментов, на которых основаны поля: what_changed_ref — где сказано, что меняется; who_affected_ref — где видно, кого касается (если прямо не сказано, тот же фрагмент, что и what_changed_ref); effective_ref — фрагмент с датой вступления в силу (0, если effective_at пустой).
10. what_changed не длиннее 300 знаков: назови главное, без перечисления всех статей.`

func LawSchema() Schema {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	return Schema{
		Name:        LawFunction,
		Description: "Записать структуру закона с цитатами из текста",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"relevant":         map[string]any{"type": "boolean", "description": "касается граждан или ИП"},
				"title":            str("нейтральный заголовок 6–14 слов"),
				"what_changed":     str("что меняется, 1–3 предложения"),
				"who_affected":     str("кого касается, одно предложение"),
				"actions":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "до 3 действий, только если прямо предписаны"},
				"audience_tags":    str("теги аудитории через запятую из списка, обязательно"),
				"effective_at":     str("YYYY-MM-DD или пустая строка"),
				"what_changed_ref": map[string]any{"type": "integer", "description": "номер фрагмента, где сказано, что меняется"},
				"who_affected_ref": map[string]any{"type": "integer", "description": "номер фрагмента про аудиторию"},
				"effective_ref":    map[string]any{"type": "integer", "description": "номер фрагмента с датой вступления в силу или 0"},
			},
			"required": []string{"relevant", "title", "what_changed", "who_affected", "audience_tags", "what_changed_ref", "who_affected_ref"},
		},
	}
}

// LawUser собирает сообщение с нумерованными фрагментами и возвращает отправленные фрагменты (по ним разбирается ответ).
func LawUser(in llm.LawInput) (string, []llm.Fragment) {
	sent := llm.FragmentsForModel(in.Fragments, 7000, 4000)
	var text strings.Builder
	for _, f := range sent {
		fmt.Fprintf(&text, "[%d] %s\n", f.N, f.Text)
	}
	return fmt.Sprintf("Закон: %s (%s)\n\nТекст по фрагментам:\n%s", in.Title, in.Number, text.String()), sent
}

func ParseLaw(args json.RawMessage, sent []llm.Fragment) (llm.LawDraft, error) {
	args = UnwrapArgs(args)
	var raw struct {
		Relevant     *bool    `json:"relevant"`
		Title        string   `json:"title"`
		WhatChanged  string   `json:"what_changed"`
		WhoAffected  string   `json:"who_affected"`
		Actions      []string `json:"actions"`
		AudienceTags string   `json:"audience_tags"`
		EffectiveAt  string   `json:"effective_at"`
		WhatRef      int      `json:"what_changed_ref"`
		WhoRef       int      `json:"who_affected_ref"`
		EffectiveRef int      `json:"effective_ref"`
	}
	if err := json.NewDecoder(bytes.NewReader(args)).Decode(&raw); err != nil {
		return llm.LawDraft{}, fmt.Errorf("аргументы функции не JSON: %w", err)
	}
	if raw.Relevant == nil {
		return llm.LawDraft{}, errors.New("нет поля relevant")
	}
	trim := strings.TrimSpace
	d := llm.LawDraft{
		Relevant: *raw.Relevant, Title: trim(raw.Title), WhatChanged: trim(raw.WhatChanged), WhoAffected: trim(raw.WhoAffected),
		AudienceTags: splitTags(raw.AudienceTags), EffectiveAt: trim(raw.EffectiveAt),
	}
	quote := func(n int) string {
		for _, f := range sent {
			if f.N == n {
				return f.Text
			}
		}
		return ""
	}
	d.Quotes = llm.Quotes{WhatChanged: quote(raw.WhatRef), WhoAffected: quote(raw.WhoRef), EffectiveDay: quote(raw.EffectiveRef)}
	if d.Relevant && (d.Quotes.WhatChanged == "" || d.Quotes.WhoAffected == "") {
		return llm.LawDraft{}, errors.New("указан несуществующий номер фрагмента")
	}
	if d.Quotes.EffectiveDay == "" {
		// Дата без подтверждающего фрагмента не принимается (модель иногда пишет «0000-00-00» вместо пустой строки).
		d.EffectiveAt = ""
	}
	for _, a := range raw.Actions {
		if a = trim(a); a != "" {
			d.Actions = append(d.Actions, a)
		}
	}
	return d, nil
}

func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
