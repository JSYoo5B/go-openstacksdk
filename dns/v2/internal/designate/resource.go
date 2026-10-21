package designate

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Ack is the retained evidence of a bodyless acknowledgement. A 202 or 204 is
// not proof that asynchronous remote work has completed.
type Ack struct {
	Header     http.Header
	StatusCode int
}

// Route is one flat-body Designate collection, such as /blacklists.
type Route[T any, P Model[T]] struct {
	Client    *gophercloud.ServiceClient
	Segments  []string // collection path segments below the version root
	PluralKey string
}

func (r Route[T, P]) member(id string) ([]string, error) {
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	return append(append([]string(nil), r.Segments...), id), nil
}

// Get performs GET <collection>/<id> accepting only 200.
func (r Route[T, P]) Get(ctx context.Context, id string, headers map[string]string) (P, error) {
	if err := Client(ctx, r.Client); err != nil {
		return nil, err
	}
	path, err := r.member(id)
	if err != nil {
		return nil, err
	}
	response, err := Do(ctx, r.Client, http.MethodGet, r.Client.ServiceURL(escape(path)...), nil, headers, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return Decode[T, P](response)
}

// Send performs a body-carrying request on the collection or one member.
func (r Route[T, P]) Send(ctx context.Context, method string, id string, body map[string]any, headers map[string]string, codes ...int) (P, error) {
	if err := Client(ctx, r.Client); err != nil {
		return nil, err
	}
	path := append([]string(nil), r.Segments...)
	if id != "" {
		var err error
		if path, err = r.member(id); err != nil {
			return nil, err
		}
	}
	var payload any
	if body != nil {
		payload = body
	}
	response, err := Do(ctx, r.Client, method, r.Client.ServiceURL(escape(path)...), payload, headers, codes...)
	if err != nil {
		return nil, err
	}
	return Decode[T, P](response)
}

// Delete performs DELETE <collection>/<id> accepting only the supplied codes.
func (r Route[T, P]) Delete(ctx context.Context, id string, headers map[string]string, codes ...int) (*Ack, error) {
	if err := Client(ctx, r.Client); err != nil {
		return nil, err
	}
	path, err := r.member(id)
	if err != nil {
		return nil, err
	}
	response, err := Do(ctx, r.Client, http.MethodDelete, r.Client.ServiceURL(escape(path)...), nil, headers, codes...)
	if err != nil {
		return nil, err
	}
	return &Ack{Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// List streams the collection with List's paging rules.
func (r Route[T, P]) List(ctx context.Context, query url.Values, headers map[string]string, control resource.ListControl) iter.Seq2[P, error] {
	return List[T, P](ctx, r.Client, ListSpec{Path: r.Segments, PluralKey: r.PluralKey}, query, headers, control)
}

func escape(segments []string) []string {
	out := make([]string, len(segments))
	for i, segment := range segments {
		out[i] = url.PathEscape(segment)
	}
	return out
}

// Body marshals a concrete request value to a flat JSON object and merges
// extension fields. Core field names cannot be overwritten by extensions.
func Body[T any](options T, extensions map[string]json.RawMessage) (map[string]any, error) {
	encoded, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("%w: request input: %v", resource.ErrInvalidOption, err)
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, fmt.Errorf("%w: request input: %v", resource.ErrInvalidOption, err)
	}
	fields := make(map[string]any, len(core))
	for key, value := range core {
		fields[key] = value
	}
	return request.MergeFieldsFor(fields, extensions, options)
}

// Query copies filters and applies extension query values over typed ones.
func Query(typed url.Values, extra url.Values) url.Values {
	out := url.Values{}
	for key, values := range typed {
		out[key] = append([]string(nil), values...)
	}
	for key, values := range extra {
		out[key] = append([]string(nil), values...)
	}
	return out
}

// NotFound adds resource.ErrNotFound to a 404 while keeping the native error.
func NotFound(kind, id string, err error) error {
	if err != nil && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return &resource.NotFoundError{Resource: kind, Reference: id, Cause: err}
	}
	return err
}
