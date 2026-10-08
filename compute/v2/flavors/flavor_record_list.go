package flavors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"slices"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/microversions"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func flavorRecordSource(ctx context.Context, api *API) (*cloudread.Source, func(context.Context) error, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, nil, err
	}
	var original *gophercloud.ServiceClient
	if api != nil {
		original = api.client
	}
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return nil, nil, err
	}
	outer := rest.OperationGuard(ctx)
	var changed error
	check := func(checkCtx context.Context) error {
		if changed == nil && (api == nil || api.client != original) {
			changed = fmt.Errorf("%w: flavor API source changed", resource.ErrInvalidOption)
		}
		var outerErr error
		if outer != nil {
			outerErr = outer(checkCtx)
		}
		return errors.Join(changed, source.Guard(checkCtx), outerErr)
	}
	if err := check(ctx); err != nil {
		return nil, nil, err
	}
	return source, check, nil
}

func flavorRecordCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	return codes
}

// Enrichment uses the same prepared extra-specs reader without recapturing a
// source, applying options again or performing discovery for each flavor.
func enrichFlavorRecord(ctx context.Context, source *cloudread.Source, check func(context.Context) error, version string, value *FlavorRecord) (*FlavorRecord, error) {
	if value == nil || value.Resource == nil {
		return value, fmt.Errorf("%w: flavor Resource is required for enrichment", resource.ErrInvalidOption)
	}
	if err := check(ctx); err != nil {
		return value, err
	}
	filled, err := cloudfilter.PythonTruthy(value.Resource.Body["extra_specs"])
	if err != nil || filled {
		return value, err
	}
	identity, seed, err := snapshotFlavorExtraSpecsInput(FlavorExtraSpecsRequest{Resource: value.Resource})
	if err != nil {
		return value, err
	}
	read := &flavorExtraSpecsRead{ctx: ctx, source: source, check: check, identity: identity, seed: seed, version: version}
	wire, response, err := read.read()
	if response == nil {
		return value, err
	}
	enriched := &FlavorExtraSpecsRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	result := *value
	result.Enrichment = enriched
	if err != nil {
		return &result, err
	}
	specs, present := wire.Body["extra_specs"]
	if !present {
		specs = json.RawMessage(`{}`)
	}
	enriched.Wire = wire.Clone()
	enriched.ExtraSpecs = bytes.Clone(specs)
	seed.Body["extra_specs"] = flavorExtraSpecsView(specs)
	if err := check(ctx); err != nil {
		return &result, response.Fail(cloudread.ContextError(ctx, err))
	}
	enriched.Resource = seed
	result.Resource = seed.Clone()
	return &result, nil
}

func listFlavorRecordsPrepared(ctx context.Context, source *cloudread.Source, check func(context.Context) error, parameters flavorListParameters, versionReady bool) iter.Seq2[*FlavorRecord, error] {
	return func(yield func(*FlavorRecord, error) bool) {
		if err := check(ctx); err != nil {
			yield(nil, err)
			return
		}
		chosen := source.Client.Microversion
		if versionReady && parameters.microversion != nil {
			chosen = *parameters.microversion
		}
		if !versionReady {
			var err error
			chosen, err = selectFlavorExtraSpecsVersion(ctx, source, parameters.microversion, parameters.headers)
			if err != nil {
				yield(nil, err)
				return
			}
		}
		if err := check(ctx); err != nil {
			yield(nil, err)
			return
		}
		path := "flavors/detail"
		if !parameters.details {
			path = "flavors"
		}
		descriptor := flavorRecordFilterDescriptor()
		collection := resource.NewCollection(resource.Adapter[FlavorRecord]{
			Kind: "flavors", BodyFilterFields: descriptor.Body, BodyFilterValue: flavorRecordFilterValue,
			IterateControlled: func(ctx context.Context, _ url.Values, _ resource.ListControl) iter.Seq2[*FlavorRecord, error] {
				return func(yield func(*FlavorRecord, error) bool) {
					var origin *rest.Response
					selected := rest.CollectionSpec[flavorListRecord]{
						Client: &source.Client, Path: path, Kind: "flavors", PluralKey: "flavors", Metadata: func(value *flavorListRecord) *resource.Metadata { return flavorRecordMetadata(&value.FlavorRecord) },
						Validate: check, SourceGuard: check, ListCodes: flavorRecordCodes(),
						ValidateResponse: func(response *rest.Response) error { origin = response; return check(ctx) },
						ValidateItem: func(value *flavorListRecord) error {
							if err := check(ctx); err != nil {
								return err
							}
							return errors.Join(prepareFlavorListRecord(&value.FlavorRecord), check(ctx))
						},
						ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
							return microversions.MemberGet(ctx, source, target, chosen, microversions.NovaProfile, codes...)
						},
						Paging: rest.PagePolicy[flavorListRecord]{LinkKeys: []string{"links", "flavors_links"}, NextKey: "next", HTTPLink: true, DictionaryLinks: true,
							MarkerFallback: true, Marker: func(value *flavorListRecord) (string, error) { return flavorRecordMarker(&value.FlavorRecord) }, MarkerOnShortPage: true, AllowFirstServerLimit: true,
							MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
					}
					for value, err := range rest.ListWithControl(ctx, selected, parameters.query, parameters.control) {
						var record *FlavorRecord
						if value != nil {
							record = &value.FlavorRecord
						}
						if !yield(record, err) {
							return
						}
						if err != nil {
							return
						}
						if err := check(ctx); err != nil {
							if origin != nil {
								err = origin.Fail(cloudread.ContextError(ctx, err))
							}
							yield(nil, err)
							return
						}
					}
				}
			},
		})
		body := make(map[string]any, len(parameters.selection.Body))
		// json.RawMessage, rather than []byte, retains the actual predicate.
		for key, raw := range parameters.selection.Body {
			body[key] = json.RawMessage(bytes.Clone(raw))
		}
		for value, err := range collection.List(ctx, resource.WithBodyFilters(body)) {
			if err == nil && parameters.getExtraSpecs {
				value, err = enrichFlavorRecord(ctx, source, check, chosen, value)
			}
			if !yield(value, err) {
				return
			}
			if err != nil {
				return
			}
			if err := check(ctx); err != nil {
				yield(nil, err)
				return
			}
		}
	}
}

// ListRecords is lazy and owns each logical iteration. Body filtering and the
// raw row cap precede conditional enrichment; break stops additional work.
func (a *API) ListRecords(ctx context.Context, options ...FlavorListOption) iter.Seq2[*FlavorRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*FlavorRecord, error) bool) {
		wrap := func(err error) error { return request.Wrap("ListRecords", "flavors", cloudread.ContextError(ctx, err)) }
		source, check, err := flavorRecordSource(ctx, a)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		operationCtx := rest.WithOperationGuard(ctx, check)
		parameters, err := prepareFlavorList(operationCtx, check, owned)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		if parameters.headers == nil {
			parameters.headers = make(map[string]string)
		}
		if !hasFlavorExtraSpecsHeader(parameters.headers, "Accept") && !hasFlavorExtraSpecsHeader(source.Client.MoreHeaders, "Accept") {
			parameters.headers["Accept"] = "application/json"
		}
		for value, err := range listFlavorRecordsPrepared(operationCtx, source, check, parameters, false) {
			if !yield(value, wrap(err)) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}
