package flavors

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// FlavorExtraSpecsOpts supplies an owned read policy. Nil Microversion keeps
// a selected version or discovers at most 2.61; explicit empty is versionless.
type FlavorExtraSpecsOpts struct {
	Microversion *string
}

type FlavorExtraSpecsOption = request.Option[FlavorExtraSpecsOpts]

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
	return func(config *request.Config[FlavorExtraSpecsOpts]) error {
		config.Options = copyFlavorExtraSpecsOptions(owned)
		return nil
	}
}

func WithFlavorExtraSpecsMicroversion(value string) FlavorExtraSpecsOption {
	return func(config *request.Config[FlavorExtraSpecsOpts]) error {
		owned := value
		config.Options.Microversion = &owned
		return nil
	}
}

func WithFlavorExtraSpecsHeader(key, value string) FlavorExtraSpecsOption {
	return request.WithHeader[FlavorExtraSpecsOpts](key, value)
}

func ownFlavorExtraSpecsConfig(config *request.Config[FlavorExtraSpecsOpts]) {
	config.Options = copyFlavorExtraSpecsOptions(config.Options)
	cloudread.OwnReadConfig(config)
}
