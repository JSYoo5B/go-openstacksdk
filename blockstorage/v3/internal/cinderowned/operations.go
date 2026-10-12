package cinderowned

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Begin captures the client, applies options and headers, then selects the
// microversion. It is the common preamble of every owned call.
func Begin[T any, O ~func(*T) error](ctx context.Context, t Target, zero T, copyValue func(T) T, headers func(T) map[string]string, version Version, options []O) (*Prepared, T, error) {
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, zero, err
	}
	value, err := Apply(ctx, p.guard, zero, copyValue, slices.Clone(options))
	if err == nil {
		err = p.Headers(headers(value))
	}
	if err == nil {
		err = p.Select(version)
	}
	return p, value, err
}

// Seed returns owned fields with the Python id (IDKey) set when id is given.
func (p *Prepared) Seed(fields map[string]json.RawMessage, id string) map[string]json.RawMessage {
	seed := CloneFields(fields)
	if seed == nil {
		seed = make(map[string]json.RawMessage)
	}
	if id != "" {
		raw, _ := json.Marshal(id)
		seed["id"] = raw
	}
	return seed
}

// Get sends Resource.fetch: GET {collection}/{id} with the optional query.
func (p *Prepared) Get(id string, query url.Values) (*Record, error) {
	if err := ValidateID("ID", id); err != nil {
		return nil, err
	}
	target := p.URL(id)
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	response, err := p.Do(http.MethodGet, target, nil)
	if err != nil {
		return p.Evidence(response, err), err
	}
	return p.Record(response, p.Seed(nil, id))
}

// Create sends Resource.create with POST {collection} and the resource_key
// envelope of the given declared attributes.
func (p *Prepared) Create(values map[string]json.RawMessage) (*Record, error) {
	fields, err := p.target.Spec.BodyFields(values, false)
	if err != nil {
		return nil, err
	}
	body := map[string]any{p.target.Spec.ResourceKey: CloneFields(fields)}
	response, err := p.Do(http.MethodPost, p.URL(), body)
	if err != nil {
		return p.Evidence(response, err), err
	}
	return p.Record(response, fields)
}

// Update sends Resource.commit with PUT and only the dirty declared
// attributes. The ID is never dirty; nothing to commit means no HTTP.
func (p *Prepared) Update(id string, values map[string]json.RawMessage) (*Record, error) {
	if err := ValidateID("ID", id); err != nil {
		return nil, err
	}
	fields, err := p.target.Spec.BodyFields(values, true)
	if err != nil {
		return nil, err
	}
	seed := p.Seed(fields, id)
	if len(fields) == 0 {
		value := &Record{Discovery: p.Discovery()}
		p.target.Spec.Project(value, seed)
		return value, p.guard(p.ctx)
	}
	body := map[string]any{p.target.Spec.ResourceKey: CloneFields(fields)}
	response, err := p.Do(http.MethodPut, p.URL(id), body)
	if err != nil {
		return p.Evidence(response, err), err
	}
	return p.Record(response, seed)
}

// Delete sends DELETE target. A clean 404 is nil when ignoreMissing is true.
func (p *Prepared) Delete(target string, ignoreMissing bool) error {
	_, err := p.Do(http.MethodDelete, target, nil)
	if ignoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return err
}

// List sends Resource.list on path: server query, Accept: application/json
// unless the caller set one, local Body filters after the raw row cap and
// Python pagination continuation rules.
func (p *Prepared) List(path string, query url.Values, filters map[string]json.RawMessage, control rest.ListControl) iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		var origin *rest.Response
		spec := p.target.Spec
		parsed, _ := url.Parse(p.source.Client.ServiceURL(path))
		collection := rest.CollectionSpec[row]{Client: &p.source.Client, Path: path, Kind: spec.Kind, PluralKey: spec.ResourcesKey, Metadata: func(r *row) *resource.Metadata { return &r.meta }, Validate: p.guard, SourceGuard: p.guard, ListCodes: Codes(),
			ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
				return p.Do(http.MethodGet, target, nil)
			},
			ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) },
			Paging:           rest.PagePolicy[row]{VersionedPath: parsed.EscapedPath(), HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerOrOffsetPagination: true, MarkerFallback: true, Marker: spec.marker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for item, err := range rest.ListWithControl(p.ctx, collection, query, control) {
			if err != nil {
				yield(nil, err)
				return
			}
			value, err := p.row(item, origin)
			if err == nil {
				var matched bool
				matched, err = jsonfilter.MatchFilters(value.View, filters)
				if err = errors.Join(err, p.guard(p.ctx)); err == nil && !matched {
					continue
				}
			}
			if err != nil {
				yield(nil, origin.Fail(err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

// ListOptions prepares Python **query and runs List on path.
func (p *Prepared) ListOptions(path string, options ListOpts) iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		query, filters, control, err := p.target.Spec.prepareList(options)
		if err != nil {
			yield(nil, err)
			return
		}
		p.Accept("application/json", false)
		for value, err := range p.List(path, query, filters, control) {
			if !yield(value, err) || err != nil {
				return
			}
		}
	}
}

func (p *Prepared) row(item *row, page *rest.Response) (*Record, error) {
	if item == nil || item.meta.Body == nil || page == nil {
		return nil, Invalid("%s list item is missing", p.target.Spec.Kind)
	}
	value := &Record{Wire: CloneFields(item.meta.Body), Envelope: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode, Discovery: p.Discovery()}
	p.target.Spec.Project(value, nil)
	return value, p.guard(p.ctx)
}

// row is one raw list item. DecodeObject keeps every original field in
// meta.Body; nothing is decoded into typed fields.
type row struct{ meta resource.Metadata }

func (*row) UnmarshalJSON([]byte) error { return nil }

func (s *Spec) marker(r *row) (string, error) {
	if r == nil || r.meta.Body == nil {
		return "", Invalid("record marker missing")
	}
	var id string
	if err := json.Unmarshal(s.PythonID(r.meta.Body), &id); err != nil || id == "" {
		return "", Invalid("record marker must be a nonempty string")
	}
	return id, nil
}

// Find follows Resource.find: GET {collection}/{name_or_id} with getQuery,
// then on a clean 400, 403 or 404 the complete list on listPath with
// listQuery, matched by exact ID (IDKey) or name. Duplicates fail.
func (p *Prepared) Find(ctx context.Context, nameOrID string, getQuery url.Values, listPath string, listQuery url.Values, ignoreMissing bool) (*Record, error) {
	frozenGet := cloneValues(getQuery)
	frozenList := cloneValues(listQuery)
	p.Accept("application/json", false)
	collection := resource.NewCollection(resource.Adapter[Record]{Kind: p.target.Spec.Kind, IdentityFind: true,
		ValidateID:        func(id string) error { return ValidateID("ID", id) },
		IdentityDirectGet: func(id string) error { return ValidateID("ID", id) },
		IdentityResponseID: func(v *Record) (string, error) {
			if v == nil {
				return "", Invalid("find response missing")
			}
			return StringValue(p.target.Spec.PythonID(v.View)), nil
		},
		Name: func(v *Record) string { return v.Text("name") },
		Get:  func(_ context.Context, id string) (*Record, error) { return p.Get(id, frozenGet) },
		Iterate: func(context.Context, url.Values) iter.Seq2[*Record, error] {
			return p.List(listPath, cloneValues(frozenList), nil, rest.ListControl{})
		}})
	value, err := collection.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(ignoreMissing))
	return value, errors.Join(err, p.guard(ctx))
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	result := make(url.Values, len(values))
	for key, list := range values {
		result[key] = slices.Clone(list)
	}
	return result
}
