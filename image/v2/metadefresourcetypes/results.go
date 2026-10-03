package metadefresourcetypes

import (
	"net/http"

	"gophercloudsdk/internal/rest"
)

// Acknowledgement records only an actual DELETE 204 independently of response
// handling errors. A handled 404 never creates an acknowledgement.
type Acknowledgement struct {
	Namespace  string
	Name       *string
	Body       []byte
	Header     http.Header
	StatusCode int
}

func acknowledgement(namespace, name string, response *rest.Response) *Acknowledgement {
	if response == nil || response.StatusCode != http.StatusNoContent {
		return nil
	}
	return &Acknowledgement{Namespace: namespace, Name: copyPointer(&name), Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
