package compute

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FlavorByIDOpts controls optional enrichment after a successful strict ID GET.
// Omitted GetExtraSpecs is false; an existing inline map needs no extra request.
type FlavorByIDOpts struct {
	GetExtraSpecs bool
}

type FlavorByIDOption = request.Option[FlavorByIDOpts]

func WithFlavorByIDOptions(value FlavorByIDOpts) FlavorByIDOption {
	return request.WithOptions(value)
}

func WithFlavorByIDExtraSpecs(value bool) FlavorByIDOption {
	return func(config *request.Config[FlavorByIDOpts]) error {
		config.Options.GetExtraSpecs = value
		return nil
	}
}

// GetFlavorByID owns cloud's default-false enrichment and strict, GET-only
// identity policy. It reuses the existing typed lookup and native enrichment.
func (s *Service) GetFlavorByID(ctx context.Context, id string, options ...FlavorByIDOption) (*Flavor, error) {
	fail := func(err error) (*Flavor, error) {
		return nil, request.Wrap("GetFlavorByID", "flavors", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if s == nil || s.API == nil || s.API.Flavors == nil {
		return fail(invalid("flavor service is required"))
	}
	api, flavorAPI, client := s.API, s.API.Flavors, s.client
	resources := flavorAPI.Resources
	if api.RawClient() != client || flavorAPI.RawClient() != client {
		return fail(invalid("flavor service components must share their selected client"))
	}
	source, err := cloudread.Capture(ctx, client, "compute")
	if err != nil {
		return fail(err)
	}
	check := func() error {
		var binding error
		if s.API != api || api.Flavors != flavorAPI || s.client != client || api.RawClient() != client || flavorAPI.RawClient() != client || flavorAPI.Resources != resources {
			binding = invalid("flavor service binding changed")
		}
		return errors.Join(binding, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := check(); err != nil {
		return fail(err)
	}
	guarded := make([]FlavorByIDOption, len(options))
	for index, option := range slices.Clone(options) {
		apply := option
		guarded[index] = func(config *request.Config[FlavorByIDOpts]) error {
			if err := check(); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil flavor by-ID option")
			}
			return errors.Join(apply(config), check())
		}
	}
	config, err := request.Apply(FlavorByIDOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, false)
	}
	if err == nil && len(config.Arguments) != 0 {
		err = invalid("flavor by-ID accepts only typed enrichment options")
	}
	if err = errors.Join(err, check()); err != nil {
		return fail(err)
	}
	value, err := flavorAPI.FindIdentity(ctx, id,
		resource.WithIdentityFindFallback(resource.FindFallbackNever),
		resource.WithIdentityFindIgnoreMissing(false),
		resource.WithIdentityFindExtraSpecs(config.Options.GetExtraSpecs))
	if err = errors.Join(err, check()); err != nil {
		return fail(err)
	}
	return value, nil
}
