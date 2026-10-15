package secrets

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
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// UpdateRecordOpts describes an owned, newly seeded Secret commit. Attributes
// are Source Body names or wire aliases; unknown ordinary attrs are discarded.
// Headers/Microversion are explicit Go transport controls. BasePath is a literal
// path within the selected service; Python printf URI templates are not enabled.
type UpdateRecordOpts struct {
	Attributes             map[string]json.RawMessage
	Headers                map[string]string
	Microversion, BasePath *string
	order                  []string
}
type UpdateRecordOption func(*UpdateRecordOpts) error

func copyUpdateRecordOpts(c UpdateRecordOpts) UpdateRecordOpts {
	c.Attributes = maps.Clone(c.Attributes)
	for k, v := range c.Attributes {
		c.Attributes[k] = bytes.Clone(v)
	}
	c.Headers = maps.Clone(c.Headers)
	c.order = slices.Clone(c.order)
	if c.Microversion != nil {
		x := *c.Microversion
		c.Microversion = &x
	}
	if c.BasePath != nil {
		x := *c.BasePath
		c.BasePath = &x
	}
	return c
}
func WithUpdateRecordOpts(value UpdateRecordOpts) UpdateRecordOption {
	owned := copyUpdateRecordOpts(value)
	return func(c *UpdateRecordOpts) error { *c = copyUpdateRecordOpts(owned); return nil }
}
func WithUpdateRecordAttribute(key string, value any) UpdateRecordOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *UpdateRecordOpts) error {
		if err != nil {
			return err
		}
		if c.Attributes == nil {
			c.Attributes = map[string]json.RawMessage{}
		}
		if _, exists := c.Attributes[key]; !exists {
			c.order = append(c.order, key)
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithUpdateRecordHeaders(value map[string]string) UpdateRecordOption {
	owned := maps.Clone(value)
	return func(c *UpdateRecordOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithUpdateRecordMicroversion(value string) UpdateRecordOption {
	return func(c *UpdateRecordOpts) error { x := value; c.Microversion = &x; return nil }
}
func WithUpdateRecordBasePath(value string) UpdateRecordOption {
	return func(c *UpdateRecordOpts) error { x := value; c.BasePath = &x; return nil }
}

// UpdatedSecretRecord keeps request seed/body separate from an actual PUT
// observation. Committed means the accepted HTTP body/Close completed; false
// never proves that the server did not execute an attempted request.
// Wire contains only received fields; Resource overlays them onto the seed.
type UpdatedSecretRecord struct {
	RequestID      string
	Seed, Request  map[string]json.RawMessage
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Committed      bool
}

// UpdateRecord implements the string-ID Secret update lane. Unlike Update's
// raw-payload upload, this sends unwrapped JSON dirty attributes. ID itself is
// not dirty; no recognized attrs means no PUT and no payload/metadata GET.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...UpdateRecordOption) (*UpdatedSecretRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateRecord", "secrets", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(updateRecordInvalid("secret API is required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "key-manager")
	if err != nil {
		return nil, wrap(err)
	}
	var changed error
	guard := func(c context.Context) error {
		if changed == nil && a.client != selected {
			changed = updateRecordInvalid("selected secret API changed")
		}
		return errors.Join(changed, source.Guard(c), rest.CheckOperationGuard(c))
	}
	config := UpdateRecordOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(updateRecordInvalid("nil update option"))
		}
		next := copyUpdateRecordOpts(config)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		config = copyUpdateRecordOpts(next)
	}
	seedID, _ := json.Marshal(id)
	seed := map[string]json.RawMessage{"id": seedID}
	result := &UpdatedSecretRecord{RequestID: id, Seed: maps.Clone(seed), Request: map[string]json.RawMessage{}}
	ordered := slices.Clone(config.order)
	seen := map[string]bool{}
	for _, k := range ordered {
		seen[k] = true
	}
	var remaining []string
	for key := range config.Attributes {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)
	for _, key := range ordered {
		raw, present := config.Attributes[key]
		if !present {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return result, wrap(updateRecordInvalid("attribute %q requires UTF-8 JSON", key))
		}
		switch key {
		case "id", "connection", "_synchronized", "cls":
			return result, wrap(updateRecordInvalid("attribute %q collides with the Source scalar-ID constructor", key))
		case "base_path":
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				config.BasePath = nil
			} else {
				var value string
				if json.Unmarshal(raw, &value) != nil {
					return result, wrap(updateRecordInvalid("base_path requires a string or null"))
				}
				config.BasePath = &value
			}
			continue
		case "microversion", "location":
			continue
		}
		canonical, known := findRecordDescriptors[key]
		if key == "secret_id" {
			canonical, known = "secret_ref", true
		}
		if !known {
			for _, value := range findRecordDescriptors {
				if key == value {
					canonical, known = value, true
					break
				}
			}
		}
		if !known {
			continue
		}
		seed[canonical] = bytes.Clone(raw)
		result.Request[canonical] = bytes.Clone(raw)
	}
	result.Seed = updateRecordFieldsCopy(seed)
	// Reuse Source descriptors before side effects, including href formatting.
	prototype := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}}}
	view, err := projectFindRecord(prototype, seed, &id)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	result.Resource = view
	if len(result.Request) == 0 {
		return result, nil
	}
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return result, wrap(updateRecordInvalid("commit requires a nonblank literal ID without controls or dot segments"))
	}
	if err := source.WithPolicy(ctx, config.Microversion, config.Headers); err != nil {
		return result, wrap(err)
	}
	path := "secrets"
	if config.BasePath != nil {
		path = strings.Trim(*config.BasePath, "/")
	}
	for _, part := range strings.Split(path, "/") {
		if part == "." || part == ".." || strings.ContainsAny(part, "%?#") || strings.IndexFunc(part, unicode.IsControl) >= 0 || !utf8.ValidString(part) {
			return result, wrap(fmt.Errorf("%w: base_path must be a literal service-relative path without templates", resource.ErrUnsupported))
		}
	}
	target := source.Client.ServiceURL(url.PathEscape(id))
	if path != "" {
		target = source.Client.ServiceURL(path, url.PathEscape(id))
	}
	headers := maps.Clone(source.Client.MoreHeaders)
	if headers == nil {
		headers = map[string]string{}
	}
	absent := []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version"}
	if source.Client.Microversion == "" {
		absent = append(absent, "OpenStack-API-Version")
	} else {
		headers["OpenStack-API-Version"] = "key-manager " + source.Client.Microversion
	}
	rejected := make([]int, 600)
	for i := range rejected {
		rejected[i] = i + 400
	}
	response, prior := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodPut, target, result.Request, rest.RequestHeaderPolicy{Values: headers, Absent: absent}, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, findRecordAcceptedCodes()...)
	if response != nil {
		result.Envelope = bytes.Clone(response.Body)
		result.Header = response.Header.Clone()
		result.StatusCode = response.StatusCode
	} else if native, ok := updateRecordNative(prior); ok {
		result.Envelope = bytes.Clone(native.Body)
		result.Header = native.ResponseHeader.Clone()
		result.StatusCode = native.Actual
	}
	if err := errors.Join(prior, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	result.Committed = true
	// Resource._translate_response tolerates ValueError from response.json(),
	// keeping the request seed; valid nonobject JSON still raises a shape error.
	if !utf8.Valid(response.Body) {
		return result, wrap(response.Fail(fmt.Errorf("%w: Requests response JSON charset decoding remains unsupported", resource.ErrUnsupported)))
	}
	if len(bytes.TrimSpace(response.Body)) > 0 && json.Valid(response.Body) {
		fields := map[string]json.RawMessage{}
		if raw := bytes.TrimSpace(response.Body); raw[0] != '{' {
			return result, wrap(response.Fail(updateRecordInvalid("commit response must support object pop/attribute merge")))
		}
		if err := json.Unmarshal(response.Body, &fields); err != nil {
			return result, wrap(response.Fail(err))
		}
		result.Wire = &resource.RawResource{Metadata: resource.Metadata{Body: updateRecordFieldsCopy(fields), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		members, parseErr := cloudfilter.ObjectMembers(response.Body)
		if parseErr != nil {
			return result, wrap(response.Fail(parseErr))
		}
		for _, member := range members {
			key, value := member.Key, member.Value
			if key == "self" {
				continue
			}
			canonical, known := findRecordDescriptors[key]
			if key == "secret_id" {
				canonical, known = "secret_ref", true
			}
			if key == "id" {
				canonical, known = "id", true
			}
			if !known {
				for _, wireKey := range findRecordDescriptors {
					if key == wireKey {
						canonical, known = wireKey, true
						break
					}
				}
			}
			if known {
				seed[canonical] = bytes.Clone(value)
			}
		}
	}
	physical := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	projected, err := projectFindRecord(physical, seed, &id)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(response.Fail(err))
	}
	result.Resource = projected
	return result, nil
}
func updateRecordFieldsCopy(fields map[string]json.RawMessage) map[string]json.RawMessage {
	out := maps.Clone(fields)
	for key, value := range out {
		out[key] = bytes.Clone(value)
	}
	return out
}
func updateRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func updateRecordNative(err error) (gophercloud.ErrUnexpectedResponseCode, bool) {
	var unable *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &unable) {
		err = unable.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	ok := errors.As(err, &native)
	return native, ok
}
