package trunks

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
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TrunkActionRecord keeps the declared Source view separate from received HTTP fields.
// Input records the submitted action value; it is never a received field.
type TrunkActionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Input          json.RawMessage
}

// SubportActionOpts owns optional request headers and an existing Resource view to retain
// cached fields. Seed does not change the explicit route ID or mutate its input.
type SubportActionOpts struct {
	Headers map[string]string
	Seed    *resource.RawResource
}
type SubportActionOption func(*SubportActionOpts) error

func copyActionOpts(v SubportActionOpts) SubportActionOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Seed = v.Seed.Clone()
	return v
}
func WithSubportActionOpts(value SubportActionOpts) SubportActionOption {
	owned := copyActionOpts(value)
	return func(v *SubportActionOpts) error { *v = copyActionOpts(owned); return nil }
}
func WithSubportActionHeaders(values map[string]string) SubportActionOption {
	owned := maps.Clone(values)
	return func(v *SubportActionOpts) error { v.Headers = maps.Clone(owned); return nil }
}
func WithSubportActionSeed(value *resource.RawResource) SubportActionOption {
	owned := value.Clone()
	return func(v *SubportActionOpts) error { v.Seed = owned.Clone(); return nil }
}
func actionInvalid(format string, values ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, values...))
}
func actionID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return actionInvalid("action identifier must be nonblank UTF-8 text")
	}
	for _, character := range id {
		if unicode.IsControl(character) {
			return actionInvalid("action identifier has controls")
		}
	}
	return nil
}
func actionFieldsClone(values map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		owned[key] = bytes.Clone(raw)
	}
	return owned
}

var actionFields = []string{"id", "name", "description", "project_id", "tenant_id", "is_admin_state_up", "port_id", "status", "sub_ports", "tags"}

func actionWire(key string) (string, bool) {
	if wire, known := neutronBodyFieldNames[key]; known {
		return wire, true
	}
	for _, wire := range neutronBodyFieldNames {
		if key == wire {
			return key, true
		}
	}
	return key, false
}
func actionListDescriptor(raw json.RawMessage) (json.RawMessage, error) {
	if !json.Valid(raw) {
		return nil, actionInvalid("list descriptor must contain JSON")
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] == '[' {
		return bytes.Clone(raw), nil
	}
	return json.RawMessage(append(append([]byte{'['}, raw...), ']')), nil
}
func actionSeed(id string, seed *resource.RawResource) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	fields, unknown := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	if seed != nil {
		for key, raw := range seed.Body {
			if key == "location" {
				continue
			}
			if !utf8.Valid(raw) || !json.Valid(raw) {
				return nil, nil, actionInvalid("seed %q must contain UTF-8 JSON", key)
			}
			wire, declared := actionWire(key)
			if _, exists := fields[wire]; exists {
				return nil, nil, actionInvalid("seed has conflicting aliases for %q", wire)
			}
			fields[wire] = bytes.Clone(raw)
			if !declared {
				unknown[wire] = bytes.Clone(raw)
			}
		}
	}
	fields["id"], _ = json.Marshal(id)
	return fields, unknown, nil
}
func actionProject(record *TrunkActionRecord, fields, unknown map[string]json.RawMessage) error {
	body := actionFieldsClone(unknown)
	for _, field := range actionFields {
		wire, _ := actionWire(field)
		raw, err := resource.BodyRecordField(fields, wire, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		if field == "project_id" {
			if _, present := fields[wire]; !present {
				raw, err = resource.BodyRecordField(fields, "tenant_id", resource.BodyFieldJSON)
			}
		}
		if field == "tags" {
			if _, present := fields[wire]; !present {
				raw = json.RawMessage(`[]`)
			}
		}
		if field == "is_admin_state_up" {
			raw, err = jsonfilter.BooleanJSON(raw)
		}
		if field == "tags" || field == "sub_ports" {
			raw, err = actionListDescriptor(raw)
		}
		if err != nil {
			return err
		}
		body[field] = bytes.Clone(raw)
	}
	body["location"] = json.RawMessage("null")
	record.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: record.Header.Clone(), StatusCode: record.StatusCode}}
	return nil
}
func actionReceipt(response *rest.Response, input json.RawMessage) *TrunkActionRecord {
	if response == nil {
		return nil
	}
	return &TrunkActionRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, Input: bytes.Clone(input)}
}
func actionResponse(response *rest.Response, input json.RawMessage, fields, unknown map[string]json.RawMessage) (*TrunkActionRecord, error) {
	record := actionReceipt(response, input)
	if !json.Valid(response.Body) {
		return record, actionInvalid("trunk action requires a JSON response object")
	}
	raw := response.Body
	if !utf8.Valid(raw) {
		return record, actionInvalid("action response must be UTF-8 JSON")
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return record, err
	}
	wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
	record.Wire = wire.Clone()
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return record, err
	}
	// Trunk's custom method directly updates wire _body keys. It does not
	// normalize aliases, unwrap a trunk envelope or collect new unknown attrs.
	for _, member := range members {
		fields[member.Key] = bytes.Clone(member.Value)
	}
	return record, actionProject(record, fields, unknown)
}
func actionCodes(start, end int) []int {
	codes := make([]int, end-start)
	for i := range codes {
		codes[i] = start + i
	}
	return codes
}

// actionList converts JSON-domain iterables as Python list(value). Scalars and
// null are not iterable; objects contribute ordered keys, strings codepoints.
func actionList(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, actionInvalid("subports must be an iterable")
	}
	switch trimmed[0] {
	case '[':
		return bytes.Clone(raw), nil
	case '{':
		members, err := cloudfilter.ObjectMembers(raw)
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(members))
		for _, member := range members {
			keys = append(keys, member.Key)
		}
		return json.Marshal(keys)
	case '"':
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		chars := make([]string, 0, len(value))
		for _, character := range value {
			chars = append(chars, string(character))
		}
		return json.Marshal(chars)
	default:
		return nil, actionInvalid("subports must be an iterable")
	}
}
func (a *API) actionRecord(ctx context.Context, id string, value any, route string, asList bool, options []SubportActionOption) (*TrunkActionRecord, error) {
	if a == nil {
		return nil, actionInvalid("action API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, err
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = actionInvalid("selected action client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err = actionID(id); err != nil {
		return nil, err
	}
	raw, encodeErr := json.Marshal(value)
	if err = errors.Join(encodeErr, guard(ctx)); err != nil {
		return nil, err
	}
	if !utf8.Valid(raw) {
		return nil, actionInvalid("action value must contain UTF-8 JSON")
	}
	raw = bytes.Clone(raw)
	if asList {
		raw, err = actionList(raw)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, err
		}
	}
	opts := SubportActionOpts{}
	for _, apply := range slices.Clone(options) {
		if err = guard(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, actionInvalid("nil action option")
		}
		next := copyActionOpts(opts)
		if err = errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, err
		}
		opts = copyActionOpts(next)
	}
	fields, unknown, err := actionSeed(id, opts.Seed)
	if err == nil {
		err = source.WithPolicy(ctx, nil, opts.Headers)
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuardedRejections(ctx, &source.Client, guard, http.MethodPut, source.Client.ServiceURL("trunks", url.PathEscape(id), route), map[string]json.RawMessage{"sub_ports": raw}, nil, rest.RejectionPolicy{Codes: actionCodes(400, 600), PreserveCleanRetry: true}, actionCodes(200, 400)...)
	if err != nil {
		return actionReceipt(response, raw), err
	}
	record, err := actionResponse(response, raw, fields, unknown)
	if err = errors.Join(err, guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return record, err
}

// AddSubportsRecord sends one envelope-free PUT. Required input stays present even
// when explicitly null, false, zero or empty; server schema validation is kept.
func (a *API) AddSubportsRecord(ctx context.Context, id string, value any, options ...SubportActionOption) (*TrunkActionRecord, error) {
	record, err := a.actionRecord(ctx, id, value, "add_subports", false, options)
	return record, request.Wrap("AddSubportsRecord", "trunks", cloudread.ContextError(ctx, err))
}

// RemoveSubportsRecord uses the Source remove action and the same physical error policy.
func (a *API) RemoveSubportsRecord(ctx context.Context, id string, value any, options ...SubportActionOption) (*TrunkActionRecord, error) {
	record, err := a.actionRecord(ctx, id, value, "remove_subports", true, options)
	return record, request.Wrap("RemoveSubportsRecord", "trunks", cloudread.ContextError(ctx, err))
}
