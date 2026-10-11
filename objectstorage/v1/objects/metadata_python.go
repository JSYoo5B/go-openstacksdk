package objects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// FreshMetadataChangeResult keeps each actual HTTP phase separate. NoOp means
// no POST was needed, and may accompany a fresh Before observation.
type FreshMetadataChangeResult struct {
	Before          *GetMetadataResult
	Acknowledgement *MetadataResponse
	Refreshed       *GetMetadataResult
	NoOp            bool
}

// MetadataKeyError preserves the pinned Python delete_after branch's missing
// delete_at error. It does not claim that any backend metadata was removed.
type MetadataKeyError struct{ Key string }

func (err *MetadataKeyError) Error() string {
	return fmt.Sprintf("object metadata key %q is absent", err.Key)
}

// HeadMetadata implements a query-free Python object metadata HEAD accepting
// 200..399. The existing GetMetadata retains its original native status policy.
func (a *API) HeadMetadata(ctx context.Context, container, object string, options ...GetMetadataOption) (*GetMetadataResult, error) {
	p, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("HeadMetadata", "objects", err)
	}
	cfg, err := applyGetMetadataOptions(append([]GetMetadataOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("HeadMetadata", "objects", err)
	}
	if cfg.Newest != nil {
		p.client.MoreHeaders["X-Newest"] = strconv.FormatBool(*cfg.Newest)
	}
	result, _, err := p.pythonReadMetadata(ctx)
	return result, request.Wrap("HeadMetadata", "objects", err)
}

// SetMetadataFromFresh implements the Python setter for an object name: fresh
// HEAD, preserve observed custom/system metadata, then POST. Falsey JSON values
// are skipped. The setter has no final HEAD and cannot provide atomic updates.
func (a *API) SetMetadataFromFresh(ctx context.Context, container, object string, values map[string]any, options ...MetadataOption) (*FreshMetadataChangeResult, error) {
	p, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("SetMetadataFromFresh", "objects", err)
	}
	incoming, conversionErr := pythonObjectMetadataValues(values, true)
	if err = joinMetadataErrors(conversionErr, p.check(ctx)); err != nil {
		return nil, request.Wrap("SetMetadataFromFresh", "objects", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataOptions(append([]MetadataOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("SetMetadataFromFresh", "objects", err)
	}
	before, response, err := p.pythonReadMetadata(ctx)
	if before == nil {
		return nil, request.Wrap("SetMetadataFromFresh", "objects", err)
	}
	result := &FreshMetadataChangeResult{Before: before}
	if err != nil {
		return result, request.Wrap("SetMetadataFromFresh", "objects", err)
	}
	systems, err := pythonObjectSystemValues(before.Header)
	if err = joinMetadataErrors(err, p.check(ctx)); err != nil {
		return result, request.Wrap("SetMetadataFromFresh", "objects", response.Fail(err))
	}
	merged, order := pythonObjectMetadataDictionary(before.Metadata.Values, incoming, systems, false)
	headers, err := pythonObjectMetadataHeaders(merged, order, before.Metadata.Values)
	if err != nil {
		return result, request.Wrap("SetMetadataFromFresh", "objects", response.Fail(joinMetadataErrors(err, p.check(ctx))))
	}
	client := *p.client
	client.MoreHeaders = cloneMetadataHeaders(p.client.MoreHeaders)
	for key, value := range headers {
		client.MoreHeaders[key] = value
	}
	ack, err := pythonObjectMetadataExchange(ctx, p, &client, "POST")
	if ack != nil {
		result.Acknowledgement = pythonObjectMetadataAcknowledgement(ack)
	}
	return result, request.Wrap("SetMetadataFromFresh", "objects", err)
}

// DeleteMetadataAndRefresh implements Python's name-based delete: nil/empty
// keys are a local no-op, absent keys require only HEAD, and actual deletion
// performs HEAD, POST, HEAD. Keys match observed lower-case custom suffixes or
// exact system aliases. No comparison-and-swap or automatic replay is added.
func (a *API) DeleteMetadataAndRefresh(ctx context.Context, container, object string, keys []string, options ...MetadataOption) (*FreshMetadataChangeResult, error) {
	p, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("DeleteMetadataAndRefresh", "objects", err)
	}
	ownedKeys := append([]string(nil), keys...)
	cfg, err := applyMetadataOptions(append([]MetadataOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("DeleteMetadataAndRefresh", "objects", err)
	}
	result := &FreshMetadataChangeResult{}
	if len(ownedKeys) == 0 {
		result.NoOp = true
		return result, nil
	}
	before, response, err := p.pythonReadMetadata(ctx)
	result.Before = before
	if err != nil {
		return result, request.Wrap("DeleteMetadataAndRefresh", "objects", err)
	}
	systems, err := pythonObjectSystemValues(before.Header)
	if err = joinMetadataErrors(err, p.check(ctx)); err != nil {
		return result, request.Wrap("DeleteMetadataAndRefresh", "objects", response.Fail(err))
	}
	merged, order := pythonObjectMetadataDictionary(before.Metadata.Values, nil, systems, true)
	deleted := false
	for _, key := range ownedKeys {
		if key == "delete_after" {
			if _, exists := merged["delete_at"]; !exists {
				return result, request.Wrap("DeleteMetadataAndRefresh", "objects", response.Fail(joinMetadataErrors(&MetadataKeyError{Key: "delete_at"}, p.check(ctx))))
			}
			delete(merged, "delete_at")
		} else if _, exists := merged[key]; exists {
			delete(merged, key)
			deleted = true
		}
	}
	if !deleted {
		result.NoOp = true
		return result, request.Wrap("DeleteMetadataAndRefresh", "objects", p.check(ctx))
	}
	headers, err := pythonObjectMetadataHeaders(merged, order, before.Metadata.Values)
	if err != nil {
		return result, request.Wrap("DeleteMetadataAndRefresh", "objects", response.Fail(joinMetadataErrors(err, p.check(ctx))))
	}
	client := *p.client
	client.MoreHeaders = cloneMetadataHeaders(p.client.MoreHeaders)
	for key, value := range headers {
		client.MoreHeaders[key] = value
	}
	ack, err := pythonObjectMetadataExchange(ctx, p, &client, "POST")
	if ack != nil {
		result.Acknowledgement = pythonObjectMetadataAcknowledgement(ack)
	}
	if err != nil {
		return result, request.Wrap("DeleteMetadataAndRefresh", "objects", err)
	}
	result.Refreshed, _, err = p.pythonReadMetadata(ctx)
	return result, request.Wrap("DeleteMetadataAndRefresh", "objects", err)
}

var pythonObjectSystemMetadata = map[string]string{
	"accept_ranges": "accept-ranges", "content_disposition": "content-disposition",
	"content_encoding": "content-encoding", "content_type": "content-type",
	"delete_after": "x-delete-after", "delete_at": "x-delete-at",
	"is_content_type_detected": "x-detect-content-type", "manifest": "x-object-manifest",
	"access_control_allow_origin": "access-control-allow-origin",
}

func pythonObjectMetadataName(key string) string {
	if header, exists := pythonObjectSystemMetadata[key]; exists {
		return http.CanonicalHeaderKey(header)
	}
	for _, header := range pythonObjectSystemMetadata {
		if key == header {
			return http.CanonicalHeaderKey(header)
		}
	}
	if !strings.HasPrefix(key, "X-Object-Meta-") {
		key = "X-Object-Meta-" + key
	}
	return http.CanonicalHeaderKey(key)
}
func pythonObjectMetadataValues(values map[string]any, skipFalsey bool) (map[string]string, error) {
	result := make(map[string]string, len(values))
	seen := make(map[string]bool, len(values))
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if skipFalsey {
			truth, err := jsonfilter.BooleanJSON(raw)
			if err != nil {
				return nil, err
			}
			if string(truth) != "true" {
				continue
			}
		}
		text, err := cloudfilter.PythonString(raw)
		if err != nil {
			return nil, err
		}
		name := pythonObjectMetadataName(key)
		if !metadataToken(name) || !metadataFieldValue(text) {
			return nil, metadataInvalid("invalid Python object metadata %q", key)
		}
		if seen[name] {
			return nil, metadataInvalid("ambiguous Python object metadata aliases %q", key)
		}
		seen[name] = true
		result[key] = text
	}
	return result, nil
}

// Source appends missing observed descriptors after incoming values. Ambiguous
// observed custom suffixes are rejected before POST because http.Header cannot
// preserve the original response order needed to choose their last wire alias.
var pythonObjectSystemMetadataOrder = []string{"accept_ranges", "content_disposition", "content_encoding", "content_type", "delete_after", "delete_at", "is_content_type_detected", "manifest", "access_control_allow_origin"}

func pythonObjectMetadataDictionary(original, incoming, systems map[string]string, overwriteSystems bool) (map[string]string, []string) {
	values := cloneMetadataHeaders(original)
	order := make([]string, 0, len(original)+len(incoming)+len(systems))
	for key := range original {
		order = append(order, key)
	}
	sort.Strings(order)
	incomingKeys := make([]string, 0, len(incoming))
	for key := range incoming {
		incomingKeys = append(incomingKeys, key)
	}
	sort.Strings(incomingKeys)
	for _, key := range incomingKeys {
		if _, present := values[key]; !present {
			order = append(order, key)
		}
		values[key] = incoming[key]
	}
	for _, key := range pythonObjectSystemMetadataOrder {
		if value, observed := systems[key]; observed {
			_, present := values[key]
			if !present {
				order = append(order, key)
			}
			if overwriteSystems || !present {
				values[key] = value
			}
		}
	}
	return values, order
}
func pythonObjectMetadataHeaders(values map[string]string, order []string, observed map[string]string) (map[string]string, error) {
	// Only surviving observed custom suffixes need this boundary. Ordered
	// incoming/system alias precedence is known, and deletion may remove one
	// of the observed aliases before this actual POST is prepared.
	seen := make(map[string]string, len(observed))
	for _, key := range order {
		if _, present := observed[key]; !present {
			continue
		}
		if _, retained := values[key]; !retained {
			continue
		}
		name := pythonObjectMetadataName(key)
		if previous, exists := seen[name]; exists && previous != key {
			return nil, metadataInvalid("ambiguous observed Python object metadata aliases %q and %q for %q", previous, key, name)
		}
		seen[name] = key
	}
	result := make(map[string]string, len(values))
	for _, key := range order {
		value, present := values[key]
		if !present {
			continue
		}
		name := pythonObjectMetadataName(key)
		if !metadataToken(name) || !metadataFieldValue(value) {
			return nil, metadataInvalid("invalid Python object metadata %q", key)
		}
		result[name] = value
	}
	return result, nil
}
func pythonObjectSystemValues(headers http.Header) (map[string]string, error) {
	result := make(map[string]string)
	seen := make(map[string]bool)
	for key, values := range headers {
		name := strings.ToLower(key)
		alias := ""
		for candidate, wire := range pythonObjectSystemMetadata {
			if name == wire {
				alias = candidate
				break
			}
		}
		if alias == "" {
			continue
		}
		if !metadataToken(key) || seen[name] || len(values) != 1 || !metadataFieldValue(values[0]) {
			return nil, fmt.Errorf("object system header %q must have one valid value", key)
		}
		seen[name] = true
		value := values[0]
		if alias == "delete_after" {
			raw, _ := json.Marshal(value)
			integer, err := jsonfilter.DescriptorIntegerJSON(raw)
			if err != nil {
				return nil, err
			}
			truth, err := jsonfilter.BooleanJSON(integer)
			if err != nil {
				return nil, err
			}
			if string(truth) != "true" {
				continue
			}
			value, err = cloudfilter.PythonString(integer)
			if err != nil {
				return nil, err
			}
		} else if alias == "is_content_type_detected" {
			if value == "" {
				continue
			}
			value = "True"
		} else if value == "" {
			continue
		}
		result[alias] = value
	}
	return result, nil
}
func pythonObjectMetadataAcknowledgement(response *rest.Response) *MetadataResponse {
	return &MetadataResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func pythonObjectMetadataExchange(ctx context.Context, p *preparedMetadata, client *gophercloud.ServiceClient, method string) (*rest.Response, error) {
	codes := make([]int, 200)
	rejected := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
		rejected[i] = 400 + i
	}
	response, err := rest.DoJSONGuardedRejections(ctx, client, p.check, method, p.target, nil, nil, rest.RejectionPolicy{Codes: rejected}, codes...)
	return response, p.observe(ctx, response, err)
}
func (p *preparedMetadata) pythonReadMetadata(ctx context.Context) (*GetMetadataResult, *rest.Response, error) {
	response, err := pythonObjectMetadataExchange(ctx, p, p.client, "HEAD")
	if response == nil {
		return nil, nil, err
	}
	result := &GetMetadataResult{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err == nil {
		result.Metadata, err = projectMetadata(response.Header)
		if err == nil {
			_, err = pythonObjectSystemValues(response.Header)
		}
		if err = joinMetadataErrors(err, p.check(ctx)); err != nil {
			result.Metadata = nil
			err = response.Fail(err)
		}
	}
	return result, response, err
}
