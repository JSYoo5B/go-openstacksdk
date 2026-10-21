// Package manilaown holds the shared transport helpers of SDK-owned Manila
// bindings that Gophercloud v2.15.0 does not provide. Every call is exactly one
// HTTP request: nothing is retried, replayed or upgraded here, and the selected
// microversion is only compared with a documented floor, never changed.
package manilaown

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Evidence keeps the actual wire facts next to a typed view: the decoded JSON
// object of the resource (null and unknown members included), the response
// headers and the HTTP status. Rows of a list share the list response header.
type Evidence struct {
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// Response is the retained result of one request.
type Response struct {
	StatusCode int
	Header     http.Header
	Raw        []byte
}

// Call describes one request. Floor is the first Manila minor version (2.N)
// that has the route; zero disables the local check.
type Call struct {
	Client *gophercloud.ServiceClient
	Floor  int
	Method string
	URL    string
	Body   any
	OK     []int
}

// CheckFloor rejects a numeric selected version below 2.floor and an empty
// selection (Manila would then serve 2.0). The symbolic "latest" passes
// because the service resolves it to its newest version.
func CheckFloor(client *gophercloud.ServiceClient, floor int) error {
	if client == nil {
		return fmt.Errorf("%w: Manila service client is required", resource.ErrInvalidOption)
	}
	if floor <= 0 {
		return nil
	}
	selected := client.Microversion
	if selected == "latest" {
		return nil
	}
	if selected == "" {
		return fmt.Errorf("%w: this Manila route needs microversion 2.%d or later; none is selected", resource.ErrUnsupported, floor)
	}
	parts := strings.Split(selected, ".")
	if len(parts) != 2 || parts[0] != "2" {
		return fmt.Errorf("%w: Manila microversion must be 2.N, got %q", resource.ErrInvalidOption, selected)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 || strconv.Itoa(minor) != parts[1] {
		return fmt.Errorf("%w: invalid Manila microversion %q", resource.ErrInvalidOption, selected)
	}
	if minor < floor {
		return fmt.Errorf("%w: this Manila route needs microversion 2.%d or later, selected %s", resource.ErrUnsupported, floor, selected)
	}
	return nil
}

// Do sends the request once and returns the retained response. A canceled
// context or a local precondition failure sends nothing.
func (c Call) Do(ctx context.Context) (*Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := CheckFloor(c.Client, c.Floor); err != nil {
		return nil, err
	}
	opts := &gophercloud.RequestOpts{OkCodes: c.OK, KeepResponseBody: true, MoreHeaders: map[string]string{"Accept": "application/json"}}
	if c.Body != nil {
		opts.JSONBody = c.Body
	}
	resp, err := c.Client.Request(ctx, c.Method, c.URL, opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Raw: raw}, nil
}

func object(raw []byte, key string) (map[string]json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	data, ok := envelope[key]
	if !ok {
		return nil, fmt.Errorf("response has no %q member", key)
	}
	return Object(data, key)
}

// Object decodes a JSON object member, rejecting null and non-objects.
func Object(data json.RawMessage, what string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("%s must be a JSON object", what)
	}
	return fields, nil
}

func fill[T any](data json.RawMessage, fields map[string]json.RawMessage, resp *Response) (*T, error) {
	value := new(T)
	if err := json.Unmarshal(data, value); err != nil {
		return nil, err
	}
	rv := reflect.ValueOf(value).Elem()
	if rv.Kind() == reflect.Struct {
		if f := rv.FieldByName("Evidence"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(Evidence{}) {
			f.Set(reflect.ValueOf(Evidence{Body: fields, Header: resp.Header.Clone(), StatusCode: resp.StatusCode}))
		}
	}
	return value, nil
}

// One decodes the object under key into T and stores Evidence in its
// Evidence field when the type has one.
func One[T any](resp *Response, key string) (*T, error) {
	fields, err := object(resp.Raw, key)
	if err != nil {
		return nil, err
	}
	encoded, _ := json.Marshal(fields)
	return fill[T](encoded, fields, resp)
}

// Flat decodes a top-level object, or the object under key when present.
func Flat[T any](resp *Response, key string) (*T, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(resp.Raw, &envelope); err != nil || envelope == nil {
		return nil, fmt.Errorf("response must be a JSON object")
	}
	if inner, ok := envelope[key]; ok {
		fields, err := Object(inner, key)
		if err != nil {
			return nil, err
		}
		return fill[T](inner, fields, resp)
	}
	return fill[T](resp.Raw, envelope, resp)
}

// Rows decodes the array under key into one T per element.
func Rows[T any](resp *Response, key string) ([]*T, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(resp.Raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(envelope[key], &rows); err != nil || envelope[key] == nil {
		return nil, fmt.Errorf("response member %q must be an array", key)
	}
	out := make([]*T, 0, len(rows))
	for _, row := range rows {
		fields, err := Object(row, key+" row")
		if err != nil {
			return nil, err
		}
		value, err := fill[T](row, fields, resp)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

// Iterate performs one GET and yields its rows. Manila list routes in this SDK
// return one response; no continuation is followed or invented. MaxItems caps
// consumption locally. A failure is yielded once.
func Iterate[T any](ctx context.Context, call Call, key string, control resource.ListControl) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		if control.MaxItems < 0 {
			yield(nil, fmt.Errorf("%w: maximum items must be non-negative", resource.ErrInvalidOption))
			return
		}
		resp, err := call.Do(ctx)
		if err != nil {
			yield(nil, err)
			return
		}
		rows, err := Rows[T](resp, key)
		if err != nil {
			yield(nil, err)
			return
		}
		for i, row := range rows {
			if control.MaxItems > 0 && i >= control.MaxItems {
				return
			}
			if !yield(row, nil) {
				return
			}
		}
	}
}

// Query serializes the `q:"name"` fields of a typed options struct. Strings are
// omitted when empty; *int, *int64 and *bool preserve zero and false.
func Query(opts any) (url.Values, error) {
	values := url.Values{}
	rv := reflect.ValueOf(opts)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return values, nil
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		key := rt.Field(i).Tag.Get("q")
		if key == "" {
			continue
		}
		f := rv.Field(i)
		switch f.Kind() {
		case reflect.String:
			if f.String() != "" {
				values.Set(key, f.String())
			}
		case reflect.Pointer:
			if f.IsNil() {
				continue
			}
			switch e := f.Elem(); e.Kind() {
			case reflect.Int, reflect.Int64:
				if e.Int() < 0 {
					return nil, fmt.Errorf("%w: query %s must not be negative", resource.ErrInvalidOption, key)
				}
				values.Set(key, strconv.FormatInt(e.Int(), 10))
			case reflect.Bool:
				values.Set(key, strconv.FormatBool(e.Bool()))
			}
		}
	}
	return values, nil
}

// ReservedQueryKeys lists every `q` tag of opts, supplied or not.
func ReservedQueryKeys(opts any) map[string]bool {
	keys := map[string]bool{}
	rt := reflect.TypeOf(opts)
	for rt != nil && rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt == nil || rt.Kind() != reflect.Struct {
		return keys
	}
	for i := 0; i < rt.NumField(); i++ {
		if key := rt.Field(i).Tag.Get("q"); key != "" {
			keys[key] = true
		}
	}
	return keys
}

// MergeQuery adds extension query values and rejects any key owned by a typed
// field, whether or not that field was supplied.
func MergeQuery(typed url.Values, extra url.Values, opts any) (url.Values, error) {
	reserved := ReservedQueryKeys(opts)
	for key, values := range extra {
		if reserved[key] || typed.Has(key) {
			return nil, fmt.Errorf("%w: query extension %q is a typed list option", resource.ErrInvalidOption, key)
		}
		typed[key] = append([]string(nil), values...)
	}
	return typed, nil
}

// WithQuery appends an encoded query to a route.
func WithQuery(target string, values url.Values) string {
	if len(values) == 0 {
		return target
	}
	return target + "?" + values.Encode()
}

// Body marshals typed options to a JSON object so extension fields can be
// merged under an envelope without losing omission, null, false or zero.
func Body(opts any) (map[string]any, error) {
	encoded, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("%w: request options: %v", resource.ErrInvalidOption, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out, nil
}

// MetadataBody reports whether data is an object of strings for the metadata
// envelope; values stay strings exactly as supplied.
func MetadataMap(resp *Response, key string) (map[string]string, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(resp.Raw), &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	out := map[string]string{}
	data, ok := envelope[key]
	if !ok || string(bytes.TrimSpace(data)) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("response member %q must be an object of strings: %w", key, err)
	}
	if out == nil {
		out = map[string]string{}
	}
	return out, nil
}

// Envelope builds {key: fields} from typed options and merges extension
// fields into it. Extension keys that collide with a typed option, or with any
// key of protected (a struct embedding the options plus server-owned names),
// are rejected before HTTP.
func Envelope(key string, opts any, fields map[string]json.RawMessage, protected any) (map[string]any, error) {
	inner, err := Body(opts)
	if err != nil {
		return nil, err
	}
	return request.MergeFieldsFor(map[string]any{key: inner}, fields, protected)
}

// NotFound converts a 404 into resource.ErrNotFound while keeping the native
// error in the chain; every other error is returned unchanged.
func NotFound(kind, reference string, ctx context.Context, err error) error {
	if err != nil && ctx.Err() == nil && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return &resource.NotFoundError{Resource: kind, Reference: reference, Cause: err}
	}
	return err
}

// ControlArgument carries WithListMaxItems/WithListPaginated through request
// options. Manila list routes return one response, so SinglePage changes nothing.
const ControlArgument = "manilaown.list_control"

// WithListControl edits the local consumption control of a typed list.
func WithListControl[O any](edit func(*resource.ListControl)) request.Option[O] {
	return func(cfg *request.Config[O]) error {
		control, _, err := request.Argument[resource.ListControl](*cfg, ControlArgument)
		if err != nil {
			return err
		}
		edit(&control)
		if cfg.Arguments == nil {
			cfg.Arguments = make(map[string]any)
		}
		cfg.Arguments[ControlArgument] = control
		return nil
	}
}

// ListFor builds a lazy typed list. route receives the applied options and
// returns the path to GET (without query); extra names additional library
// arguments the operation understands.
func ListFor[T any, O any](ctx context.Context, client *gophercloud.ServiceClient, floor int, key string, route func(request.Config[O]) string, control resource.ListControl, options []request.Option[O], extra ...string) iter.Seq2[*T, error] {
	options = append([]request.Option[O](nil), options...)
	return func(yield func(*T, error) bool) {
		var zero O
		cfg, err := request.Apply(zero, options...)
		if err == nil {
			err = request.ValidateCapabilities(cfg, false, true, false, append([]string{ControlArgument}, extra...)...)
		}
		if err != nil {
			yield(nil, err)
			return
		}
		if override, ok, err := request.Argument[resource.ListControl](cfg, ControlArgument); err != nil {
			yield(nil, err)
			return
		} else if ok {
			control = override
		}
		typed, err := Query(cfg.Options)
		if err == nil {
			typed, err = MergeQuery(typed, cfg.Query, cfg.Options)
		}
		if err != nil {
			yield(nil, err)
			return
		}
		call := Call{Client: client, Floor: floor, Method: http.MethodGet, URL: WithQuery(route(cfg), typed), OK: []int{http.StatusOK}}
		for value, err := range Iterate[T](ctx, call, key, control) {
			if !yield(value, err) || err != nil {
				return
			}
		}
	}
}

// Whole decodes the entire response object into T and stores Evidence.
func Whole[T any](resp *Response) (*T, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(resp.Raw, &fields); err != nil || fields == nil {
		return nil, fmt.Errorf("response must be a JSON object")
	}
	return fill[T](resp.Raw, fields, resp)
}
