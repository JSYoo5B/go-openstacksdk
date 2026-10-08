package metadefresourcetypes

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

// CreateRecord creates a namespace association from supplied raw Body
// attributes. No name/id is required by the SDK and an empty definition posts
// {}. Response fields remain passive and cannot change the fixed namespace.
// Resource has all eight declared Body, URI and computed view fields; Wire and
// Envelope retain the actual response independently of the seed and defaults.
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
	// Source's fresh constructor eagerly projects descriptors, while leaving
	// only explicitly supplied raw Body values dirty for the flat POST.
	if _, err := projectAssociationRecord(seed, namespace, location, resource.Metadata{}); err != nil {
		return fail(errors.Join(err, check(opctx)))
	}
	if err = errors.Join(p.finish(opctx, headers, nil), check(opctx)); err != nil {
		return fail(err)
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodPost, endpoint, seed, nil, recordMutationCodes()...)
	if err != nil {
		return fail(err)
	}
	value, err := associationRecordFromResponse(opctx, check, seed, namespace, location, response)
	return value, wrap(ctx, "CreateRecord", err)
}

// DeleteRecord removes one fixed association identity. Resource input is
// privately snapshotted before callbacks and is never mutated. The opaque
// actual response is acknowledged independently of response handling errors.
// Accepted statuses are actual200..399 and, by default, a clean physical404.
// IgnoreMissing=false preserves native rejection/retry handling for404.
func (s *NamespaceScope) DeleteRecord(ctx context.Context, input RecordRequest, options ...DeleteOption) (*Acknowledgement, error) {
	fail := func(err error) (*Acknowledgement, error) { return nil, wrap(ctx, "DeleteRecord", err) }
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
	identity, err := recordDeletionIdentity(input)
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	endpoint := p.url(&identity)
	policy, err := prepareDelete(guardRecordDeleteOptions(opctx, check, slices.Clone(options)))
	if err = errors.Join(p.finish(opctx, policy.Headers, err), check(opctx)); err != nil {
		return fail(err)
	}
	codes := recordMutationCodes()
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodDelete, endpoint, nil, nil, codes...)
	if guardErr := check(opctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	return recordDeletionAcknowledgement(namespace, identity, response), wrap(ctx, "DeleteRecord", err)
}

// A sticky guard prevents a later callback from undoing an observed source,
// namespace or outer Connection failure. Native retry/read/close callbacks
// also use this guard through the shared rest transport.
func (p *preparedSource) operationGuard(ctx context.Context) func(context.Context) error {
	outer := rest.OperationGuard(ctx)
	var namespace *string
	if p.scope != nil {
		namespace = copyPointer(&p.scope.namespace)
	}
	var observed error
	return func(checkCtx context.Context) error {
		if observed != nil {
			return observed
		}
		var parent, binding error
		if outer != nil {
			parent = outer(checkCtx)
		}
		if namespace != nil && p.scope.namespace != *namespace {
			binding = invalid("association namespace changed")
		}
		observed = errors.Join(p.check(checkCtx), parent, binding)
		return observed
	}
}

func (p *preparedSource) recordLocation(ctx context.Context, check func(context.Context) error) (json.RawMessage, error) {
	location := json.RawMessage("null")
	var err error
	if p.api.dependencies.CloudLocation != nil {
		facts, readErr := p.api.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	return location, errors.Join(err, check(ctx))
}

// Existing association list projection supplies the same eight-field view.
// A private synthetic wire is only a projection input; it never becomes an
// actual Wire receipt, and each projected field/header owns its storage.
func projectAssociationRecord(fields map[string]json.RawMessage, namespace string, location json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, invalid("association location must be complete UTF-8 JSON")
	}
	seed := make(map[string]json.RawMessage, len(fields))
	for _, key := range associationRecordBodyFields {
		if raw, present := fields[key]; present {
			if !utf8.Valid(raw) || !json.Valid(raw) {
				return nil, invalid("association attribute %q must be complete UTF-8 JSON", key)
			}
			seed[key] = bytes.Clone(raw)
		}
	}
	row := &recordRow{Record: Record{Wire: &resource.RawResource{Metadata: resource.Metadata{
		Body: seed, Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode}}}}
	if err := prepareRecord(row, &namespace, location, nil); err != nil {
		return nil, err
	}
	return row.Resource, nil
}

func associationRecordFromResponse(ctx context.Context, check func(context.Context) error, seed map[string]json.RawMessage, namespace string, location json.RawMessage, response *rest.Response) (*Record, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(invalid("association response must be UTF-8"))
	}
	record := &Record{Namespace: copyPointer(&namespace), Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	// Source tolerates JSON decoding failure but a parsed nonobject has no
	// descriptor dictionary. Unknown object fields survive only in actual Wire.
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if err := json.Unmarshal(response.Body, wire); err != nil {
			return nil, response.Fail(err)
		}
		for _, key := range associationRecordBodyFields {
			if raw, present := wire.Body[key]; present {
				seed[key] = bytes.Clone(raw)
			}
		}
		record.Wire = wire
	}
	var err error
	record.Resource, err = projectAssociationRecord(seed, namespace, location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, response.Fail(err)
	}
	return record, nil
}

func recordMutationCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = 200 + index
	}
	return codes
}
func recordDeletionAcknowledgement(namespace, identity string, response *rest.Response) *Acknowledgement {
	if response == nil {
		return nil
	}
	return &Acknowledgement{Namespace: namespace, Name: copyPointer(&identity), Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
