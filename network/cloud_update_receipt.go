package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// UpdateReceipt is the owned evidence of one accepted Neutron PUT issued by a
// cloud update helper (UpdateRouterRecord, UpdateSubnetRecord,
// UpdatePortRecord and the three firewall update helpers).
//
// Envelope, Header and StatusCode are the actual accepted response. Body is
// the resource object selected the way the pinned Python
// Resource._translate_response selects it: the member named by the resource
// key when present, otherwise the whole top-level object, without a "self"
// member. Unknown members are preserved as raw JSON.
//
// BodyIgnored reports Python's ValueError branch: an empty, truncated or
// otherwise unparsable accepted body leaves the existing resource state
// unchanged and is not an error.
//
// Merged is the resulting Python resource state keyed by wire name: the
// lookup's Python body attributes (SourceBody: canonical-only members such as
// "is_admin_state_up" are stored under their wire key and canonical/wire
// collisions keep Python's ordered last-member-wins result), overlaid with
// every effective update attribute (the same values Python assigns to the
// found Resource before commit), overlaid with Body translated through the
// same Resource._consume_attrs mapping. Body itself keeps the actual response
// member names. Its Header and StatusCode are those of the PUT response.
//
// ProjectionError is non-nil when some Merged members cannot be decoded into
// the helper's native typed model (for example a raw caller string for a
// boolean field). The update itself still succeeded, as in Python, so the
// helper returns no error; its native model omits exactly those members (or
// is nil if the rest still cannot be decoded). Merged remains the
// authoritative post-commit view.
type UpdateReceipt struct {
	Envelope        json.RawMessage
	Header          http.Header
	StatusCode      int
	Body            map[string]json.RawMessage
	BodyIgnored     bool
	Merged          *resource.RawResource
	ProjectionError error
}

// cloudUpdateCodes mirrors Python raise_from_response: every status below 400
// that reaches the SDK is an accepted commit.
func cloudUpdateCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

// cloudUpdatePut sends exactly one guarded PUT {key: sent} and translates the
// accepted response into an UpdateReceipt. A rejected PUT returns no receipt.
// An accepted response whose body cannot be read returns the partial receipt
// (without Merged) and the read error. A parsable non-object body, or a
// resource-key member that is not an object, is an error after remote success
// because Python raises AttributeError/TypeError there; the receipt is kept.
// The request is never replayed.
// cloudUpdateLookup is the Python body-attribute state of a found resource
// plus the resource's own response-body mapping.
type cloudUpdateLookup struct {
	state     map[string]json.RawMessage
	translate func(json.RawMessage) (map[string]json.RawMessage, error)
}

// cloudUpdateSource prefers the Source state captured with the lookup and
// falls back to the actual wire row only when no Source state exists.
func cloudUpdateSource(state map[string]json.RawMessage, translate func(json.RawMessage) (map[string]json.RawMessage, error), wire *resource.RawResource) cloudUpdateLookup {
	if state == nil && wire != nil {
		state = maps.Clone(wire.Body)
		delete(state, "self")
	}
	if state == nil {
		state = map[string]json.RawMessage{}
	}
	return cloudUpdateLookup{state: state, translate: translate}
}

func cloudUpdatePut(ctx context.Context, client *gophercloud.ServiceClient, guard func(context.Context) error, endpoint, key string, lookup cloudUpdateLookup, effective, sent map[string]json.RawMessage) (*UpdateReceipt, error) {
	payload := make(map[string]json.RawMessage, len(sent))
	for name, raw := range sent {
		payload[name] = bytes.Clone(raw)
	}
	response, err := rest.DoJSONGuarded(ctx, client, guard, http.MethodPut, endpoint, map[string]any{key: payload}, nil, cloudUpdateCodes()...)
	if response == nil {
		return nil, err
	}
	receipt := &UpdateReceipt{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err != nil {
		return receipt, err
	}
	body, object, ignored, cause := cloudUpdateBody(response.Body, key)
	if cause != nil {
		return receipt, response.Fail(cause)
	}
	receipt.Body, receipt.BodyIgnored = body, ignored
	translated := body
	if lookup.translate != nil && !ignored {
		if translated, cause = lookup.translate(object); cause != nil {
			return receipt, response.Fail(cause)
		}
	}
	merged := make(map[string]json.RawMessage, len(lookup.state)+len(effective)+len(translated))
	for _, layer := range []map[string]json.RawMessage{lookup.state, effective, translated} {
		for name, raw := range layer {
			merged[name] = bytes.Clone(raw)
		}
	}
	receipt.Merged = &resource.RawResource{Metadata: resource.Metadata{Body: merged, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	return receipt, nil
}

// cloudUpdateBody returns the selected object's members (without "self") and
// the selected object's raw JSON for ordered Source translation.
func cloudUpdateBody(raw []byte, key string) (map[string]json.RawMessage, json.RawMessage, bool, error) {
	if !json.Valid(raw) {
		return nil, nil, true, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if trimmed[0] != '{' {
		return nil, nil, false, fmt.Errorf("%w: accepted update response is JSON but not an object", resource.ErrInvalidOption)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return nil, nil, false, err
	}
	object := json.RawMessage(bytes.Clone(trimmed))
	if nested, present := fields[key]; present {
		nested = bytes.TrimSpace(nested)
		if len(nested) == 0 || nested[0] != '{' {
			return nil, nil, false, fmt.Errorf("%w: accepted update response %q member is not an object", resource.ErrInvalidOption, key)
		}
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil {
			return nil, nil, false, err
		}
		object = bytes.Clone(nested)
	}
	delete(fields, "self")
	return maps.Clone(fields), object, false, nil
}

// decodeUpdateMerged projects the merged Python state into a native typed
// model. Known-field type mismatches are recorded as ProjectionError without
// failing the accepted update.
func decodeUpdateMerged[T any](receipt *UpdateReceipt) (*T, error) {
	if receipt == nil || receipt.Merged == nil {
		return nil, nil
	}
	data, err := json.Marshal(receipt.Merged.Body)
	if err != nil {
		return nil, err
	}
	value := new(T)
	if err := json.Unmarshal(data, value); err == nil {
		return value, nil
	}
	// Omit only the members the native schema cannot hold, one at a time,
	// so the remaining merged state is still projected.
	kept := map[string]json.RawMessage{}
	omitted := []string{}
	for _, key := range slices.Sorted(maps.Keys(receipt.Merged.Body)) {
		single, _ := json.Marshal(map[string]json.RawMessage{key: receipt.Merged.Body[key]})
		if json.Unmarshal(single, new(T)) != nil {
			omitted = append(omitted, key)
			continue
		}
		kept[key] = receipt.Merged.Body[key]
	}
	receipt.ProjectionError = fmt.Errorf("%w: merged update members %q do not match the native model", resource.ErrInvalidOption, omitted)
	data, _ = json.Marshal(kept)
	value = new(T)
	if json.Unmarshal(data, value) != nil {
		return nil, nil
	}
	return value, nil
}
