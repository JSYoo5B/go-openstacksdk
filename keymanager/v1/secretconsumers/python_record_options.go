package secretconsumers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordMutationOpts supplies Source new() attributes. Missing fields stay
// absent from the request; null, false, zero and untyped JSON remain distinct.
// Unknown semantic attributes are discarded before JSON encoding.
type RecordMutationOpts struct {
	Attributes    map[string]any
	IgnoreMissing *bool
}
type CreateRecordOpts = RecordMutationOpts
type DeleteRecordOpts = RecordMutationOpts
type CreateRecordOption = request.Option[RecordMutationOpts]
type DeleteRecordOption = request.Option[RecordMutationOpts]

func WithCreateRecordOptions(value CreateRecordOpts) CreateRecordOption {
	return mutationRecordOptions(value)
}
func WithDeleteRecordOptions(value DeleteRecordOpts) DeleteRecordOption {
	return mutationRecordOptions(value)
}
func WithCreateRecordAttributes(value map[string]any) CreateRecordOption {
	return mutationRecordOptions(RecordMutationOpts{Attributes: value})
}
func WithDeleteRecordAttributes(value map[string]any) DeleteRecordOption {
	return mutationRecordOptions(RecordMutationOpts{Attributes: value})
}
func WithCreateRecordAttribute(key string, value any) CreateRecordOption {
	return mutationRecordAttribute(key, value)
}
func WithDeleteRecordAttribute(key string, value any) DeleteRecordOption {
	return mutationRecordAttribute(key, value)
}
func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return request.WithHeader[RecordMutationOpts](key, value)
}
func WithDeleteRecordHeader(key, value string) DeleteRecordOption {
	return request.WithHeader[RecordMutationOpts](key, value)
}
func WithDeleteRecordIgnoreMissing(value bool) DeleteRecordOption {
	return func(config *request.Config[RecordMutationOpts]) error {
		owned := value
		config.Options.IgnoreMissing = &owned
		return nil
	}
}

func mutationRecordOptions(value RecordMutationOpts) request.Option[RecordMutationOpts] {
	fields, prior := recordAttributes(value.Attributes, false)
	var missing *bool
	if value.IgnoreMissing != nil {
		copied := *value.IgnoreMissing
		missing = &copied
	}
	return func(config *request.Config[RecordMutationOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeRecordAttributes(fields)
		if err != nil {
			return err
		}
		config.Options = RecordMutationOpts{Attributes: owned}
		if missing != nil {
			copied := *missing
			config.Options.IgnoreMissing = &copied
		}
		return nil
	}
}
func mutationRecordAttribute(key string, value any) request.Option[RecordMutationOpts] {
	fields, prior := recordAttributes(map[string]any{key: value}, false)
	return func(config *request.Config[RecordMutationOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeRecordAttributes(fields)
		if err != nil {
			return err
		}
		if config.Options.Attributes == nil {
			config.Options.Attributes = make(map[string]any)
		}
		maps.Copy(config.Options.Attributes, owned)
		return nil
	}
}

// ListRecordOpts classifies the five declared Body descriptors as local
// filters; only inherited limit and marker are wire query parameters.
// Unknown names, including offset, are discarded, matching the pinned Source.
type ListRecordOpts struct {
	Query     map[string]any
	MaxItems  int
	Paginated *bool
}
type ListRecordOption = request.Option[ListRecordOpts]

func WithListRecordOptions(value ListRecordOpts) ListRecordOption {
	fields, prior := recordAttributes(value.Query, true)
	var paginated *bool
	if value.Paginated != nil {
		copied := *value.Paginated
		paginated = &copied
	}
	cap := value.MaxItems
	return func(config *request.Config[ListRecordOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeRecordAttributes(fields)
		if err != nil {
			return err
		}
		config.Options = ListRecordOpts{Query: owned, MaxItems: cap}
		if paginated != nil {
			copied := *paginated
			config.Options.Paginated = &copied
		}
		return nil
	}
}
func WithListRecordQuery(value map[string]any) ListRecordOption {
	option := WithListRecordOptions(ListRecordOpts{Query: value})
	return func(config *request.Config[ListRecordOpts]) error {
		keep := config.Options
		if err := option(config); err != nil {
			return err
		}
		config.Options.MaxItems, config.Options.Paginated = keep.MaxItems, keep.Paginated
		return nil
	}
}
func WithListRecordMaxItems(value int) ListRecordOption {
	return func(config *request.Config[ListRecordOpts]) error { config.Options.MaxItems = value; return nil }
}
func WithListRecordPaginated(value bool) ListRecordOption {
	return func(config *request.Config[ListRecordOpts]) error {
		copied := value
		config.Options.Paginated = &copied
		return nil
	}
}
func WithListRecordHeader(key, value string) ListRecordOption {
	return request.WithHeader[ListRecordOpts](key, value)
}

func recordBodyField(key string) bool {
	switch key {
	case "id", "name", "service", "resource_type", "resource_id":
		return true
	}
	return false
}
func recordAttributes(values map[string]any, query bool) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	for key, value := range values {
		if !recordBodyField(key) && !(query && (key == "limit" || key == "marker")) {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("%w: consumer attribute %q: %w", resource.ErrInvalidOption, key, err)
		}
		if !utf8.Valid(raw) {
			return nil, recordInvalid("consumer attributes must be UTF-8 JSON")
		}
		// encoding/json replaces invalid UTF-8 strings. Reject those before it
		// can silently change a selected request value.
		if err := recordUTF8(value); err != nil {
			return nil, err
		}
		fields[key] = bytes.Clone(raw)
	}
	return fields, nil
}
func recordUTF8(value any) error {
	switch value := value.(type) {
	case string:
		if !utf8.ValidString(value) {
			return recordInvalid("consumer attributes must contain valid UTF-8")
		}
	case map[string]any:
		for key, item := range value {
			if !utf8.ValidString(key) {
				return recordInvalid("consumer keys must contain valid UTF-8")
			}
			if err := recordUTF8(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range value {
			if err := recordUTF8(item); err != nil {
				return err
			}
		}
	}
	return nil
}
func decodeRecordAttributes(fields map[string]json.RawMessage) (map[string]any, error) {
	result := make(map[string]any, len(fields))
	for key, raw := range fields {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}
func recordInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}
func recordCloneFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		result[key] = bytes.Clone(raw)
	}
	return result
}
func ownRecordConfig[T any](config *request.Config[T]) {
	config.Fields = recordCloneFields(config.Fields)
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	config.Query = maps.Clone(config.Query)
	if config.Query == nil {
		config.Query = make(map[string][]string)
	}
	for key, values := range config.Query {
		config.Query[key] = slices.Clone(values)
	}
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}
func recordHeaderPolicy(headers map[string]string) error {
	for key := range headers {
		switch strings.ToLower(key) {
		case "content-type", "accept", "openstack-api-version", "x-openstack-nova-api-version", "x-openstack-volume-api-version":
			return recordInvalid("consumer content/version headers are SDK owned")
		}
	}
	return nil
}
