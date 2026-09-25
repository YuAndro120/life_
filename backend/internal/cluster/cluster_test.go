package cluster

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)

func doc(id int64, source int64, minutes int, text string) Doc {
	return Doc{ID: id, SourceID: source, Text: text, At: t0.Add(time.Duration(minutes) * time.Minute)}
}

// Корпус реалистичных заголовков: три события с разными формулировками и посторонние новости.
var corpus = []Doc{
	doc(1, 1, 0, "Банк России сохранил ключевую ставку на уровне 16% годовых"),
	doc(2, 2, 3, "ЦБ оставил ключевую ставку без изменений — 16% годовых"),
	doc(3, 3, 7, "Банк России принял решение сохранить ключевую ставку 16%, инфляция замедляется"),
	doc(4, 1, 20, "Минфин предложил изменить порядок уплаты авансовых платежей для ИП на УСН"),
	doc(5, 2, 24, "ИП на УСН смогут платить авансовые платежи по новому графику, предложил Минфин"),
	doc(6, 1, 40, "Сборная России сыграла вничью с Сербией в товарищеском матче"),
	doc(7, 3, 45, "Футболисты сборной России не смогли обыграть сербов: 1:1 в товарищеском матче"),
	doc(8, 2, 60, "Банк России ужесточил требования к микрофинансовым организациям"),
	doc(9, 3, 70, "В Москве отключат горячую воду в октябре: опубликован график по районам"),
}

func groupOf(assign []Assignment) map[int64]int64 {
	g := map[int64]int64{}
	for _, a := range assign {
		g[a.DocID] = a.StoryID
	}
	return g
}

func TestSameEventGetsOneStory(t *testing.T) {
	assign, _ := Assign(DefaultConfig(), nil, corpus)
	g := groupOf(assign)
	if !(g[1] == g[2] && g[2] == g[3]) {
		t.Errorf("ставка ЦБ не склеилась: %v", g)
	}
	if g[4] != g[5] {
		t.Errorf("Минфин и ИП на УСН не склеились: %v", g)
	}
	if g[6] != g[7] {
		t.Errorf("матч сборной не склеился: %v", g)
	}
}

func TestDifferentEventsStaySeparate(t *testing.T) {
	assign, _ := Assign(DefaultConfig(), nil, corpus)
	g := groupOf(assign)
	groups := map[int64]bool{g[1]: true, g[4]: true, g[6]: true, g[8]: true, g[9]: true}
	if len(groups) != 5 {
		t.Errorf("должно быть 5 разных сюжетов, получилось %d: %v", len(groups), g)
	}
	if g[8] == g[1] {
		t.Error("«Банк России ужесточил требования к МФО» склеился со ставкой: общие слова «Банк России» не должны хватать")
	}
}

func TestJoinsExistingStory(t *testing.T) {
	existing := []Story{{ID: 42, Docs: corpus[:2]}}
	fresh := []Doc{doc(3, 3, 7, corpus[2].Text), doc(6, 1, 40, corpus[5].Text)}
	assign, _ := Assign(DefaultConfig(), existing, fresh)
	g := groupOf(assign)
	if g[3] != 42 {
		t.Errorf("пост о ставке должен присоединиться к сюжету 42, получил %d", g[3])
	}
	if g[6] == 42 {
		t.Error("матч не должен присоединяться к сюжету о ставке")
	}
	for _, a := range assign {
		if a.DocID == 6 && !a.NewStory {
			t.Error("матч должен создать новый сюжет")
		}
	}
}

func TestStoryWindowExpires(t *testing.T) {
	old := []Story{{ID: 7, Docs: []Doc{doc(1, 1, -60*40, corpus[0].Text)}}} // 40 часов назад
	assign, _ := Assign(DefaultConfig(), old, []Doc{doc(2, 2, 0, corpus[1].Text)})
	if assign[0].StoryID == 7 {
		t.Error("к сюжету старше окна присоединяться нельзя")
	}
}

func TestOrderOfInputDoesNotMatter(t *testing.T) {
	rev := append([]Doc(nil), corpus...)
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	a, _ := Assign(DefaultConfig(), nil, corpus)
	b, _ := Assign(DefaultConfig(), nil, rev)
	ga, gb := groupOf(a), groupOf(b)
	same := func(g map[int64]int64, x, y int64) bool { return g[x] == g[y] }
	for _, pair := range [][2]int64{{1, 2}, {2, 3}, {4, 5}, {6, 7}, {1, 8}, {1, 4}, {6, 9}} {
		if same(ga, pair[0], pair[1]) != same(gb, pair[0], pair[1]) {
			t.Errorf("результат зависит от порядка входа для пары %v", pair)
		}
	}
}

func TestNumbersDistinguishSimilarHeadlines(t *testing.T) {
	docs := []Doc{
		doc(1, 1, 0, "Банк России снизил ключевую ставку до 15% годовых"),
		doc(2, 2, 5, "Банк России повысил ключевую ставку до 18% годовых"),
	}
	cfg := DefaultConfig()
	cfg.Threshold = 0.6
	assign, _ := Assign(cfg, nil, docs)
	if groupOf(assign)[1] == groupOf(assign)[2] {
		t.Error("разные решения (снизил до 15% и повысил до 18%) при строгом пороге не должны склеиваться")
	}
}

func TestEmptyAndSingle(t *testing.T) {
	if a, s := Assign(DefaultConfig(), nil, nil); len(a) != 0 || len(s) != 0 {
		t.Error("пустой вход")
	}
	a, s := Assign(DefaultConfig(), nil, []Doc{doc(1, 1, 0, "!!!")})
	if len(a) != 1 || len(s) != 1 || !a[0].NewStory {
		t.Errorf("пост без слов: %+v", a)
	}
}

func TestScoresAreReported(t *testing.T) {
	assign, _ := Assign(DefaultConfig(), nil, corpus[:2])
	if assign[1].NewStory || assign[1].Score < 0.3 {
		t.Errorf("второй пост должен присоединиться с оценкой >= 0.3: %+v", assign[1])
	}
}

func en(id int64, source int64, minutes int, text string) Doc {
	d := doc(id, source, minutes, text)
	d.Lang = "en"
	return d
}

func TestEnglishHeadlinesFromDifferentOutletsMerge(t *testing.T) {
	docs := []Doc{
		en(1, 1, 0, "NASA delays Artemis III moon landing to 2028 after heat shield review"),
		en(2, 2, 5, "Artemis III moon mission delayed to 2028, NASA says, as heat shield tests continue"),
		en(3, 3, 9, "Mars Sample Return mission cancelled as NASA budget shrinks"),
		en(4, 1, 20, "UK inflation falls to 3.1% in August, ONS says"),
	}
	assign, _ := Assign(DefaultConfig(), nil, docs)
	g := groupOf(assign)
	if g[1] != g[2] {
		t.Errorf("два заголовка про Artemis III не склеились: %v", g)
	}
	if g[3] == g[1] || g[4] == g[1] || g[3] == g[4] {
		t.Errorf("разные события не должны склеиваться: %v", g)
	}
}

func TestDifferentLanguagesNeverMerge(t *testing.T) {
	ru := doc(1, 1, 0, "Банк России сохранил ключевую ставку на уровне 16% годовых")
	ru.Lang = "ru"
	// Заведомо совпадающий по словам «английский» пост с теми же токенами.
	enPost := doc(2, 2, 3, "Банк России сохранил ключевую ставку на уровне 16% годовых")
	enPost.Lang = "en"
	assign, _ := Assign(DefaultConfig(), nil, []Doc{ru, enPost})
	g := groupOf(assign)
	if g[1] == g[2] {
		t.Error("посты на разных языках не должны склеиваться")
	}
}

func TestMissingLangIsCompatible(t *testing.T) {
	a := doc(1, 1, 0, corpus[0].Text) // Lang пуст
	b := doc(2, 2, 3, corpus[1].Text)
	b.Lang = "ru"
	assign, _ := Assign(DefaultConfig(), nil, []Doc{a, b})
	if g := groupOf(assign); g[1] != g[2] {
		t.Error("пустой язык совместим с любым (старые записи)")
	}
}
