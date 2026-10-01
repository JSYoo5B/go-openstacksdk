package profiles

import (
	"context"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/resource"
)

// FindOpts controls automatic name/ID lookup. Nil IgnoreMissing keeps the
// Senlin proxy default of returning nil when no resource matches.
type FindOpts = senlin.FindOpts
type FindOption = senlin.FindOption

func WithFindOptions(value FindOpts) FindOption   { return senlin.Snapshot(value) }
func WithFindIgnoreMissing(value bool) FindOption { return senlin.WithFindIgnoreMissing(value) }
func WithFindFallback(value resource.FindFallbackPolicy) FindOption {
	return senlin.WithFindFallback(value)
}
func WithFindHeader(key, value string) FindOption  { return senlin.WithFindHeader(key, value) }
func WithFindMicroversion(value string) FindOption { return senlin.WithFindMicroversion(value) }

// FindIdentity tries the direct Senlin route first, then searches all advertised
// list pages for an exact ID or name match according to the fallback policy.
// Explicit resource.ID/resource.Name lookup remains available through Find.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...FindOption) (*Profile, error) {
	return senlin.FindIdentity(ctx, a.RawClient(), spec,
		func(value *Profile, id, name string) { value.ID, value.Name = id, name },
		identity, options...)
}
