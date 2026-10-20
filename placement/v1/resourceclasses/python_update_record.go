package resourceclasses

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateRecordOpts carries update_resource_class(resource_class, **attrs).
// An omitted Name sends {}, request.Present sends {"name": new} and
// request.Null sends {"name": null}.
type UpdateRecordOpts struct {
	Name request.Optional[string] `json:"name,omitzero"`
}
type UpdateRecordOption = request.Option[UpdateRecordOpts]

func WithUpdateRecordOptions(value UpdateRecordOpts) UpdateRecordOption {
	return request.WithOptions(value)
}
func WithUpdateRecordName(value string) UpdateRecordOption {
	return func(c *request.Config[UpdateRecordOpts]) error { c.Options.Name = request.Present(value); return nil }
}
func WithUpdateRecordHeader(key, value string) UpdateRecordOption {
	return request.WithHeader[UpdateRecordOpts](key, value)
}

// ResourceClassRecord is the committed Python ResourceClass view.
// ResourceClass is the native typed decode and Fields keeps the response
// object; both stay nil for an empty or non-JSON body.
type ResourceClassRecord struct {
	ResourceClass *ResourceClass
	Fields        map[string]json.RawMessage
	Header        http.Header
	StatusCode    int
}

// updateRecordMicroversion is ResourceClass._max_microversion. From 1.7 the
// PUT becomes a bodyless ensure-exists call, so the rename always uses 1.2.
const updateRecordMicroversion = "1.2"

// UpdateRecord implements update_resource_class: Resource.commit sends PUT
// resource_classes/{name} with the dirty attributes at microversion 1.2, even
// when the client selects another version, and accepts every status below
// 400.
func (a *API) UpdateRecord(ctx context.Context, name string, options ...UpdateRecordOption) (*ResourceClassRecord, error) {
	const op = "UpdateRecord"
	fail := func(value *ResourceClassRecord, err error) (*ResourceClassRecord, error) {
		return value, request.Wrap(op, "resourceclasses", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: resource class API is required", resource.ErrInvalidOption))
	}
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || !utf8.ValidString(name) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fail(nil, fmt.Errorf("%w: resource class name must be nonblank text", resource.ErrInvalidOption))
	}
	source, err := cloudread.Capture(ctx, a.client, "placement")
	if err != nil {
		return fail(nil, err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, UpdateRecordOpts{}, options, cloudread.OwnReadConfig[UpdateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		// The client's own version header belongs to its selected version and
		// is replaced on this operation's copy only.
		source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
		version := updateRecordMicroversion
		err = source.WithPolicy(ctx, &version, config.Headers)
	}
	if err != nil {
		return fail(nil, err)
	}
	body := map[string]any{}
	if config.Options.Name.IsSet() {
		body["name"] = config.Options.Name
	}
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	target := source.Client.ServiceURL("resource_classes", url.PathEscape(name))
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPut, target, body, nil, codes...)
	if err != nil {
		return fail(nil, err)
	}
	value, err := pythonResourceClassRecord(response)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return fail(value, response.Fail(err))
	}
	return value, nil
}

// pythonResourceClassRecord follows _translate_response: an empty or non-JSON
// body leaves no fields, and a JSON body must be an object.
func pythonResourceClassRecord(response *rest.Response) (*ResourceClassRecord, error) {
	value := &ResourceClassRecord{Header: response.Header.Clone(), StatusCode: response.StatusCode}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return value, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil || fields == nil {
		return value, fmt.Errorf("%w: resource class response must be a JSON object", resource.ErrInvalidOption)
	}
	typed := new(ResourceClass)
	if err := json.Unmarshal(trimmed, typed); err != nil {
		return value, err
	}
	value.ResourceClass, value.Fields = typed, fields
	return value, nil
}
