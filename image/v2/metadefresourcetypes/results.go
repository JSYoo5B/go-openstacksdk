package metadefresourcetypes

import (
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Acknowledgement owns an actual deletion response independently of handling
// errors. Bounded Delete acknowledges only204; its handled404 has no receipt.
// DeleteRecord acknowledges all accepted200..399 and default physical404.
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
