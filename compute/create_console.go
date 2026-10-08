package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/remoteconsoles"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/microversions"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ConsoleOpts owns the optional protocol of the automatic console operation.
// An unset or null value has the source default None. An explicit empty string
// remains distinct and participates in the modern request's protocol inference.
type ConsoleOpts struct {
	Protocol request.Optional[string]
}

type ConsoleOption = request.Option[ConsoleOpts]

func WithConsoleProtocol(value string) ConsoleOption {
	return WithConsoleProtocolValue(request.Present(value))
}

func WithConsoleProtocolValue(value request.Optional[string]) ConsoleOption {
	return func(config *request.Config[ConsoleOpts]) error {
		config.Options.Protocol = value
		return nil
	}
}

// CreateConsole selects the modern or legacy executor from the endpoint's
// advertised range and the selected version, before making a console request.
// A failed console request is never retried through the other API. Returned
// JSON is a modern source dictionary or the exact legacy console value.
func (s *Service) CreateConsole(ctx context.Context, serverID, consoleType string, options ...ConsoleOption) (json.RawMessage, error) {
	fail := func(err error) (json.RawMessage, error) {
		return nil, request.Wrap("CreateConsole", "compute", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if err := resource.ID(serverID).Validate(); err != nil {
		return fail(err)
	}
	if !utf8.ValidString(serverID) || strings.IndexFunc(serverID, unicode.IsControl) >= 0 {
		return fail(fmt.Errorf("%w: server ID must be valid text without controls", resource.ErrInvalidOption))
	}
	if s == nil || s.API == nil || s.Servers == nil || s.API.Servers == nil || s.API.RemoteConsoles == nil {
		return fail(fmt.Errorf("%w: compute console service is required", resource.ErrInvalidOption))
	}
	original, api, collection := s.client, s.API, s.Servers
	serverAPI, consoleAPI := api.Servers, api.RemoteConsoles
	if api.RawClient() != original || serverAPI.RawClient() != original || consoleAPI.RawClient() != original {
		return fail(fmt.Errorf("%w: compute console components must share their selected client", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(err)
	}
	locationReader := collection.dependencies.CloudLocation
	var changed error
	// This source-only guard does not invoke the outer guard: the composed
	// context carries both, so discovery and child executors cannot recurse.
	guard := func(checkCtx context.Context) error {
		if changed == nil && (s.client != original || s.API != api || s.Servers != collection || api.Servers != serverAPI || api.RemoteConsoles != consoleAPI || api.RawClient() != original || serverAPI.RawClient() != original || consoleAPI.RawClient() != original) {
			changed = fmt.Errorf("%w: compute console source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(checkCtx))
	}
	check := func() error { return errors.Join(guard(ctx), rest.CheckOperationGuard(ctx)) }
	owned := slices.Clone(options)
	guarded := make([]ConsoleOption, len(owned))
	for index, option := range owned {
		apply := option
		guarded[index] = func(config *request.Config[ConsoleOpts]) error {
			if err := check(); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil console option", resource.ErrInvalidOption)
			}
			ownConsoleConfig(config)
			applyErr := apply(config)
			ownConsoleConfig(config)
			return errors.Join(applyErr, check())
		}
	}
	config, err := request.Apply(ConsoleOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, false)
	}
	if err = errors.Join(err, check()); err != nil {
		return fail(err)
	}
	operationCtx := rest.WithOperationGuard(ctx, guard)
	advertised, err := microversions.Read(operationCtx, source, microversions.NovaProfile)
	if err != nil {
		return fail(err)
	}
	withDiscoveryProof := func(err error) error {
		if len(advertised.Responses) != 0 {
			return advertised.Responses[len(advertised.Responses)-1].Fail(cloudread.ContextError(ctx, err))
		}
		return cloudread.ContextError(ctx, err)
	}
	modern, err := microversions.Supports(advertised.Maximum, advertised.Minimum, source.Client.Microversion, "2.6")
	if err != nil {
		return fail(withDiscoveryProof(fmt.Errorf("%w: console version support: %w", resource.ErrInvalidOption, err)))
	}
	ceiling := "2.100"
	if modern {
		ceiling = "2.99"
		minimum := ""
		switch consoleType {
		case "webmks":
			minimum = "2.8"
		case "spice-direct":
			minimum = "2.99"
		}
		if minimum != "" {
			supported, supportErr := microversions.Supports(advertised.Maximum, advertised.Minimum, source.Client.Microversion, minimum)
			if supportErr != nil {
				return fail(withDiscoveryProof(fmt.Errorf("%w: console type version support: %w", resource.ErrInvalidOption, supportErr)))
			}
			if !supported {
				return fail(withDiscoveryProof(fmt.Errorf("%w: console type %q requires advertised and selected Compute support for %s", resource.ErrUnsupported, consoleType, minimum)))
			}
		}
	}
	chosen := source.Client.Microversion
	if chosen == "" {
		chosen, err = microversions.SelectVersion(advertised.Maximum, advertised.Minimum, ceiling)
		if err != nil {
			return fail(withDiscoveryProof(fmt.Errorf("%w: console version selection: %w", resource.ErrInvalidOption, err)))
		}
	}
	if err := check(); err != nil {
		return fail(withDiscoveryProof(err))
	}
	client := source.Client
	client.Microversion = chosen
	if !modern {
		value, err := servers.New(&client).ConsoleURL(operationCtx, serverID, consoleType)
		if err != nil {
			return fail(err)
		}
		return bytes.Clone(value), nil
	}
	location := json.RawMessage("null")
	if locationReader != nil {
		value, locationErr := locationReader()
		if locationErr == nil {
			location, locationErr = value.ForResource(nil, nil)
		}
		if locationErr = errors.Join(locationErr, check()); locationErr != nil {
			return fail(withDiscoveryProof(locationErr))
		}
	}
	protocol := config.Options.Protocol
	if !protocol.IsSet() {
		protocol = request.Null[string]()
	}
	created, err := remoteconsoles.New(&client).CreateConsole(operationCtx, serverID,
		remoteconsoles.WithConsoleCreateType(consoleType), remoteconsoles.WithConsoleCreateProtocolValue(protocol))
	if err != nil {
		return fail(err)
	}
	if created == nil || created.Resource == nil {
		return fail(fmt.Errorf("%w: missing modern console Resource", resource.ErrInvalidOption))
	}
	proof := &rest.Response{Body: created.Envelope, Header: created.Header, StatusCode: created.StatusCode}
	// Resource.to_dict excludes URI fields and includes inherited Body id/name
	// and computed location. Unknown body/location values never replace these.
	view := map[string]json.RawMessage{"location": bytes.Clone(location)}
	for _, key := range []string{"protocol", "type", "url"} {
		view[key] = bytes.Clone(created.Resource.Body[key])
	}
	for _, key := range []string{"id", "name"} {
		view[key] = json.RawMessage("null")
		if created.Wire != nil {
			if raw, present := created.Wire.Body[key]; present {
				view[key] = bytes.Clone(raw)
			}
		}
	}
	result, err := json.Marshal(view)
	if err = errors.Join(err, check()); err != nil {
		return fail(proof.Fail(cloudread.ContextError(ctx, err)))
	}
	return result, nil
}

func ownConsoleConfig(config *request.Config[ConsoleOpts]) {
	fields := make(map[string]json.RawMessage, len(config.Fields))
	for key, raw := range config.Fields {
		fields[key] = bytes.Clone(raw)
	}
	config.Fields = fields
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	query := make(map[string][]string, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}
