package metadefnamespaces

import (
	"gophercloudsdk/internal/rest"
	"net/http"
)

// Acknowledgement records an actual namespace DELETE 204 independently of
// errors while handling that response. A handled missing 404 has no result.
type Acknowledgement struct {
	Namespace  string
	Body       []byte
	Header     http.Header
	StatusCode int
}

func acknowledgement(namespace string, response *rest.Response) *Acknowledgement {
	if response == nil || response.StatusCode != http.StatusNoContent {
		return nil
	}
	return &Acknowledgement{Namespace: namespace, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
