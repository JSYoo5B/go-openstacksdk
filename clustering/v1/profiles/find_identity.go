package profiles

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FindOpts controls automatic name/ID lookup. Nil IgnoreMissing keeps the
// Senlin proxy default of returning nil when no resource matches.
type FindOpts = senlin.FindOpts

// FindOption is one option of FindIdentity: WithFindOptions, WithFindFallback,
// request.IgnoreMissing, request.Header, request.Microversion (both phases;
// explicit empty selects the server default) or request.Custom over
// *request.Config[FindOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type FindOption = request.FindCall

func WithFindOptions(value FindOpts) FindOption { return request.Custom(senlin.Snapshot(value)) }
func WithFindFallback(value resource.FindFallbackPolicy) FindOption {
	return request.Custom(senlin.WithFindFallback(value))
}

// FindIdentity tries the direct Senlin route first, then searches all advertised
// list pages for an exact ID or name match according to the fallback policy.
// Explicit resource.ID/resource.Name lookup remains available through Find.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...FindOption) (*Profile, error) {
	return senlin.FindIdentity(ctx, a.RawClient(), spec,
		func(value *Profile, id, name string) { value.ID, value.Name = id, name },
		identity, calls.Options(options, senlin.FindCalls)...)
}
