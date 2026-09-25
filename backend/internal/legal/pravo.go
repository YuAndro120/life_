// Package legal собирает федеральные законы: список с publication.pravo.gov.ru, полный текст с kremlin.ru.
// PDF на портале опубликования — сканы без текстового слоя, поэтому текст берём с сайта Президента.
package legal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	PravoBase   = "http://publication.pravo.gov.ru"
	federalLaw  = "82a8bf1c-3bc7-47ed-827f-7affd43a7f27" // тип документа «Федеральный закон»
	pageSize    = 30                                     // портал принимает только 10, 30, 100…
	maxBodySize = 8 << 20
)

// Doc — запись о законе на портале опубликования.
type Doc struct {
	EONumber string    // номер опубликования
	Number   string    // «333-ФЗ»
	Signed   time.Time // дата подписания
	Title    string    // название без кавычек
}

// OfficialURL — страница документа на официальном портале.
func (d Doc) OfficialURL() string { return PravoBase + "/document/" + d.EONumber }

type Pravo struct {
	HTTP *http.Client
	Base string
}

func NewPravo() *Pravo {
	return &Pravo{HTTP: &http.Client{Timeout: 30 * time.Second}, Base: PravoBase}
}

// Laws возвращает федеральные законы, подписанные не раньше from, от новых к старым, не более limit.
func (p *Pravo) Laws(ctx context.Context, from time.Time, limit int) ([]Doc, error) {
	var out []Doc
	for page := 1; len(out) < limit; page++ {
		q := url.Values{
			"block":         {"president"},
			"DocumentTypes": {federalLaw},
			"PageSize":      {fmt.Sprint(pageSize)},
			"Index":         {fmt.Sprint(page)},
			"SignDateFrom":  {from.Format("02.01.2006")},
		}
		var resp struct {
			Items []struct {
				EONumber     string `json:"eoNumber"`
				Number       string `json:"number"`
				DocumentDate string `json:"documentDate"`
				Name         string `json:"name"`
			} `json:"items"`
			PagesTotal int `json:"pagesTotalCount"`
		}
		if err := getJSON(ctx, p.HTTP, p.Base+"/api/Documents?"+q.Encode(), &resp); err != nil {
			return nil, err
		}
		for _, it := range resp.Items {
			signed, err := time.Parse("2006-01-02T15:04:05", it.DocumentDate)
			if err != nil || it.EONumber == "" {
				continue
			}
			out = append(out, Doc{
				EONumber: it.EONumber, Number: strings.TrimSpace(it.Number), Signed: signed,
				Title: strings.Trim(strings.TrimSpace(it.Name), `"«»`),
			})
			if len(out) == limit {
				break
			}
		}
		if page >= resp.PagesTotal || len(resp.Items) == 0 {
			break
		}
	}
	return out, nil
}

func getJSON(ctx context.Context, c *http.Client, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pravo.gov.ru: HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxBodySize)).Decode(v)
}
