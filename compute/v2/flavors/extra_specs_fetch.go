package flavors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FlavorExtraSpecsRequest supplies a string ID or one existing flavor. An
// explicit ID controls only the request route; it does not rewrite existing
// Resource fields. Without it, existing id/name/original_name string data selects
// the route. Flavor and Resource are mutually exclusive concrete inputs.
type FlavorExtraSpecsRequest struct {
	ID       string
	Flavor   *Flavor
	Resource *resource.RawResource
}

// FlavorExtraSpecsRecord separates the updated owned input from the physical
// response. ExtraSpecs preserves the response value as JSON, including null,
// scalars and arrays; an omitted extra_specs is an empty object. Resource uses
// the Source dict descriptor: nonnull nonobjects become empty objects. It keeps
// input metadata, while Wire and the receipt carry this request's metadata.
type FlavorExtraSpecsRecord struct {
	Resource, Wire *resource.RawResource
	ExtraSpecs     json.RawMessage
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func snapshotFlavorExtraSpecsInput(input FlavorExtraSpecsRequest) (string, *resource.RawResource, error) {
	invalid := func(message string) (string, *resource.RawResource, error) {
		return "", nil, fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
	}
	if input.Flavor != nil && input.Resource != nil {
		return invalid("supply one existing flavor representation")
	}
	var seed *resource.RawResource
	switch {
	case input.Resource != nil:
		seed = input.Resource.Clone()
		if seed.Body == nil {
			return invalid("existing flavor Resource fields are required")
		}
		for key, raw := range seed.Body {
			if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
				return invalid("existing flavor fields must contain valid UTF-8 JSON")
			}
		}
	case input.Flavor != nil:
		owned := *input.Flavor
		owned.ExtraSpecs = maps.Clone(owned.ExtraSpecs)
		body, err := json.Marshal(owned)
		if err != nil {
			return "", nil, fmt.Errorf("%w: existing flavor: %w", resource.ErrInvalidOption, err)
		}
		seed = &resource.RawResource{}
		if err := json.Unmarshal(body, seed); err != nil {
			return "", nil, err
		}
		// The native decoder exposes Swap but excludes it from JSON encoding.
		seed.Body["swap"], _ = json.Marshal(owned.Swap)
		for wire, field := range map[string]string{
			"os-flavor-access:is_public": "is_public",
			"OS-FLV-EXT-DATA:ephemeral":  "ephemeral",
		} {
			seed.Body[field] = seed.Body[wire]
			delete(seed.Body, wire)
		}
	default:
		id, _ := json.Marshal(input.ID)
		seed = &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": id}}}
	}
	identity := input.ID
	if identity == "" {
		for _, key := range []string{"id", "name", "original_name"} {
			raw, present := seed.Body[key]
			if !present {
				continue
			}
			truthy, err := cloudfilter.PythonTruthy(raw)
			if err != nil {
				return "", nil, err
			}
			if !truthy {
				continue
			}
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return invalid("existing flavor route identity must be a string")
			}
			identity = value
			break
		}
	}
	if err := resource.ID(identity).Validate(); err != nil {
		return "", nil, err
	}
	if !utf8.ValidString(identity) || strings.IndexFunc(identity, unicode.IsControl) >= 0 {
		return invalid("flavor ID must be valid text without controls")
	}
	return identity, seed, nil
}

type flavorExtraSpecsRead struct {
	ctx      context.Context
	source   *cloudread.Source
	check    func(context.Context) error
	identity string
	seed     *resource.RawResource
	version  string
}

// prepareFlavorExtraSpecsRead snapshots the same input and read policy for the
// full dictionary and single-property endpoints before any option callback.
func (a *API) prepareFlavorExtraSpecsRead(ctx context.Context, input FlavorExtraSpecsRequest, options ...FlavorExtraSpecsOption) (*flavorExtraSpecsRead, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("%w: flavor API is required", resource.ErrInvalidOption)
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(checkCtx context.Context) error {
		if changed == nil && a.client != original {
			changed = fmt.Errorf("%w: flavor API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(checkCtx))
	}
	operationCtx := rest.WithOperationGuard(ctx, guard)
	check := func(checkCtx context.Context) error {
		return errors.Join(guard(checkCtx), rest.CheckOperationGuard(checkCtx))
	}
	identity, seed, err := snapshotFlavorExtraSpecsInput(input)
	if err = errors.Join(err, check(operationCtx)); err != nil {
		return nil, err
	}
	owned := slices.Clone(options)
	guarded := make([]FlavorExtraSpecsOption, len(owned))
	for index, option := range owned {
		apply := option
		guarded[index] = func(config *request.Config[FlavorExtraSpecsOpts]) error {
			if err := check(operationCtx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil extra-specs option", resource.ErrInvalidOption)
			}
			ownFlavorExtraSpecsConfig(config)
			applyErr := apply(config)
			ownFlavorExtraSpecsConfig(config)
			return errors.Join(applyErr, check(operationCtx))
		}
	}
	config, err := request.Apply(FlavorExtraSpecsOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err = errors.Join(err, check(operationCtx)); err != nil {
		return nil, err
	}
	if !hasFlavorExtraSpecsHeader(config.Headers, "Accept") && !hasFlavorExtraSpecsHeader(source.Client.MoreHeaders, "Accept") {
		config.Headers["Accept"] = "application/json"
	}
	chosen, err := selectFlavorExtraSpecsVersion(operationCtx, source, config.Options.Microversion, config.Headers)
	if err = errors.Join(err, check(operationCtx)); err != nil {
		return nil, err
	}
	return &flavorExtraSpecsRead{ctx: operationCtx, source: source, check: check, identity: identity, seed: seed, version: chosen}, nil
}

// read uses the shared versioned, guarded bodyless transport and strict object
// decoder. It preserves the physical receipt for any accepted processing error.
func (read *flavorExtraSpecsRead) read(property ...string) (*resource.RawResource, *rest.Response, error) {
	parts := []string{"flavors", url.PathEscape(read.identity), "os-extra_specs"}
	for _, value := range property {
		parts = append(parts, url.PathEscape(value))
	}
	target := read.source.Client.ServiceURL(parts...)
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	response, prior := microversions.MemberGet(read.ctx, read.source, target, read.version, microversions.NovaProfile, codes...)
	if response == nil {
		return nil, nil, errors.Join(prior, read.check(read.ctx))
	}
	if err := errors.Join(prior, read.check(read.ctx)); err != nil {
		return nil, response, response.Fail(cloudread.ContextError(read.ctx, err))
	}
	wire, err := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
	return wire, response, err
}

// FetchExtraSpecs always reads /flavors/{id}/os-extra_specs, even when the
// supplied flavor already has inline specs. The existing input and selected
// client are never mutated. Native ListExtraSpecs retains its typed ABI.
func (a *API) FetchExtraSpecs(ctx context.Context, input FlavorExtraSpecsRequest, options ...FlavorExtraSpecsOption) (*FlavorExtraSpecsRecord, error) {
	fail := func(value *FlavorExtraSpecsRecord, err error) (*FlavorExtraSpecsRecord, error) {
		return value, request.Wrap("FetchExtraSpecs", "flavors", cloudread.ContextError(ctx, err))
	}
	read, err := a.prepareFlavorExtraSpecsRead(ctx, input, options...)
	if err != nil {
		return fail(nil, err)
	}
	wire, response, err := read.read()
	if response == nil {
		return fail(nil, err)
	}
	result := &FlavorExtraSpecsRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err != nil {
		return fail(result, err)
	}
	result.Wire = wire.Clone()
	specs, present := wire.Body["extra_specs"]
	if !present {
		specs = json.RawMessage(`{}`)
	}
	result.ExtraSpecs = bytes.Clone(specs)
	// Flavor.extra_specs uses Body(type=dict): object and null are retained;
	// any other nonnull value reads as {}. Keep the raw value independently.
	read.seed.Body["extra_specs"] = flavorExtraSpecsView(specs)
	if err := read.check(read.ctx); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(read.ctx, err)))
	}
	result.Resource = read.seed
	return result, nil
}

// The source dict descriptor is shared by full reads, declared flavor views
// and conditional enrichment. Actual response values remain separate.
func flavorExtraSpecsView(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' && !bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage(`{}`)
	}
	return bytes.Clone(trimmed)
}

func hasFlavorExtraSpecsHeader(headers map[string]string, name string) bool {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

func selectFlavorExtraSpecsVersion(ctx context.Context, source *cloudread.Source, override *string, headers map[string]string) (string, error) {
	trial := *source
	trial.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
	preflight := maps.Clone(headers)
	for key := range trial.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			delete(trial.Client.MoreHeaders, key)
		}
	}
	if override == nil && source.Client.Microversion == "" {
		for key := range preflight {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
				delete(preflight, key)
			}
		}
	}
	if err := trial.WithPolicy(ctx, override, preflight); err != nil {
		return "", err
	}
	chosen := source.Client.Microversion
	if override != nil {
		chosen = *override
	} else if chosen == "" {
		advertised, err := microversions.Read(ctx, source, microversions.NovaProfile)
		if err != nil {
			return "", err
		}
		chosen, err = microversions.SelectVersion(advertised.Maximum, advertised.Minimum, "2.61")
		if err != nil {
			err = fmt.Errorf("%w: flavor version selection: %w", resource.ErrInvalidOption, err)
			if len(advertised.Responses) != 0 {
				err = advertised.Responses[len(advertised.Responses)-1].Fail(cloudread.ContextError(ctx, err))
			}
			return "", err
		}
	}
	for key := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			delete(source.Client.MoreHeaders, key)
		}
	}
	if err := source.WithPolicy(ctx, &chosen, headers); err != nil {
		return "", err
	}
	return chosen, rest.CheckOperationGuard(ctx)
}
