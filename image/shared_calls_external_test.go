package image_test

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// applyCustom applies one request.Custom option, such as a With<Op>Headers
// option, to state the way the operation's apply loop runs it.
func applyCustom[S any, C request.Call](call C, state *S) error {
	return calls.Translate([]C{call}, calls.Spec[S]{})[0](state)
}
