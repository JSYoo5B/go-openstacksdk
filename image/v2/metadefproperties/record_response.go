package metadefproperties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// recordLocation captures the Connection facts before caller options execute.
// The fixed namespace URI keeps this location independent of response fields.
func (p *preparedSource) recordLocation(ctx context.Context, check func(context.Context) error) (json.RawMessage, error) {
	location := json.RawMessage("null")
	var err error
	if p.scope.api.dependencies.CloudLocation != nil {
		facts, readErr := p.scope.api.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	return location, errors.Join(err, check(ctx))
}

// propertyRecordFromResponse overlays present response fields on a private
// seed, then projects the declared descriptors. Source create/fetch/commit all
// tolerate JSON decoding failures, while a successfully parsed nonobject fails.
// Only an actual response contributes receipt metadata.
func propertyRecordFromResponse(ctx context.Context, check func(context.Context) error, seed map[string]json.RawMessage, namespace string, location json.RawMessage, response *rest.Response) (*Record, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(invalid("property response must be UTF-8"))
	}
	record := &Record{Namespace: namespace, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if err := json.Unmarshal(response.Body, wire); err != nil {
			return nil, response.Fail(err)
		}
		fields, err := normalizePropertyRecord(wire.Body, response.Body)
		if err != nil {
			return nil, response.Fail(err)
		}
		for key, raw := range fields {
			seed[key] = bytes.Clone(raw)
		}
		record.Wire = wire
	}
	var err error
	record.Resource, err = projectPropertyRecord(seed, namespace, location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, response.Fail(err)
	}
	return record, nil
}
