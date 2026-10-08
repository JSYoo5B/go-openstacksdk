package remoteconsoles

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

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/microversions"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ConsoleCreateOpts preserves omitted, null and empty console attributes.
// Nova validates the supplied strings; the SDK derives only a missing/falsy
// protocol and checks the selected version for webmks and spice-direct.
type ConsoleCreateOpts struct {
	Protocol request.Optional[string] `json:"protocol,omitzero"`
	Type     request.Optional[string] `json:"type,omitzero"`
	URL      request.Optional[string] `json:"url,omitzero"`
}

type ConsoleCreateOption = request.Option[ConsoleCreateOpts]

func WithConsoleCreateOptions(value ConsoleCreateOpts) ConsoleCreateOption {
	return func(config *request.Config[ConsoleCreateOpts]) error { config.Options = value; return nil }
}
func WithConsoleCreateProtocol(value string) ConsoleCreateOption {
	return WithConsoleCreateProtocolValue(request.Present(value))
}
func WithConsoleCreateProtocolValue(value request.Optional[string]) ConsoleCreateOption {
	return func(config *request.Config[ConsoleCreateOpts]) error { config.Options.Protocol = value; return nil }
}
func WithConsoleCreateType(value string) ConsoleCreateOption {
	return WithConsoleCreateTypeValue(request.Present(value))
}
func WithConsoleCreateTypeValue(value request.Optional[string]) ConsoleCreateOption {
	return func(config *request.Config[ConsoleCreateOpts]) error { config.Options.Type = value; return nil }
}
func WithConsoleCreateURL(value string) ConsoleCreateOption {
	return WithConsoleCreateURLValue(request.Present(value))
}
func WithConsoleCreateURLValue(value request.Optional[string]) ConsoleCreateOption {
	return func(config *request.Config[ConsoleCreateOpts]) error { config.Options.URL = value; return nil }
}
func WithConsoleCreateField(key string, value any) ConsoleCreateOption {
	return request.WithField[ConsoleCreateOpts](key, value)
}
func WithConsoleCreateHeader(key, value string) ConsoleCreateOption {
	return request.WithHeader[ConsoleCreateOpts](key, value)
}

// ConsoleRecord separates seeded attributes from the actual selected response
// object. An accepted unparseable response retains Resource and leaves Wire nil.
// Passive response values never grant authority to route another request.
type ConsoleRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// CreateConsole makes a fixed remote-console POST without a preliminary server
// lookup or legacy-console fallback. Native Create keeps its existing API.
func (a *API) CreateConsole(ctx context.Context, serverID string, options ...ConsoleCreateOption) (*ConsoleRecord, error) {
	fail := func(value *ConsoleRecord, err error) (*ConsoleRecord, error) {
		return value, request.Wrap("CreateConsole", "remoteconsoles", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if err := resource.ID(serverID).Validate(); err != nil {
		return fail(nil, err)
	}
	if !utf8.ValidString(serverID) || strings.IndexFunc(serverID, unicode.IsControl) >= 0 {
		return fail(nil, fmt.Errorf("%w: server ID must be valid text without controls", resource.ErrInvalidOption))
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: remote console API is required", resource.ErrInvalidOption))
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(nil, err)
	}
	guard := func(ctx context.Context) error {
		var replacement error
		if a.client != original {
			replacement = fmt.Errorf("%w: remote console API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(replacement, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	owned := slices.Clone(options)
	guarded := make([]ConsoleCreateOption, len(owned))
	for i, option := range owned {
		apply := option
		guarded[i] = func(config *request.Config[ConsoleCreateOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil console creation option", resource.ErrInvalidOption)
			}
			ownConsoleCreateConfig(config)
			applyErr := apply(config)
			ownConsoleCreateConfig(config)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	config, err := request.Apply(ConsoleCreateOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	seed := map[string]json.RawMessage{}
	var body map[string]any
	if err == nil {
		// Transform a private request copy. Python's dirty map is also a copy;
		// derivation must not replace the Resource's original protocol seed.
		for key, value := range map[string]request.Optional[string]{"protocol": config.Options.Protocol, "type": config.Options.Type, "url": config.Options.URL} {
			if value.IsSet() {
				seed[key], err = json.Marshal(value)
				if err != nil {
					break
				}
			}
		}
		transmitted := config.Options
		protocol, _ := transmitted.Protocol.Get()
		kind, _ := transmitted.Type.Get()
		if protocol == "" && kind != "" {
			if derived, known := map[string]string{"novnc": "vnc", "xvpvnc": "vnc", "spice-html5": "spice", "spice-direct": "spice", "rdp-html5": "rdp", "serial": "serial", "webmks": "mks"}[kind]; known {
				transmitted.Protocol = request.Present(derived)
			} else {
				transmitted.Protocol = request.Null[string]()
			}
		}
		minimum := 0
		if kind == "webmks" {
			minimum = 8
		}
		if kind == "spice-direct" {
			minimum = 99
		}
		if minimum != 0 {
			matches, parseErr := microversions.Matches(source.Client.Microversion, fmt.Sprintf("2.%d", minimum))
			if parseErr != nil || !matches {
				err = errors.Join(fmt.Errorf("%w: console type %q requires selected Compute microversion 2.%d or later (client uses %q)", resource.ErrUnsupported, kind, minimum, source.Client.Microversion), parseErr)
			}
		}
		fields := make(map[string]any)
		for key, value := range map[string]request.Optional[string]{"protocol": transmitted.Protocol, "type": transmitted.Type, "url": transmitted.URL} {
			if value.IsSet() {
				fields[key] = value
			}
		}
		for key, raw := range config.Fields {
			if key == "server_id" || strings.TrimSpace(key) == "" || !utf8.Valid(raw) || !json.Valid(raw) {
				err = errors.Join(err, fmt.Errorf("%w: console extension conflicts with the URI parent or is not valid UTF-8 JSON", resource.ErrInvalidOption))
				break
			}
		}
		if err == nil {
			body, err = request.MergeFieldsFor(map[string]any{"remote_console": fields}, config.Fields, transmitted)
		}
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(nil, err)
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	target := source.Client.ServiceURL("servers", url.PathEscape(serverID), "remote-consoles")
	response, prior := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodPost, target, body, nil, codes...)
	if response == nil {
		return fail(nil, prior)
	}
	result := &ConsoleRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err := errors.Join(prior, guard(ctx)); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	// _translate_response tolerates response.json ValueError. Physical read,
	// Close, context and source failures are never converted into this success.
	if utf8.Valid(response.Body) && json.Valid(response.Body) {
		root, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
		if decodeErr != nil {
			return fail(result, decodeErr)
		}
		wire := root
		if _, present := root.Body["remote_console"]; present {
			wire, decodeErr = rest.Decode(response, "remote_console", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
			if decodeErr != nil {
				return fail(result, decodeErr)
			}
		}
		result.Wire = wire.Clone()
		for _, key := range []string{"protocol", "type", "url"} {
			if raw, present := wire.Body[key]; present {
				seed[key] = bytes.Clone(raw)
			}
		}
	}
	view := &resource.RawResource{Metadata: resource.Metadata{Body: make(map[string]json.RawMessage, 4), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	for _, key := range []string{"protocol", "type", "url"} {
		raw, fieldErr := resource.BodyRecordField(seed, key, resource.BodyFieldJSON)
		if fieldErr != nil {
			return fail(result, response.Fail(cloudread.ContextError(ctx, fieldErr)))
		}
		view.Body[key] = raw
	}
	view.Body["server_id"], _ = json.Marshal(serverID)
	if err := guard(ctx); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	result.Resource = view
	return result, nil
}

func ownConsoleCreateConfig(config *request.Config[ConsoleCreateOpts]) {
	fields := make(map[string]json.RawMessage, len(config.Fields))
	for key, raw := range config.Fields {
		fields[key] = bytes.Clone(raw)
	}
	config.Fields = fields
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	query := make(url.Values, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}
