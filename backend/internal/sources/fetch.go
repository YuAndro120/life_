package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetcher загружает страницу целиком. Интерфейс нужен, чтобы подменять сеть в тестах.
type Fetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// HTTPFetcher — вежливый клиент: своё имя в User-Agent, таймаут, потолок размера, только http(s).
type HTTPFetcher struct {
	Client    *http.Client
	UserAgent string
	MaxBytes  int64
}

func NewHTTPFetcher(userAgent string) *HTTPFetcher {
	return &HTTPFetcher{
		Client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("слишком много перенаправлений")
				}
				if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
					return errors.New("перенаправление на неподдерживаемую схему")
				}
				return nil
			},
		},
		UserAgent: userAgent,
		MaxBytes:  4 << 20,
	}
}

type StatusError struct{ Code int }

func (e StatusError) Error() string { return fmt.Sprintf("HTTP %d", e.Code) }

func (f *HTTPFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, errors.New("поддерживаются только http и https")
	}
	req.Header.Set("User-Agent", f.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, text/html;q=0.8, */*;q=0.5")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, StatusError{Code: resp.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, f.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > f.MaxBytes {
		return nil, fmt.Errorf("ответ больше %d байт", f.MaxBytes)
	}
	return body, nil
}
