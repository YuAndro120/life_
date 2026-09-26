package legal

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Типы документов «Закон» и «Областной закон» в блоке «ОГВ Субъектов РФ» портала опубликования.
var regionalLawTypes = []string{"8910e61f-371d-490c-a1b8-aa7b8a5a48be", "e880fa45-e4f4-4db9-bba0-847666adebab"}

// KnownRegions — коды субъектов, которые есть в приложении (совпадают со списком iOS).
var KnownRegions = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range strings.Fields("01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55 56 57 58 59 60 61 62 63 64 65 66 67 68 69 70 71 72 73 74 75 76 77 78 79 82 83 86 87 89 92") {
		m[c] = true
	}
	return m
}()

// RegionalDoc — закон субъекта РФ: только официальные данные с портала (текст на портале — скан, пересказ не делается).
type RegionalDoc struct {
	Doc
	Region string
	Tags   []string
	Who    string
}

// RegionalLaws возвращает законы субъектов, подписанные не раньше from (от новых к старым), не более limit.
func (p *Pravo) RegionalLaws(ctx context.Context, from time.Time, limit int) ([]Doc, error) {
	var out []Doc
	for _, typeID := range regionalLawTypes {
		for page := 1; page <= 40; page++ {
			q := url.Values{
				"block": {"subjects"}, "DocumentTypes": {typeID}, "PageSize": {"100"},
				"Index": {fmt.Sprint(page)}, "SignDateFrom": {from.Format("02.01.2006")},
			}
			var resp struct {
				Items []struct {
					EONumber     string `json:"eoNumber"`
					Number       string `json:"number"`
					DocumentDate string `json:"documentDate"`
					Name         string `json:"name"`
				} `json:"items"`
			}
			if err := getJSON(ctx, p.HTTP, p.Base+"/api/Documents?"+q.Encode(), &resp); err != nil {
				return nil, err
			}
			stop := len(resp.Items) == 0
			for _, it := range resp.Items {
				signed, err := time.Parse("2006-01-02T15:04:05", it.DocumentDate)
				if err != nil || it.EONumber == "" {
					continue
				}
				if signed.Before(from) {
					stop = true
					continue
				}
				out = append(out, Doc{EONumber: it.EONumber, Number: strings.TrimSpace(it.Number), Signed: signed, Title: strings.Trim(strings.TrimSpace(it.Name), `"«»`)})
			}
			if stop || len(out) >= limit*20 {
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Signed.After(out[j].Signed) })
	return out, nil
}

var excluded = regexp.MustCompile(`(?i)бюджет|наделени[а-яё]* органов|о награжден|о присвоени|границ|преобразовани|административно-территориальн|о структуре|о правительстве|о наказах|о почётн|о почетн`)

type rule struct {
	re   *regexp.Regexp
	tags []string
	who  string
}

// Правила по названию закона: без модели, по ключевым словам. Нет совпадения — закон не показываем.
var regionalRules = []rule{
	{regexp.MustCompile(`(?i)транспортн[а-яё]+ налог`), []string{"transport:driver"}, "водителей и владельцев транспорта"},
	{regexp.MustCompile(`(?i)налог[а-яё]* на имущество физических|земельн[а-яё]+ налог|налог на имущество.*граждан`), []string{"housing:owner"}, "владельцев жилья и земли"},
	{regexp.MustCompile(`(?i)профессиональн[а-яё]+ доход|самозанят`), []string{"work:selfemployed"}, "самозанятых"},
	{regexp.MustCompile(`(?i)патентн[а-яё]+ систем`), []string{"work:ip"}, "ИП"},
	{regexp.MustCompile(`(?i)упрощ[её]нн[а-яё]+ систем`), []string{"work:ip_usn"}, "ИП на упрощённой системе"},
	{regexp.MustCompile(`(?i)малого и среднего предпринимательств|поддержк[а-яё]+ предпринимател`), []string{"work:ip"}, "предпринимателей"},
	{regexp.MustCompile(`(?i)капитальн[а-яё]+ ремонт|жилищн[а-яё]+ кодекс|жилищно-коммунальн|коммунальн[а-яё]+ услуг|тариф[а-яё]+ на (тепл|вод|электр|газ)`), []string{"housing:owner", "housing:renter"}, "жильцов и владельцев жилья"},
	{regexp.MustCompile(`(?i)ипотек|жилищн[а-яё]+ субсид|социальн[а-яё]+ найм`), []string{"housing:mortgage", "housing:owner"}, "тех, кто покупает или получает жильё"},
	{regexp.MustCompile(`(?i)об образовании|в сфере образования|общего образования|профессионального образования|высшего образования|студент|стипенд`), []string{"work:student"}, "студентов и учащихся"},
	{regexp.MustCompile(`(?i)алкогольн[а-яё]+|розничн[а-яё]+ продаж`), []string{"sells:alcohol", "sells:goods"}, "продавцов алкоголя и розницы"},
	{regexp.MustCompile(`(?i)такси|пассажирск[а-яё]+ перевозк|парковк|дорожн[а-яё]+ движени`), []string{"transport:driver", "sells:transport"}, "водителей и перевозчиков"},
	{regexp.MustCompile(`(?i)торгов[а-яё]+ деятельност|нестационарн[а-яё]+ торгов|ярмарк`), []string{"sells:goods", "work:ip"}, "торгующих"},
}

// ClassifyRegional подбирает теги аудитории по названию закона региона; ok = false, если закон не подходит.
func ClassifyRegional(d Doc) (tags []string, who string, ok bool) {
	if excluded.MatchString(d.Title) {
		return nil, "", false
	}
	seen := map[string]bool{}
	var whos []string
	for _, r := range regionalRules {
		if r.re.MatchString(d.Title) {
			for _, t := range r.tags {
				if !seen[t] {
					seen[t] = true
					tags = append(tags, t)
				}
			}
			whos = append(whos, r.who)
		}
	}
	if len(tags) == 0 {
		return nil, "", false
	}
	return tags, "Касается: " + strings.Join(whos, ", "), true
}

// RegionOf — код региона из номера публикации (первые две цифры).
func RegionOf(eoNumber string) string {
	if len(eoNumber) < 2 || !KnownRegions[eoNumber[:2]] {
		return ""
	}
	return eoNumber[:2]
}

// RegionalStore — хранилище для сборщика региональных законов.
type RegionalStore interface {
	Seen(ctx context.Context, eoNumber string) (bool, error)
	SaveRegional(ctx context.Context, d RegionalDoc) error
}

type RegionalReport struct{ Listed, Seen, Relevant, Saved int }

// RunRegional собирает региональные законы за период: без модели, только официальные данные и теги по названию.
// Результат — черновики (verified = false): в API они попадут после `lawtool approve-regional`.
func RunRegional(ctx context.Context, p *Pravo, store RegionalStore, from time.Time, limit int) (RegionalReport, error) {
	var rep RegionalReport
	docs, err := p.RegionalLaws(ctx, from, limit)
	if err != nil {
		return rep, err
	}
	rep.Listed = len(docs)
	for _, d := range docs {
		region := RegionOf(d.EONumber)
		if region == "" {
			continue
		}
		tags, who, ok := ClassifyRegional(d)
		if !ok {
			continue
		}
		rep.Relevant++
		if seen, err := store.Seen(ctx, d.EONumber); err != nil {
			return rep, err
		} else if seen {
			rep.Seen++
			continue
		}
		if rep.Saved >= limit {
			break
		}
		if err := store.SaveRegional(ctx, RegionalDoc{Doc: d, Region: region, Tags: tags, Who: who}); err != nil {
			return rep, err
		}
		rep.Saved++
	}
	return rep, nil
}
