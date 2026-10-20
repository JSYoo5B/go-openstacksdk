package cinderrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Negotiate(ctx context.Context, source *cloudread.Source, ceiling string) (string, []*rest.Response, error) {
	return negotiate(ctx, source, ceiling, "")
}

// NegotiateRequired always checks advertised support before accepting a
// selected version. The same advertisement also supplies ordinary negotiation.
func NegotiateRequired(ctx context.Context, source *cloudread.Source, ceiling, required string) (string, []*rest.Response, error) {
	if required == "" {
		return "", nil, invalid("a required Cinder microversion is needed")
	}
	return negotiate(ctx, source, ceiling, required)
}

func negotiate(ctx context.Context, source *cloudread.Source, ceiling, required string) (string, []*rest.Response, error) {
	if err := cloudread.Context(ctx); err != nil {
		return "", nil, err
	}
	if source == nil {
		return "", nil, invalid("Cinder discovery source is required")
	}
	guard := func() error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	if required == "" && source.Client.Microversion != "" {
		return source.Client.Microversion, nil, guard()
	}
	discovery, err := microversions.Read(ctx, source, microversions.CinderProfile)
	pages := discovery.Responses
	if err != nil {
		return "", pages, err
	}
	if discovery.Found {
		wire := pages[len(pages)-1]
		if required != "" {
			if err := requireSupport(discovery.Maximum, discovery.Minimum, source.Client.Microversion, required); err != nil {
				return "", pages, wire.Fail(err)
			}
		}
		version := source.Client.Microversion
		if version == "" {
			version, err = SelectVersion(discovery.Maximum, discovery.Minimum, ceiling)
		}
		if err != nil {
			return "", pages, wire.Fail(err)
		}
		if err := guard(); err != nil {
			return "", pages, wire.Fail(err)
		}
		return version, pages, nil
	}
	if required != "" {
		err := fmt.Errorf("%w: Cinder endpoint does not advertise required microversion %s", resource.ErrUnsupported, required)
		cause := errors.Join(err, discovery.LastRejection, guard())
		if len(pages) != 0 {
			return "", pages, pages[len(pages)-1].Fail(cause)
		}
		return "", pages, cause
	}
	return "", pages, guard()
}
