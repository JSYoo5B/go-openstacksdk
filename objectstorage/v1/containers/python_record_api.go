package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordStatuses() ([]int, []int) {
	accepted, rejected := make([]int, 300), make([]int, 200)
	for index := range accepted {
		accepted[index] = 100 + index
	}
	for index := range rejected {
		rejected[index] = 400 + index
	}
	return accepted, rejected
}
func recordJSON(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(encoded) {
		return nil, metadataInvalid("container attrs must encode UTF-8 JSON")
	}
	return json.RawMessage(bytes.Clone(encoded)), nil
}
func prepareRecordAttrs(name string, attrs map[string]any) (map[string]json.RawMessage, map[string]json.RawMessage, map[string]string, error) {
	body := make(map[string]json.RawMessage)
	headers := make(map[string]json.RawMessage)
	wire := make(map[string]string)
	body["name"], _ = json.Marshal(name)
	for key, value := range attrs {
		if key == "name" {
			return nil, nil, nil, metadataInvalid("name is already the required create argument")
		}
		if key == "id" {
			continue
		} // Container.new pops id; name was already supplied.
		if key == "metadata" {
			continue
		}
		if key == "count" || key == "bytes" {
			raw, err := recordJSON(value)
			if err != nil {
				return nil, nil, nil, err
			}
			body[key] = raw
			continue
		}
		decl, known := recordHeader(key)
		if !known {
			continue
		}
		if _, present := headers[decl.field]; present {
			return nil, nil, nil, metadataInvalid("ambiguous container attribute aliases %q", key)
		}
		raw, err := recordJSON(value)
		if err != nil {
			return nil, nil, nil, err
		}
		text, err := recordHeaderString(raw)
		if err != nil {
			return nil, nil, nil, err
		}
		if !metadataFieldValue(text) {
			return nil, nil, nil, metadataInvalid("invalid container attribute header %q", key)
		}
		headers[decl.field] = raw
		wire[http.CanonicalHeaderKey(decl.wire)] = text
	}
	if value, present := attrs["metadata"]; present {
		raw, err := recordJSON(value)
		if err != nil {
			return nil, nil, nil, err
		}
		if string(raw) != "null" {
			var values map[string]json.RawMessage
			if len(raw) == 0 || raw[0] != '{' {
				return nil, nil, nil, metadataInvalid("container metadata must be a JSON object or null")
			}
			if err := json.Unmarshal(raw, &values); err != nil {
				return nil, nil, nil, err
			}
			seen := make(map[string]bool)
			for key, value := range values {
				// BaseResource constructor first prefixes suffixes, then its
				// case-sensitive calculator runs. It differs from set_metadata.
				header := key
				if !strings.HasPrefix(strings.ToLower(header), "x-container-meta-") {
					header = "X-Container-Meta-" + header
				}
				if !strings.HasPrefix(header, "X-Container-Meta-") {
					header = "X-Container-Meta-" + header
				}
				text, err := cloudfilter.PythonString(value)
				if err != nil {
					return nil, nil, nil, err
				}
				if !metadataToken(header) || !metadataFieldValue(text) {
					return nil, nil, nil, metadataInvalid("invalid container metadata %q", key)
				}
				canonical := http.CanonicalHeaderKey(header)
				if seen[canonical] {
					return nil, nil, nil, metadataInvalid("ambiguous container metadata aliases %q", key)
				}
				seen[canonical] = true
				wire[canonical] = text
			}
		}
	}
	return body, headers, wire, nil
}

// CreateRecord implements Source container creation: bodyless PUT, generic
// non-error status acceptance and owned Body/Header seed-plus-response projection.
func (a *API) CreateRecord(ctx context.Context, name string, attrs map[string]any, options ...RecordCreateOption) (*ContainerRecord, error) {
	const operation = "CreateRecord"
	wrap := func(err error) error { return request.Wrap(operation, "containers", metadataContextError(ctx, err)) }
	p, err := a.captureMetadata(ctx, name)
	if err != nil {
		return nil, wrap(err)
	}
	bodySeed, headerSeed, headers, conversionErr := prepareRecordAttrs(name, attrs)
	if err = errors.Join(conversionErr, p.check(ctx)); err != nil {
		return nil, wrap(err)
	}
	cfg, err := applyRecordOptions(ctx, p, RecordCreateOpts{Headers: make(map[string]string)}, copyRecordCreate, slices.Clone(options))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, wrap(err)
	}
	// Source Resource headers override ordinary request headers; constructor
	// metadata overlays Source attrs without altering their stored seeds.
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	accepted, rejected := recordStatuses()
	response, err := rest.DoJSONGuardedRejectionsHeaders(ctx, p.client, p.check, http.MethodPut, p.target, nil, headers, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, wrap(err)
	}
	if err != nil {
		return &ContainerRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, SeedBody: cloneRecordFields(bodySeed), SeedHeaders: cloneRecordFields(headerSeed)}, wrap(err)
	}
	value, projectionErr := recordProjection(response, bodySeed, headerSeed)
	if projectionErr = errors.Join(projectionErr, p.check(ctx)); projectionErr != nil {
		return value, wrap(response.Fail(projectionErr))
	}
	return value, nil
}

// DeleteRecord follows Source's default clean404 ignore policy without admitting
// rejection status as success or suppressing rejected-body physical failures.
func (a *API) DeleteRecord(ctx context.Context, name string, options ...RecordDeleteOption) (*ContainerResponse, error) {
	const operation = "DeleteRecord"
	wrap := func(err error) error { return request.Wrap(operation, "containers", metadataContextError(ctx, err)) }
	p, err := a.captureMetadata(ctx, name)
	if err != nil {
		return nil, wrap(err)
	}
	cfg, err := applyRecordOptions(ctx, p, RecordDeleteOpts{Headers: make(map[string]string)}, copyRecordDelete, slices.Clone(options))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, wrap(err)
	}
	accepted, rejected := recordStatuses()
	response, err := rest.DoJSONGuardedRejections(ctx, p.client, p.check, http.MethodDelete, p.target, nil, nil, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	err = p.observe(ctx, response, err)
	if err != nil && (cfg.IgnoreMissing == nil || *cfg.IgnoreMissing) && resource.IgnoreMissing(err) == nil {
		var missing gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &missing) {
			return &ContainerResponse{Body: bytes.Clone(missing.Body), Header: missing.ResponseHeader.Clone(), StatusCode: missing.Actual, IgnoredMissing: true}, nil
		}
	}
	if response == nil {
		return nil, wrap(err)
	}
	return &ContainerResponse{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, wrap(err)
}
