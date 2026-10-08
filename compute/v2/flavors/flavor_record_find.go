package flavors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FlavorFindOpts controls a member read and its detailed list fallback. Nil
// IgnoreMissing defaults to true. GetExtraSpecs defaults to false and enriches
// only a uniquely resolved flavor whose projected extra_specs is falsey.
// Nil Microversion retains a selected version or discovers at most 2.61;
// explicit empty requests a versionless read without changing the client.
type FlavorFindOpts struct {
	GetExtraSpecs bool
	IgnoreMissing *bool
	Limit         int
	Marker        string
	MaxItems      int
	Paginated     *bool
	Microversion  *string
}

type FlavorFindOption = request.Option[FlavorFindOpts]

func copyFlavorFindOptions(value FlavorFindOpts) FlavorFindOpts {
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return value
}

// WithFlavorFindOptions replaces typed fields while preserving separately
// supplied headers, wire queries and semantic filters.
func WithFlavorFindOptions(value FlavorFindOpts) FlavorFindOption {
	owned := copyFlavorFindOptions(value)
	return func(config *request.Config[FlavorFindOpts]) error {
		config.Options = copyFlavorFindOptions(owned)
		return nil
	}
}
func WithFlavorFindExtraSpecs(value bool) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error { config.Options.GetExtraSpecs = value; return nil }
}
func WithFlavorFindIgnoreMissing(value bool) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error {
		owned := value
		config.Options.IgnoreMissing = &owned
		return nil
	}
}
func WithFlavorFindLimit(value int) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error { config.Options.Limit = value; return nil }
}
func WithFlavorFindMarker(value string) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error { config.Options.Marker = value; return nil }
}

// WithFlavorFindMaxItems caps raw fallback rows before local Body filtering.
func WithFlavorFindMaxItems(value int) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error { config.Options.MaxItems = value; return nil }
}
func WithFlavorFindPaginated(value bool) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error {
		owned := value
		config.Options.Paginated = &owned
		return nil
	}
}
func WithFlavorFindMicroversion(value string) FlavorFindOption {
	return func(config *request.Config[FlavorFindOpts]) error {
		owned := value
		config.Options.Microversion = &owned
		return nil
	}
}
func WithFlavorFindHeader(key, value string) FlavorFindOption {
	return request.WithHeader[FlavorFindOpts](key, value)
}
func WithFlavorFindQuery(key, value string) FlavorFindOption {
	return request.WithQuery[FlavorFindOpts](key, value)
}

// WithFlavorFindFilter sends the original attribute spelling to member GET,
// then classifies it as a server query or local Body field for list fallback.
func WithFlavorFindFilter(field string, value any) FlavorFindOption {
	return flavorFilterOption[FlavorFindOpts](resource.WithFilter(field, value))
}

// WithFlavorFindFilters replaces only semantic filters. Nil clears them.
func WithFlavorFindFilters(values map[string]any) FlavorFindOption {
	return flavorFilterOption[FlavorFindOpts](resource.WithFilters(values))
}

func validateFlavorRecordIdentity(identity string) error {
	if strings.TrimSpace(identity) == "" || !utf8.ValidString(identity) || identity == "." || identity == ".." {
		return fmt.Errorf("%w: flavor identity must be nonempty UTF-8 literal text", resource.ErrInvalidOption)
	}
	for _, char := range identity {
		if unicode.IsControl(char) {
			return fmt.Errorf("%w: flavor identity must not contain controls", resource.ErrInvalidOption)
		}
	}
	return nil
}

func fetchFlavorRecord(ctx context.Context, source *cloudread.Source, check func(context.Context) error, identity, version string, query url.Values, seedFields map[string]json.RawMessage) (*FlavorRecord, error) {
	target := source.Client.ServiceURL("flavors", url.PathEscape(identity))
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	response, prior := microversions.MemberGet(ctx, source, target, version, microversions.NovaProfile, flavorRecordCodes()...)
	if response == nil {
		return nil, errors.Join(prior, check(ctx))
	}
	result := &FlavorRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	fail := func(err error) (*FlavorRecord, error) {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	if err := errors.Join(prior, check(ctx)); err != nil {
		return fail(err)
	}
	seed := make(map[string]json.RawMessage, len(seedFields)+1)
	for key, raw := range seedFields {
		seed[key] = bytes.Clone(raw)
	}
	seed["id"], _ = json.Marshal(identity)
	// Resource.fetch tolerates JSON ValueError. A parsed response and its
	// selected flavor member must still be objects. Physical failures have
	// already returned with the accepted receipt and never enter this branch.
	if utf8.Valid(response.Body) && json.Valid(response.Body) {
		root, err := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
		if err != nil {
			return result, err
		}
		wire := root
		selected := json.RawMessage(response.Body)
		if member, present := root.Body["flavor"]; present {
			wire, err = rest.Decode(response, "flavor", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
			if err != nil {
				return result, err
			}
			selected = member
		}
		result.Wire = wire.Clone()
		fields, err := normalizedFlavorRecordFields(wire.Body, selected)
		if err != nil {
			return fail(err)
		}
		maps.Copy(seed, fields)
	}
	view, err := projectFlavorRecord(seed, resource.Metadata{Header: response.Header, StatusCode: response.StatusCode})
	if err != nil {
		return fail(err)
	}
	if err := check(ctx); err != nil {
		return fail(err)
	}
	result.Resource = view
	return result, nil
}

// FindFlavor always attempts an escaped member GET first. A clean native
// 400, 403 or 404 falls back to a detailed list with exact ID-or-name matching.
// It never sends an automatic name hint or the list-only is_public default to
// member GET. Missing is ignored by default; duplicate and list errors remain
// errors. Native Get, Find and the existing typed collection are unchanged.
func (a *API) FindFlavor(ctx context.Context, nameOrID string, options ...FlavorFindOption) (*FlavorRecord, error) {
	fail := func(value *FlavorRecord, err error) (*FlavorRecord, error) {
		return value, request.Wrap("FindFlavor", "flavors", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if err := validateFlavorRecordIdentity(nameOrID); err != nil {
		return fail(nil, err)
	}
	source, guard, err := flavorRecordSource(ctx, a)
	if err != nil {
		return fail(nil, err)
	}
	operationCtx := rest.WithOperationGuard(ctx, guard)
	check := func(checkCtx context.Context) error {
		return errors.Join(guard(checkCtx), rest.CheckOperationGuard(checkCtx))
	}
	owned := slices.Clone(options)
	config, err := prepareFlavorReadConfig(operationCtx, check, FlavorFindOpts{}, copyFlavorFindOptions, owned)
	if err != nil {
		return fail(nil, err)
	}
	value := copyFlavorFindOptions(config.Options)
	filters, err := capturedFlavorFilters(config.Arguments)
	if err != nil {
		return fail(nil, err)
	}
	selection, err := resource.PrepareFilterSelection(flavorRecordFilterDescriptor(), filters...)
	if err != nil {
		return fail(nil, err)
	}
	if _, present, err := selection.Attribute("id"); err != nil {
		return fail(nil, err)
	} else if present {
		return fail(nil, fmt.Errorf("%w: semantic id conflicts with the fixed find identity", resource.ErrInvalidOption))
	}
	attributes, err := selection.OriginalAttributes()
	if err != nil {
		return fail(nil, err)
	}
	seed, err := normalizedFlavorRecordFields(attributes, nil)
	if err != nil {
		return fail(nil, err)
	}
	listValue := FlavorListOpts{Limit: value.Limit, Marker: value.Marker, MaxItems: value.MaxItems, Paginated: value.Paginated, Microversion: value.Microversion}
	p, err := flavorReadParameters(listValue, config.Query, config.Headers, selection)
	if err != nil {
		return fail(nil, err)
	}
	query, err := baseFlavorQuery(listValue, config.Query)
	if err != nil {
		return fail(nil, err)
	}
	original, err := selection.OriginalQuery()
	if err != nil {
		return fail(nil, err)
	}
	for key, values := range original {
		if previous, present := query[key]; present && !slices.Equal(previous, values) {
			return fail(nil, fmt.Errorf("%w: member query %q conflicts with typed or raw query", resource.ErrInvalidOption, key))
		}
		query[key] = slices.Clone(values)
	}
	if p.headers == nil {
		p.headers = make(map[string]string)
	}
	if !hasFlavorExtraSpecsHeader(p.headers, "Accept") && !hasFlavorExtraSpecsHeader(source.Client.MoreHeaders, "Accept") {
		p.headers["Accept"] = "application/json"
	}
	chosen, err := selectFlavorExtraSpecsVersion(operationCtx, source, value.Microversion, p.headers)
	if err = errors.Join(err, check(operationCtx)); err != nil {
		return fail(nil, err)
	}
	// Discovery and explicit overrides belong to this prepared operation;
	// fallback must not infer a different version from the original client.
	p.microversion = &chosen
	var acceptedPartial *FlavorRecord
	get := func(readCtx context.Context, identity string, query url.Values) (*FlavorRecord, error) {
		result, err := fetchFlavorRecord(readCtx, source, check, identity, chosen, query, seed)
		if result != nil && result.StatusCode >= 200 && result.StatusCode < 400 {
			acceptedPartial = result
		}
		return result, err
	}
	collection := resource.NewCollection(resource.Adapter[FlavorRecord]{
		Kind: "flavors", IdentityFind: true, ValidateID: validateFlavorRecordIdentity, IdentityDirectGet: validateFlavorRecordIdentity,
		Get: func(readCtx context.Context, identity string) (*FlavorRecord, error) {
			return get(readCtx, identity, nil)
		},
		GetIdentityQuery: get,
		IdentityResponseID: func(value *FlavorRecord) (string, error) {
			if value == nil || value.Resource == nil {
				return "", fmt.Errorf("%w: flavor Resource is required", resource.ErrInvalidOption)
			}
			return flavorRecordIdentity(value), nil
		},
		Name: flavorRecordName,
		Iterate: func(readCtx context.Context, _ url.Values) iter.Seq2[*FlavorRecord, error] {
			return listFlavorRecordsPrepared(readCtx, source, check, p, true)
		},
	})
	result, err := collection.FindIdentity(operationCtx, nameOrID, resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query, IgnoreMissing: value.IgnoreMissing}))
	if err != nil {
		return fail(acceptedPartial, err)
	}
	if err := check(operationCtx); err != nil {
		return fail(result, err)
	}
	if result != nil && value.GetExtraSpecs {
		result, err = enrichFlavorRecord(operationCtx, source, check, chosen, result)
		if err != nil {
			return fail(result, err)
		}
	}
	if err := check(operationCtx); err != nil {
		return fail(result, err)
	}
	return result, nil
}
