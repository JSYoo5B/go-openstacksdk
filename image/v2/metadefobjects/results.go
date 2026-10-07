package metadefobjects

import (
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"net/http"
)

// Acknowledgement records an actual DELETE 204 independently of response
// handling errors. Name is nil for DeleteAll; a handled 404 has no result.
type Acknowledgement struct {
	Namespace  string
	Name       *string
	Body       []byte
	Header     http.Header
	StatusCode int
}

func acknowledgement(namespace string, name *string, response *rest.Response) *Acknowledgement {
	if response == nil || response.StatusCode != http.StatusNoContent {
		return nil
	}
	return &Acknowledgement{Namespace: namespace, Name: copyPointer(name), Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
