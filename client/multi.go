package client

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
)

type Multi struct {
	Client interface {
		Do(*http.Request) (*http.Response, error)
	}
	ConcurrencyLimit int
}

type validator func(*http.Response) error

// Do attempts to make the http requests and returns a <-chan of *Response.
// Optionally, it accepts a list of validators for *http.Response.
func (m *Multi) Do(ctx context.Context, rr []*http.Request, vv ...validator) <-chan *Response {
	in := m.generate(rr)
	limit := m.limit(len(rr))

	cs := make([]<-chan *Response, limit)
	for i := 0; i < limit; i++ {
		cs[i] = m.do(ctx, in, vv)
	}

	return m.merge(ctx, cs...)
}

type handler func(*Response) error

// Handle drains the resp channel and invokes fn on reach *Response.
func (m *Multi) Handle(ctx context.Context, resp <-chan *Response, fn handler) error {
	var errs *multierror.Error

	for r := range resp {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if r.Err != nil {
			errs = multierror.Append(errs, r.Err)
			continue
		}

		if err := fn(r); err != nil {
			errs = multierror.Append(errs, r.Err)
			continue
		}
	}

	return errs.ErrorOrNil()
}

func (m *Multi) generate(rr []*http.Request) <-chan *http.Request {
	out := make(chan *http.Request)

	go func() {
		defer close(out)
		for _, req := range rr {
			out <- req
		}
	}()

	return out
}

func (m *Multi) do(ctx context.Context, rr <-chan *http.Request, vv []validator) <-chan *Response {
	out := make(chan *Response)
	go func() {
		defer close(out)
		for req := range rr {
			select {
			case <-ctx.Done():
				return
			default:
			}
			resp, err := m.Client.Do(req)
			if err != nil {
				out <- &Response{Err: err}
				continue
			}
			for _, validate := range vv {
				if err := validate(resp); err != nil {
					out <- &Response{Err: err}
					continue
				}
			}
			out <- &Response{Response: resp}
		}
	}()
	return out
}

func (m *Multi) merge(ctx context.Context, cs ...<-chan *Response) <-chan *Response {
	var wg sync.WaitGroup
	wg.Add(len(cs))

	out := make(chan *Response)

	for _, c := range cs {
		go func(c <-chan *Response) {
			defer wg.Done()
			for r := range c {
				select {
				case out <- r:
				case <-ctx.Done():
					return
				}
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func (m *Multi) limit(count int) int {
	if m.ConcurrencyLimit == 0 {
		return count
	}
	return m.ConcurrencyLimit
}

func ValidateStatusOK(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status %d", resp.StatusCode)
	}
	return nil
}
