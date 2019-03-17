package transport

import (
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/pkg/errors"
)

type Debugger struct {
	Writer    io.Writer
	Transport http.RoundTripper
}

func (d Debugger) RoundTrip(req *http.Request) (*http.Response, error) {
	b, err := httputil.DumpRequest(req, true)
	if err != nil {
		d.mustWriteError(errors.Wrap(err, "DumpRequest"))
	}
	d.mustWrite(b)

	t := d.Transport
	if t == nil {
		t = http.DefaultTransport
	}

	resp, err := t.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	b, err = httputil.DumpResponse(resp, true)
	if err != nil {
		d.mustWriteError(errors.Wrap(err, "DumpResponse"))
	}
	d.mustWrite(b)

	return resp, nil
}

func (d Debugger) mustWrite(b []byte) {
	if _, err := d.Writer.Write(b); err != nil {
		panic(err)
	}
}

func (d Debugger) mustWriteError(err error) {
	d.mustWrite([]byte(err.Error()))
}
