// Package glancecalls holds shared request call handlers of the hand-written
// image/v2 packages. Each package composes them into a calls.Spec for its own
// option states; a control a package leaves out of its Spec fails with
// request.UnsupportedCall before HTTP.
package glancecalls

import (
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
)

// SetHeader returns a request.Header handler that stores one header in the
// map target points at. Existing case-insensitive aliases of key are removed
// and key is stored in canonical form; validation happens when the operation
// prepares its request.
func SetHeader[S any](target func(*S) *map[string]string) func(*S, string, string) error {
	return func(s *S, key, value string) error {
		p := target(s)
		if *p == nil {
			*p = make(map[string]string)
		}
		for old := range *p {
			if strings.EqualFold(old, key) {
				delete(*p, old)
			}
		}
		(*p)[http.CanonicalHeaderKey(key)] = value
		return nil
	}
}

// HeaderSpec returns a Spec that accepts only request.Header, stored with
// SetHeader. Operations whose only shared control is one additional header
// (their merging With<Op>Headers stays a request.Custom option) use it.
func HeaderSpec[S any](target func(*S) *map[string]string) calls.Spec[S] {
	return calls.Spec[S]{Header: SetHeader(target)}
}
