package cyborg

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// FieldHeaderCalls is the shared call spec of Cyborg creates: an extension
// body member (request.Field) and an extension header (request.Header).
// Other shared calls are unsupported.
func FieldHeaderCalls[T any]() calls.Spec[request.Config[T]] {
	return calls.Spec[request.Config[T]]{Field: calls.ConfigField[T], Header: calls.ConfigHeader[T]}
}

// HeaderCalls is the shared call spec of Cyborg actions, patches and batch
// deletes without a typed body: an extension header (request.Header) only.
// Other shared calls are unsupported.
var HeaderCalls = calls.Spec[request.Config[struct{}]]{Header: calls.ConfigHeader[struct{}]}
