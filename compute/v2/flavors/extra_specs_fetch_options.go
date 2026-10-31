package flavors

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// FlavorExtraSpecsOpts supplies an owned read policy. Nil Microversion keeps
// a selected version or discovers at most 2.61; explicit empty is versionless.
type FlavorExtraSpecsOpts struct {
	Microversion *string
}

// FlavorExtraSpecsOption is one option of FetchExtraSpecs and
// GetExtraSpecsProperty: WithFlavorExtraSpecsOptions, request.Microversion,
// request.Header or request.Custom over *request.Config[FlavorExtraSpecsOpts].
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type FlavorExtraSpecsOption = request.GetCall

var flavorExtraSpecsCalls = calls.Spec[request.Config[FlavorExtraSpecsOpts]]{
	Header:       calls.ConfigHeader[FlavorExtraSpecsOpts],
	Microversion: calls.Pointer(func(c *request.Config[FlavorExtraSpecsOpts]) **string { return &c.Options.Microversion }),
}

func copyFlavorExtraSpecsOptions(value FlavorExtraSpecsOpts) FlavorExtraSpecsOpts {
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return value
}

// WithFlavorExtraSpecsOptions snapshots its pointer now and for each call.
func WithFlavorExtraSpecsOptions(value FlavorExtraSpecsOpts) FlavorExtraSpecsOption {
	owned := copyFlavorExtraSpecsOptions(value)
	return request.Custom(func(config *request.Config[FlavorExtraSpecsOpts]) error {
		config.Options = copyFlavorExtraSpecsOptions(owned)
		return nil
	})
}

func ownFlavorExtraSpecsConfig(config *request.Config[FlavorExtraSpecsOpts]) {
	config.Options = copyFlavorExtraSpecsOptions(config.Options)
	cloudread.OwnReadConfig(config)
}
