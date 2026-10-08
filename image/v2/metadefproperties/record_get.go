package metadefproperties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetRecord fetches a declared property view while preserving the actual Wire
// and HTTP receipt separately. Recognized seed attributes survive omissions;
// empty or invalid JSON after actual200..399 leaves those seeded defaults.
// The input Resource is copied and no cached object is reused or mutated.
func (s *NamespaceScope) GetRecord(ctx context.Context, input RecordRequest, options ...RecordGetOption) (*Record, error) {
	fail := func(err error) (*Record, error) { return nil, wrap(ctx, "GetRecord", err) }
	p, err := s.capture(ctx, nil)
	if err != nil {
		return fail(err)
	}
	namespace := s.namespace
	outer := rest.OperationGuard(ctx)
	var observed error
	check := func(checkCtx context.Context) error {
		if observed != nil {
			return observed
		}
		var parent, binding error
		if outer != nil {
			parent = outer(checkCtx)
		}
		if s.namespace != namespace {
			binding = invalid("property namespace changed")
		}
		observed = errors.Join(p.check(checkCtx), parent, binding)
		return observed
	}
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return fail(err)
	}
	if input.ID != "" && input.Resource != nil {
		return fail(invalid("select ID or Resource, not both"))
	}
	if input.ID == "" && input.Resource == nil {
		return fail(invalid("property input is required"))
	}
	seed := make(map[string]json.RawMessage)
	if input.Resource != nil {
		seed = input.Resource.Clone().Body
	}
	if input.ID != "" {
		if err := literal(input.ID); err != nil {
			return fail(err)
		}
		seed["id"], _ = json.Marshal(input.ID)
	}
	seed, err = normalizePropertyRecord(seed, nil)
	if err != nil {
		return fail(err)
	}
	location := json.RawMessage("null")
	if s.api.dependencies.CloudLocation != nil {
		facts, readErr := s.api.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	attrs, headers, err := prepareRecordGet(opctx, check, slices.Clone(options))
	if err != nil {
		return fail(err)
	}
	if _, present := attrs["id"]; present && input.ID != "" {
		return fail(invalid("string input already binds id"))
	}
	attrs, err = normalizePropertyRecord(attrs, nil)
	if err != nil {
		return fail(err)
	}
	for key, raw := range attrs {
		seed[key] = bytes.Clone(raw)
	}
	raw, hasID := seed["id"]
	if !hasID {
		raw = seed["name"]
	}
	var identity string
	if err := json.Unmarshal(raw, &identity); err != nil {
		return fail(errors.Join(invalid("property identity must be a string"), err))
	}
	if err := literal(identity); err != nil {
		return fail(err)
	}
	// Constructor/update descriptor failures are preflight failures too. Do not
	// hide invalid seed data merely because a server could replace it later.
	if _, err := projectPropertyRecord(seed, namespace, location, resource.Metadata{}); err != nil {
		return fail(err)
	}
	if err = errors.Join(p.finish(opctx, headers, nil), check(opctx)); err != nil {
		return fail(err)
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodGet, p.url(&identity), nil, nil, codes...)
	if err != nil {
		return fail(err)
	}
	if !utf8.Valid(response.Body) {
		return fail(response.Fail(invalid("property response must be UTF-8")))
	}
	record := &Record{Namespace: namespace, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if err := json.Unmarshal(response.Body, wire); err != nil {
			return fail(response.Fail(err))
		}
		fields, err := normalizePropertyRecord(wire.Body, response.Body)
		if err != nil {
			return fail(response.Fail(err))
		}
		for key, raw := range fields {
			seed[key] = bytes.Clone(raw)
		}
		record.Wire = wire
	}
	record.Resource, err = projectPropertyRecord(seed, namespace, location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(response.Fail(err))
	}
	return record, nil
}
