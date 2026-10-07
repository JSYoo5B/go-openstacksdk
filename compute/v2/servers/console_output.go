package servers

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
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ConsoleOutputOpts distinguishes an omitted length from an explicit zero.
// Length is sent literally; Nova validates its permitted values.
type ConsoleOutputOpts struct {
	Length *int `json:"length,omitempty"`
}

type ConsoleOutputOption = request.Option[ConsoleOutputOpts]

// WithConsoleOutputOptions owns the pointer now and for each application.
func WithConsoleOutputOptions(value ConsoleOutputOpts) ConsoleOutputOption {
	owned := cloneConsoleOutputOptions(value)
	return func(config *request.Config[ConsoleOutputOpts]) error {
		config.Options = cloneConsoleOutputOptions(owned)
		return nil
	}
}

func WithConsoleOutputLength(value int) ConsoleOutputOption {
	return func(config *request.Config[ConsoleOutputOpts]) error {
		copy := value
		config.Options.Length = &copy
		return nil
	}
}

// WithConsoleOutputField adds JSON extensions inside os-getConsoleOutput.
// Declared length must be set with the concrete length options.
func WithConsoleOutputField(key string, value any) ConsoleOutputOption {
	return request.WithField[ConsoleOutputOpts](key, value)
}

// ConsoleOutput owns the optional length and returns the native output string
// projection. It makes one action request without a preliminary server lookup.
// Generated ShowConsoleOutput retains its native options and behavior.
func (a *API) ConsoleOutput(ctx context.Context, serverID string, options ...ConsoleOutputOption) (string, error) {
	fail := func(err error) (string, error) {
		return "", request.Wrap("ConsoleOutput", "servers", cloudread.ContextError(ctx, err))
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
	if a == nil {
		return fail(fmt.Errorf("%w: server API is required", resource.ErrInvalidOption))
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != original {
			changed = fmt.Errorf("%w: server API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	owned := slices.Clone(options)
	guarded := make([]ConsoleOutputOption, len(owned))
	for i, option := range owned {
		apply := option
		guarded[i] = func(config *request.Config[ConsoleOutputOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil console output option", resource.ErrInvalidOption)
			}
			ownConsoleOutputConfig(config)
			applyErr := apply(config)
			ownConsoleOutputConfig(config)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	config, err := request.Apply(ConsoleOutputOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	var body map[string]any
	if err == nil {
		fields := make(map[string]any)
		if config.Options.Length != nil {
			fields["length"] = *config.Options.Length
		}
		for key, raw := range config.Fields {
			if strings.TrimSpace(key) == "" || !utf8.Valid(raw) || !json.Valid(raw) {
				err = fmt.Errorf("%w: console extension must have a key and valid UTF-8 JSON", resource.ErrInvalidOption)
				break
			}
		}
		if err == nil {
			body, err = request.MergeFieldsFor(map[string]any{"os-getConsoleOutput": fields}, config.Fields, config.Options)
		}
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(err)
	}

	// Server._action accepts final status<400. The existing native action still
	// accepts only 200; this owned lane retains strict response parsing evidence.
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	target := source.Client.ServiceURL("servers", url.PathEscape(serverID), "action")
	response, prior := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodPost, target, body, map[string]string{"Accept": ""}, codes...)
	if response == nil {
		return fail(prior)
	}
	_, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
	var output struct {
		Output string `json:"output"`
	}
	if decodeErr == nil {
		decodeErr = json.Unmarshal(response.Body, &output)
	}
	if err := errors.Join(prior, decodeErr, guard(ctx)); err != nil {
		return fail(response.Fail(cloudread.ContextError(ctx, err)))
	}
	return output.Output, nil
}

func cloneConsoleOutputOptions(value ConsoleOutputOpts) ConsoleOutputOpts {
	if value.Length != nil {
		length := *value.Length
		value.Length = &length
	}
	return value
}

func ownConsoleOutputConfig(config *request.Config[ConsoleOutputOpts]) {
	config.Options = cloneConsoleOutputOptions(config.Options)
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
