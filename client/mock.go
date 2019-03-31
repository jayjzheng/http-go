package client

import (
	"io"
	"net/http"
)

type Mock struct {
	Fn      func(*http.Request) (*http.Response, error)
	Invoked bool
}

func (m *Mock) Do(req *http.Request) (*http.Response, error) {
	m.Invoked = true
	return m.Fn(req)
}

func NewMock(status int, body io.ReadCloser, err error) *Mock {
	return &Mock{
		Fn: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Body:       body,
				StatusCode: status,
			}, err
		},
	}
}
