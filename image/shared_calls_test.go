package image

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// applyCall applies one shared call to state through spec, as the operation's
// own apply loop does.
func applyCall[S any, C request.Call](spec calls.Spec[S], call C, state *S) error {
	return calls.Translate([]C{call}, spec)[0](state)
}
