package client

import (
	"fmt"
	"net/http"
)

type Multi struct {
	Client interface {
		Get(string) (*http.Response, error)
	}
	ConcurrencyLimit int
}

type validator func(*http.Response) error

type Response struct {
	*http.Response
	Err error
}

// Get attempts to fetch from urls, and returns a <-chan of Response.
func (m *Multi) Get(urls []string, vv ...validator) <-chan Response {
	limit := m.ConcurrencyLimit
	if limit == 0 {
		limit = len(urls)
	}

	ch := make(chan Response, limit)

	for _, u := range urls {
		go func(u string) {
			resp, err := m.Client.Get(u)
			if err != nil {
				ch <- Response{Err: err}
				return
			}
			for _, validate := range vv {
				if err := validate(resp); err != nil {
					ch <- Response{Err: err}
					return
				}
			}
			ch <- Response{Response: resp}
		}(u)
	}

	return ch
}

func ValidateStatusOK(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status %d", resp.StatusCode)
	}
	return nil
}
