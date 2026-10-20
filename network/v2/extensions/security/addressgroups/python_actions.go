package addressgroups

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
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AddressActionRecord keeps the declared Source view separate from received HTTP fields.
// Input records the submitted action value; it is never a received field.
type AddressActionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Input          json.RawMessage
}

// AddressActionOpts owns optional request headers and an existing Resource view to retain
// cached fields. Seed does not change the explicit route ID or mutate its input.
type AddressActionOpts struct {
	Headers map[string]string
	Seed    *resource.RawResource
}
type AddressActionOption func(*AddressActionOpts) error

func copyActionOpts(v AddressActionOpts) AddressActionOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Seed = v.Seed.Clone()
	return v
}
func WithAddressActionOpts(value AddressActionOpts) AddressActionOption {
	owned := copyActionOpts(value)
	return func(v *AddressActionOpts) error { *v = copyActionOpts(owned); return nil }
}
func WithAddressActionHeaders(values map[string]string) AddressActionOption {
	owned := maps.Clone(values)
	return func(v *AddressActionOpts) error { v.Headers = maps.Clone(owned); return nil }
}
func WithAddressActionSeed(value *resource.RawResource) AddressActionOption {
	owned := value.Clone()
	return func(v *AddressActionOpts) error { v.Seed = owned.Clone(); return nil }
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

var actionFields = []string{"id", "name", "description", "project_id", "tenant_id", "addresses"}

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
func actionProject(record *AddressActionRecord, fields, unknown map[string]json.RawMessage) error {
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
		if field == "addresses" {
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
func actionReceipt(response *rest.Response, input json.RawMessage) *AddressActionRecord {
	if response == nil {
		return nil
	}
	return &AddressActionRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, Input: bytes.Clone(input)}
}
func actionResponse(response *rest.Response, input json.RawMessage, fields, unknown map[string]json.RawMessage) (*AddressActionRecord, error) {
	record := actionReceipt(response, input)
	if !json.Valid(response.Body) {
		return record, actionProject(record, fields, unknown)
	}
	var root resource.RawResource
	if err := json.Unmarshal(response.Body, &root); err != nil {
		return record, err
	}
	raw := response.Body
	if nested, exists := root.Body["address_group"]; exists {
		raw = nested
	}
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
	// AddressGroup uses Resource._translate_response, normalizing Body aliases
	// and collecting unknown attrs; self is discarded before Resource creation.
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		wire, declared := actionWire(member.Key)
		fields[wire] = bytes.Clone(member.Value)
		if !declared {
			unknown[wire] = bytes.Clone(member.Value)
		}
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
func (a *API) actionRecord(ctx context.Context, id string, value any, route string, options []AddressActionOption) (*AddressActionRecord, error) {
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
	opts := AddressActionOpts{}
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
	response, err := rest.DoJSONGuardedRejections(ctx, &source.Client, guard, http.MethodPut, source.Client.ServiceURL("address-groups", url.PathEscape(id), route), map[string]json.RawMessage{"addresses": raw}, nil, rest.RejectionPolicy{Codes: actionCodes(400, 600), PreserveCleanRetry: true}, actionCodes(200, 400)...)
	if err != nil {
		return actionReceipt(response, raw), err
	}
	record, err := actionResponse(response, raw, fields, unknown)
	if err = errors.Join(err, guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return record, err
}

// AddAddressesRecord sends one envelope-free PUT. Required input stays present even
// when explicitly null, false, zero or empty; server schema validation is kept.
func (a *API) AddAddressesRecord(ctx context.Context, id string, value any, options ...AddressActionOption) (*AddressActionRecord, error) {
	record, err := a.actionRecord(ctx, id, value, "add_addresses", options)
	return record, request.Wrap("AddAddressesRecord", "addressgroups", cloudread.ContextError(ctx, err))
}

// RemoveAddressesRecord uses the Source remove action and the same physical error policy.
func (a *API) RemoveAddressesRecord(ctx context.Context, id string, value any, options ...AddressActionOption) (*AddressActionRecord, error) {
	record, err := a.actionRecord(ctx, id, value, "remove_addresses", options)
	return record, request.Wrap("RemoveAddressesRecord", "addressgroups", cloudread.ContextError(ctx, err))
}
