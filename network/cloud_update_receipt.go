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
// Merged is the resulting Python resource state: the lookup wire row,
// overlaid with every effective update attribute (the same values Python
// assigns to the found Resource before commit), overlaid with Body members.
// Its Header and StatusCode are those of the PUT response.
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
func cloudUpdatePut(ctx context.Context, client *gophercloud.ServiceClient, guard func(context.Context) error, endpoint, key string, lookup, effective, sent map[string]json.RawMessage) (*UpdateReceipt, error) {
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
	body, ignored, cause := cloudUpdateBody(response.Body, key)
	if cause != nil {
		return receipt, response.Fail(cause)
	}
	receipt.Body, receipt.BodyIgnored = body, ignored
	merged := make(map[string]json.RawMessage, len(lookup)+len(effective)+len(body))
	for _, layer := range []map[string]json.RawMessage{lookup, effective, body} {
		for name, raw := range layer {
			merged[name] = bytes.Clone(raw)
		}
	}
	receipt.Merged = &resource.RawResource{Metadata: resource.Metadata{Body: merged, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	return receipt, nil
}

func cloudUpdateBody(raw []byte, key string) (map[string]json.RawMessage, bool, error) {
	if !json.Valid(raw) {
		return nil, true, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if trimmed[0] != '{' {
		return nil, false, fmt.Errorf("%w: accepted update response is JSON but not an object", resource.ErrInvalidOption)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return nil, false, err
	}
	if nested, present := fields[key]; present {
		nested = bytes.TrimSpace(nested)
		if len(nested) == 0 || nested[0] != '{' {
			return nil, false, fmt.Errorf("%w: accepted update response %q member is not an object", resource.ErrInvalidOption, key)
		}
		fields = nil
		if err := json.Unmarshal(nested, &fields); err != nil {
			return nil, false, err
		}
	}
	delete(fields, "self")
	return maps.Clone(fields), false, nil
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
