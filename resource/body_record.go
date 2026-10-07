package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// BodyRecord pairs a native typed value with the original JSON fields of the
// same page row. It is an SDK binding detail; collections still return *T.
// Fields own their bytes and preserve numbers, nulls and extension fields.
type BodyRecord[T any] struct {
	Value  T
	Fields map[string]json.RawMessage
}

type BodyFieldType uint8

const (
	// BodyFieldJSON retains the decoded response value without coercion.
	BodyFieldJSON BodyFieldType = iota
	// BodyFieldInteger applies an audited integer response descriptor.
	BodyFieldInteger
	// BodyFieldBoolean applies audited response-only JSON truthiness.
	BodyFieldBoolean
)

// BodyRecordField projects one original response field. Omission and null both
// match a null filter; an empty string, list or object remains distinct. Integer
// descriptors accept exact integer JSON numbers and decimal integer strings.
// Boolean descriptors preserve null and normalize JSON truthiness without
// changing caller filter types. The returned bytes are independent of the record.
func BodyRecordField(fields map[string]json.RawMessage, key string, kind BodyFieldType) (json.RawMessage, error) {
	raw, exists := fields[key]
	if !exists {
		raw = json.RawMessage("null")
	}
	switch kind {
	case BodyFieldJSON:
		var owned json.RawMessage
		if err := json.Unmarshal(raw, &owned); err != nil {
			return nil, fmt.Errorf("%w: response JSON field %q: %w", ErrInvalidOption, key, err)
		}
		return owned, nil
	case BodyFieldInteger:
		normalized, err := jsonfilter.IntegerJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: response integer field %q: %w", ErrInvalidOption, key, err)
		}
		return normalized, nil
	case BodyFieldBoolean:
		normalized, err := jsonfilter.BooleanJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: response boolean field %q: %w", ErrInvalidOption, key, err)
		}
		return normalized, nil
	default:
		return nil, invalid("unknown Body field type %d", kind)
	}
}

// BodyStreamWithControl preserves native page validation and extraction before
// pairing rows by index. It reads the pager's already decoded original body,
// without refetching or converting json.Number values through float64. Only
// audited bindings whose extractor preserves row count and order may use it.
// Row caps, early break and continuation policy are shared with typed streams.
func BodyStreamWithControl[T any](ctx context.Context, pager pagination.Pager, extract func(pagination.Page) ([]T, error), envelope string, control ListControl) iter.Seq2[*BodyRecord[T], error] {
	return StreamWithControl(ctx, pager, func(page pagination.Page) ([]BodyRecord[T], error) {
		values, err := extract(page)
		if err != nil {
			return nil, err
		}
		return pairBodyRecords(values, page.GetBody(), envelope)
	}, control)
}

func pairBodyRecords[T any](values []T, body any, envelope string) ([]BodyRecord[T], error) {
	if envelope == "" {
		return nil, invalid("Body record envelope must not be empty")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: Body record page JSON: %w", ErrInvalidOption, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("%w: Body record page object: %w", ErrInvalidOption, err)
	}
	rowsJSON, exists := fields[envelope]
	if !exists {
		return nil, invalid("Body record page is missing envelope %q", envelope)
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(rowsJSON, &rows); err != nil {
		return nil, fmt.Errorf("%w: Body record envelope %q: %w", ErrInvalidOption, envelope, err)
	}
	if len(rows) != len(values) {
		return nil, invalid("Body record envelope %q has %d rows for %d native values", envelope, len(rows), len(values))
	}
	records := make([]BodyRecord[T], len(values))
	for i, row := range rows {
		var original map[string]json.RawMessage
		if err := json.Unmarshal(row, &original); err != nil {
			return nil, fmt.Errorf("%w: Body record row %d: %w", ErrInvalidOption, i, err)
		}
		records[i] = BodyRecord[T]{Value: values[i], Fields: original}
	}
	return records, nil
}
