package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ServerConsoleOpts distinguishes lookup from a supplied dictionary and an
// omitted length from explicit false, zero, or any other finite JSON value.
type ServerConsoleOpts struct{ Length, Server *json.RawMessage }
type ServerConsoleOption func(*ServerConsoleOpts) error

func WithServerConsoleOpts(value ServerConsoleOpts) ServerConsoleOption {
	owned := copyServerConsoleOpts(value)
	return func(c *ServerConsoleOpts) error { *c = copyServerConsoleOpts(owned); return nil }
}
func WithServerConsoleLength(value int) ServerConsoleOption {
	raw, _ := json.Marshal(value)
	return WithServerConsoleLengthJSON(raw)
}
func WithServerConsoleLengthJSON(value json.RawMessage) ServerConsoleOption {
	owned := bytes.Clone(value)
	return func(c *ServerConsoleOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		c.Length = &raw
		return nil
	}
}
func WithoutServerConsoleLength() ServerConsoleOption {
	return func(c *ServerConsoleOpts) error { c.Length = nil; return nil }
}
func WithServerConsoleServerJSON(value json.RawMessage) ServerConsoleOption {
	owned := bytes.Clone(value)
	return func(c *ServerConsoleOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		c.Server = &raw
		return nil
	}
}

// WithServerConsoleServerRecord owns the current descriptor dictionary now.
// Its actual lookup receipt remains separate from this explicitly supplied seed.
func WithServerConsoleServerRecord(value *servers.ServerFindRecord) ServerConsoleOption {
	var raw json.RawMessage
	var err error
	if value == nil || value.Resource == nil {
		err = consoleCloudInvalid("supplied server record requires its descriptor Resource")
	} else {
		raw, err = json.Marshal(value.Resource.Body)
	}
	if err != nil {
		return func(*ServerConsoleOpts) error { return err }
	}
	return WithServerConsoleServerJSON(raw)
}
func WithoutServerConsoleServer() ServerConsoleOption {
	return func(c *ServerConsoleOpts) error { c.Server = nil; return nil }
}
func copyServerConsoleOpts(value ServerConsoleOpts) ServerConsoleOpts {
	if value.Length != nil {
		raw := json.RawMessage(bytes.Clone(*value.Length))
		value.Length = &raw
	}
	if value.Server != nil {
		raw := json.RawMessage(bytes.Clone(*value.Server))
		value.Server = &raw
	}
	return value
}

// ServerConsoleResult separates the lookup/explicit seed from actual action
// proof. Value preserves arbitrary output JSON; HasOutput distinguishes a
// missing output key from an explicitly null value. Unsupported means that a
// clean action HTTP 400 produced the Source empty-string result.
type ServerConsoleResult struct {
	Value                  json.RawMessage
	HasOutput, Unsupported bool
	Server                 *servers.ServerFindRecord
	Seed                   json.RawMessage
	Action                 *servers.CloudConsoleRecord
}

// GetServerConsole owns the bare server lookup and console action. A supplied
// server dictionary bypasses lookup. Missing/ambiguous servers and faulty HTTP
// 400 responses remain errors; only a clean console-action 400 becomes "".
func (s *Service) GetServerConsole(ctx context.Context, nameOrID string, options ...ServerConsoleOption) (*ServerConsoleResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerConsole", "compute", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if s == nil || s.API == nil || s.API.Servers == nil || s.Servers == nil {
		return nil, wrap(consoleCloudInvalid("compute server console service is required"))
	}
	selected, api, collection, leaf := s.client, s.API, s.Servers, s.API.Servers
	if api.RawClient() != selected || leaf.RawClient() != selected || collection.client != selected {
		return nil, wrap(consoleCloudInvalid("compute console components must share their selected client"))
	}
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, wrap(err)
	}
	// The private reader owns headers before any caller option can mutate the
	// selected shared client. Original bindings remain guarded independently.
	reader := servers.New(&source.Client)
	var changed error
	ownGuard := func(c context.Context) error {
		if changed == nil && (s.client != selected || s.API != api || s.Servers != collection || api.Servers != leaf || api.RawClient() != selected || leaf.RawClient() != selected || collection.client != selected) {
			changed = consoleCloudInvalid("compute console source changed")
		}
		return errors.Join(changed, source.Guard(c))
	}
	ctx = rest.WithOperationGuard(ctx, ownGuard)
	check := func() error { return errors.Join(ownGuard(ctx), rest.CheckOperationGuard(ctx)) }
	config := ServerConsoleOpts{}
	for _, apply := range slices.Clone(options) {
		if err := check(); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(consoleCloudInvalid("nil server console option"))
		}
		next := copyServerConsoleOpts(config)
		if err := errors.Join(apply(&next), check()); err != nil {
			return nil, wrap(err)
		}
		config = copyServerConsoleOpts(next)
	}
	result := &ServerConsoleResult{}
	var fields map[string]json.RawMessage
	if config.Server != nil {
		raw := bytes.TrimSpace(*config.Server)
		if len(raw) == 0 || raw[0] != '{' || !utf8.Valid(raw) || !json.Valid(raw) {
			return result, wrap(consoleCloudInvalid("supplied server requires a JSON dictionary"))
		}
		if err := json.Unmarshal(raw, &fields); err != nil {
			return result, wrap(err)
		}
		result.Seed = bytes.Clone(raw)
	} else {
		locationReader := collection.dependencies.CloudLocation
		var findOptions []servers.RecordFindOption
		if locationReader != nil {
			location, e := locationReader()
			if err := errors.Join(e, check()); err != nil {
				return result, wrap(err)
			}
			findOptions = append(findOptions, servers.WithRecordFindLocation(location))
		}
		result.Server, err = reader.FindRecord(ctx, nameOrID, findOptions...)
		if err = errors.Join(err, check()); err != nil {
			return result, wrap(err)
		}
		if result.Server != nil && result.Server.Resource != nil {
			fields = result.Server.Resource.Body
		}
	}
	if len(fields) == 0 {
		return result, wrap(fmt.Errorf("%w: console requested for invalid server", resource.ErrNotFound))
	}
	rawID, present := fields["id"]
	if !present {
		return result, wrap(consoleCloudInvalid("supplied server dictionary requires id"))
	}
	// Proxy._get_resource interprets a dictionary ID as a new Server seed.
	// This is one conversion, not a recursive constructor chain.
	if trimmed := bytes.TrimSpace(rawID); len(trimmed) > 0 && trimmed[0] == '{' {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &nested); err != nil {
			return result, wrap(err)
		}
		rawID = nested["id"]
	}
	id := ""
	truth, err := cloudfilter.PythonJSONTruthy(rawID)
	if err == nil && truth {
		id, err = cloudfilter.PythonString(rawID)
	}
	if err = errors.Join(err, check()); err != nil {
		return result, wrap(err)
	}
	var actionOptions []servers.CloudConsoleOption
	if config.Length != nil {
		actionOptions = append(actionOptions, servers.WithCloudConsoleLengthJSON(*config.Length))
	}
	result.Action, err = reader.ConsoleOutputRecord(ctx, id, actionOptions...)
	err = errors.Join(err, check())
	if consoleCloudClean400(err) {
		result.Unsupported = true
		result.Value = json.RawMessage(`""`)
		return result, nil
	}
	if err != nil {
		return result, wrap(err)
	}
	result.Value = bytes.Clone(result.Action.Value)
	result.HasOutput = result.Action.HasOutput
	return result, nil
}
func consoleCloudInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func consoleCloudClean400(err error) bool {
	switch value := err.(type) {
	case gophercloud.ErrUnexpectedResponseCode:
		return value.Actual == 400
	case *resource.OperationError:
		return consoleCloudClean400(value.Cause)
	case *gophercloud.ErrErrorAfterReauthentication:
		return consoleCloudClean400(value.ErrOriginal)
	case interface{ Unwrap() []error }:
		causes := value.Unwrap()
		return len(causes) == 1 && consoleCloudClean400(causes[0])
	}
	return false
}
