package client

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/go-multierror"
)

type Simple struct {
	Client interface {
		Get(string) (*http.Response, error)
	}
	ConcurrencyLimit int
}

type validator func(*http.Response) error

// Get attempts gets the http.Response from urls.
// If one of the urls returns an error, the rest of the response and the error will be returned.
//
// Optionally, it accepts an array of validators.
// If one of responses fails to valiate, the rest of the responses and the error will be returned.
func (s *Simple) Get(urls []string, vv ...validator) ([]*http.Response, error) {
	type out struct {
		resp *http.Response
		err  error
	}

	limit := s.ConcurrencyLimit
	if limit == 0 {
		limit = len(urls)
	}

	ch := make(chan out, limit)

	for _, u := range urls {
		go func(u string) {
			resp, err := s.Client.Get(u)
			ch <- out{resp: resp, err: err}
		}(u)
	}

	var (
		rr   []*http.Response
		errs *multierror.Error
	)
	for i := 0; i < len(urls); i++ {
		o := <-ch
		if o.err != nil {
			errs = multierror.Append(errs, o.err)
			continue
		}
		for _, validate := range vv {
			if err := validate(o.resp); err != nil {
				errs = multierror.Append(errs, err)
				continue
			}
		}
		rr = append(rr, o.resp)
	}

	return rr, errs.ErrorOrNil()
}

func ValidateStatusOK(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status %d", resp.StatusCode)
	}
	return nil
}
