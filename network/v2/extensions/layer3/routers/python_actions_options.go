package routers

import (
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
type RouterActionOption func(*RouterActionOpts) error

func copyRouterActionOpts(value RouterActionOpts) RouterActionOpts {
	value.Headers = maps.Clone(value.Headers)
	value.Seed = value.Seed.Clone()
	return value
}

// WithRouterActionOpts snapshots the concrete options and is safe to reuse.
func WithRouterActionOpts(value RouterActionOpts) RouterActionOption {
	owned := copyRouterActionOpts(value)
	return func(config *RouterActionOpts) error { *config = copyRouterActionOpts(owned); return nil }
}

// WithRouterActionHeaders snapshots ordinary HTTP headers. It does not move
// if_match or if_revision keys from the positional body into a header.
func WithRouterActionHeaders(values map[string]string) RouterActionOption {
	owned := maps.Clone(values)
	return func(config *RouterActionOpts) error { config.Headers = maps.Clone(owned); return nil }
}

// WithRouterActionSeed snapshots cached Router attributes. It never supplies
// request JSON, changes the selected service or replaces the literal route ID.
// The response overlays this view; fields absent from an empty ACK stay cached.
func WithRouterActionSeed(value *resource.RawResource) RouterActionOption {
	owned := value.Clone()
	return func(config *RouterActionOpts) error { config.Seed = owned.Clone(); return nil }
}
