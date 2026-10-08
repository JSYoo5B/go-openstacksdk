package flavors

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"

	"github.com/JSYoo5B/gophercloudsdk/request"
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
	query := make(url.Values, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	fields := make(map[string]json.RawMessage, len(config.Fields))
	for key, raw := range config.Fields {
		fields[key] = bytes.Clone(raw)
	}
	config.Fields = fields
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}
