package ports

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetRecord fetches one literal port without the find fallback. It reuses the
// declared Port descriptors so workflows observe the same conversion errors
// and keep the physical response separately from the Resource view.
func (a *API) GetRecord(ctx context.Context, identity string, options ...RecordFindOption) (*PortRecord, error) {
	wrap := func(err error) error { return request.Wrap("GetRecord", "ports", cloudread.ContextError(ctx, err)) }
	profile := portListProfile()
	profile.SourceStatusPolicy = true
	engine, err := a.portRecordEngine(ctx, profile)
	if err != nil {
		return nil, wrap(err)
	}
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil port read option"))
		}
		next := copyFindOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	value, err := engine.Get(ctx, identity, internalFindOpts(policy))
	return portListRecord(value), wrap(err)
}
