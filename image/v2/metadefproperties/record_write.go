package metadefproperties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateRecord posts a fresh flat property definition. The request contains
// only supplied raw values under their wire spellings; the returned Resource
// applies descriptor conversions and defaults. Name/type/title and id are
// passive Body attributes, and an empty definition still sends a POST {}.
func (s *NamespaceScope) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*Record, error) {
	fail := func(err error) (*Record, error) { return nil, wrap(ctx, "CreateRecord", err) }
	p, err := s.capture(ctx, nil)
	if err != nil {
		return fail(err)
	}
	namespace := s.namespace
	check := p.operationGuard(ctx)
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return fail(err)
	}
	endpoint := p.url(nil)
	location, err := p.recordLocation(opctx, check)
	if err != nil {
		return fail(err)
	}
	seed, headers, err := prepareRecordCreate(opctx, check, slices.Clone(options))
	if err != nil {
		return fail(err)
	}
	// Source construction eagerly reads descriptors through to_dict without
	// replacing stored raw values. Validate that projection before sending.
	if _, err := projectPropertyRecord(seed, namespace, location, resource.Metadata{}); err != nil {
		return fail(errors.Join(err, check(opctx)))
	}
	if err = errors.Join(p.finish(opctx, headers, nil), check(opctx)); err != nil {
		return fail(err)
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodPost, endpoint, propertyRecordWriteBody(seed, false), nil, propertyRecordWriteCodes()...)
	if err != nil {
		return fail(err)
	}
	value, err := propertyRecordFromResponse(opctx, check, seed, namespace, location, response)
	return value, wrap(ctx, "CreateRecord", err)
}

// UpdateRecord forms a fresh definition from the selected identity plus
// explicit attributes. Other input Resource fields are discarded. Null and
// default-equal explicit attributes remain dirty; no attributes means a local
// Resource result with no HTTP receipt, even if extension headers are supplied.
func (s *NamespaceScope) UpdateRecord(ctx context.Context, input RecordRequest, options ...RecordUpdateOption) (*Record, error) {
	fail := func(err error) (*Record, error) { return nil, wrap(ctx, "UpdateRecord", err) }
	p, err := s.capture(ctx, nil)
	if err != nil {
		return fail(err)
	}
	namespace := s.namespace
	check := p.operationGuard(ctx)
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return fail(err)
	}
	// Public Source normalizes Resource._get_id before creating a fresh
	// resource; copying only its selected identity also isolates callbacks.
	identity, err := recordDeletionIdentity(input)
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	endpoint := p.url(&identity)
	location, err := p.recordLocation(opctx, check)
	if err != nil {
		return fail(err)
	}
	attributes, headers, err := prepareRecordUpdate(opctx, check, slices.Clone(options))
	if err != nil {
		return fail(err)
	}
	id, err := json.Marshal(identity)
	if err != nil {
		return fail(err)
	}
	seed := map[string]json.RawMessage{"id": id}
	for key, raw := range attributes {
		seed[key] = bytes.Clone(raw)
	}
	projected, err := projectPropertyRecord(seed, namespace, location, resource.Metadata{})
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	if err = errors.Join(p.finish(opctx, headers, nil), check(opctx)); err != nil {
		return fail(err)
	}
	body := propertyRecordWriteBody(attributes, true)
	if len(body) == 0 {
		return &Record{Namespace: namespace, Resource: projected}, nil
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodPut, endpoint, body, nil, propertyRecordWriteCodes()...)
	if err != nil {
		return fail(err)
	}
	value, err := propertyRecordFromResponse(opctx, check, seed, namespace, location, response)
	return value, wrap(ctx, "UpdateRecord", err)
}

// Defaults and fallback identity are output projections, never newly dirty
// input. Only a caller's supplied fields are transposed into the wire body.
func propertyRecordWriteBody(attributes map[string]json.RawMessage, update bool) map[string]json.RawMessage {
	body := make(map[string]json.RawMessage)
	for _, field := range propertyRecordFields {
		if update && field.canonical == "id" {
			continue
		}
		if raw, present := attributes[field.canonical]; present {
			body[field.wire] = bytes.Clone(raw)
		}
	}
	return body
}
func propertyRecordWriteCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = 200 + index
	}
	return codes
}
