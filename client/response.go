package client

import "net/http"

type Response struct {
	*http.Response
	Err error
}
