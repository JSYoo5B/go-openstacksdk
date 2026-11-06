package routers

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
)

// RouterActionOpts holds optional operation headers. The named Python actions
// have no body defaults, revision argument or per-call microversion argument.
// Headers are a Go extension; authentication and service selection remain owned
// by the configured client. Source action bodies are the positional JSON value.
type RouterActionOpts struct {
	Headers map[string]string
	Seed    *resource.RawResource
}

// RouterActionOption is one option of the extra route and external gateway
// actions: WithRouterActionOpts, WithRouterActionSeed, request.Headers
// (replaces the headers) or request.Custom over *RouterActionOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type RouterActionOption = request.WriteCall

func copyRouterActionOpts(value RouterActionOpts) RouterActionOpts {
	value.Headers = maps.Clone(value.Headers)
	value.Seed = value.Seed.Clone()
	return value
}

// WithRouterActionOpts snapshots the concrete options and is safe to reuse.
func WithRouterActionOpts(value RouterActionOpts) RouterActionOption {
	owned := copyRouterActionOpts(value)
	return request.Custom(func(config *RouterActionOpts) error { *config = copyRouterActionOpts(owned); return nil })
}

// withRouterActionHeaders snapshots ordinary HTTP headers. It does not move
// if_match or if_revision keys from the positional body into a header.
func withRouterActionHeaders(values map[string]string) func(*RouterActionOpts) error {
	owned := maps.Clone(values)
	return func(config *RouterActionOpts) error { config.Headers = maps.Clone(owned); return nil }
}

// WithRouterActionSeed snapshots cached Router attributes. It never supplies
// request JSON, changes the selected service or replaces the literal route ID.
// The response overlays this view; fields absent from an empty ACK stay cached.
func WithRouterActionSeed(value *resource.RawResource) RouterActionOption {
	owned := value.Clone()
	return request.Custom(func(config *RouterActionOpts) error { config.Seed = owned.Clone(); return nil })
}

// routerActionCalls interprets the shared calls of RouterActionOption.
var routerActionCalls = calls.Spec[RouterActionOpts]{
	Headers: neutronfind.Use(withRouterActionHeaders),
}
