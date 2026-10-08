package metadefproperties

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"net/http"
)

// Acknowledgement records an actual DELETE response independently of handling
// errors. Name is nil for a collection deletion. Legacy Delete/DeleteAll return
// only actual204 receipts; owned DeleteRecord also retains handled404 evidence,
// and both owned methods retain actual200..399 receipts.
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
