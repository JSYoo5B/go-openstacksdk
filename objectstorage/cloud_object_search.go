package objectstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ObjectSearchOpts owns the cloud name/glob, JSON-domain mapping or JMESPath
// filter and location. Filters nil omits filtering; explicit JSON null, false,
// zero, empty string/list/dict follow Python truthiness after all pages load.
// Location nil preserves the direct Source Object location=null. Connection
// wrappers supply configured current location without a per-row builder.
type ObjectSearchOpts struct {
	Name     string
	Filters  *json.RawMessage
	Location *resource.CloudLocation
	Headers  map[string]string
}
type ObjectSearchOption func(*ObjectSearchOpts) error

func copyObjectSearchOpts(c ObjectSearchOpts) ObjectSearchOpts {
	c.Headers = maps.Clone(c.Headers)
	if c.Headers == nil {
		c.Headers = map[string]string{}
	}
	if c.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*c.Filters))
		c.Filters = &raw
	}
	if c.Location != nil {
		value := c.Location.Clone()
		c.Location = &value
	}
	return c
}
func WithObjectSearchOpts(value ObjectSearchOpts) ObjectSearchOption {
	value = copyObjectSearchOpts(value)
	return func(c *ObjectSearchOpts) error { *c = copyObjectSearchOpts(value); return nil }
}
func WithObjectSearchName(value string) ObjectSearchOption {
	return func(c *ObjectSearchOpts) error { c.Name = value; return nil }
}
func WithObjectSearchFiltersJSON(value json.RawMessage) ObjectSearchOption {
	owned := bytes.Clone(value)
	return func(c *ObjectSearchOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		c.Filters = &raw
		return nil
	}
}
func WithoutObjectSearchFilters() ObjectSearchOption {
	return func(c *ObjectSearchOpts) error { c.Filters = nil; return nil }
}
func WithObjectSearchExpression(value string) ObjectSearchOption {
	raw, _ := json.Marshal(value)
	return WithObjectSearchFiltersJSON(raw)
}
func WithObjectSearchLocation(value resource.CloudLocation) ObjectSearchOption {
	owned := value.Clone()
	return func(c *ObjectSearchOpts) error { snapshot := owned.Clone(); c.Location = &snapshot; return nil }
}
func WithObjectSearchHeaders(value map[string]string) ObjectSearchOption {
	owned := maps.Clone(value)
	return func(c *ObjectSearchOpts) error {
		if c.Headers == nil {
			c.Headers = map[string]string{}
		}
		maps.Copy(c.Headers, owned)
		return nil
	}
}

// ObjectSearchResult separates all actual rows/last page evidence from a
// successful selected view. Expression output may be a scalar, null, list or
// object and never invents an association with HTTP source rows.
type ObjectSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Records    []*objects.ObjectRecord
	AllRecords []*objects.ObjectRecord
	LastPage   *objects.ObjectRecord
}

func (c *Cloud) SearchObjects(ctx context.Context, container string, options ...ObjectSearchOption) (*ObjectSearchResult, error) {
	const operation = "SearchObjects"
	fail := func(result *ObjectSearchResult, err error) (*ObjectSearchResult, error) {
		return result, request.Wrap(operation, "objects", cloudread.ContextError(ctx, err))
	}
	if ctx == nil {
		return fail(nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if c == nil {
		return fail(nil, fmt.Errorf("%w: object-store cloud is required", resource.ErrInvalidOption))
	}
	ctx = rest.WithOperationSources(ctx)
	guard := func() error { return errors.Join(ctx.Err(), context.Cause(ctx), rest.CheckOperationGuard(ctx)) }
	owned := slices.Clone(options)
	cfg := copyObjectSearchOpts(ObjectSearchOpts{})
	var location json.RawMessage
	// The child captures and registers its account source before running this
	// library-owned option. The literal container route is captured before callbacks.
	configure := func(list *objects.RecordListOpts) error {
		for _, option := range owned {
			if err := guard(); err != nil {
				return err
			}
			if option == nil {
				return fmt.Errorf("%w: nil object search option", resource.ErrInvalidOption)
			}
			next := copyObjectSearchOpts(cfg)
			if err := errors.Join(option(&next), guard()); err != nil {
				return err
			}
			cfg = copyObjectSearchOpts(next)
		}
		location = json.RawMessage("null")
		if cfg.Location != nil {
			var err error
			location, err = cfg.Location.ForResource(nil, cfg.Location.Zone)
			if err != nil {
				return err
			}
		}
		list.Headers = maps.Clone(cfg.Headers)
		return guard()
	}
	result := &ObjectSearchResult{}
	rows := []json.RawMessage{}
	for record, err := range objects.New(c.client).ListRecords(ctx, container, configure) {
		if record != nil {
			result.LastPage = copyObjectSearchRecord(record)
		}
		if err != nil {
			if record == nil {
				result.LastPage = objectSearchRejectedRecord(container, err)
			}
			return fail(result, err)
		}
		if err := guard(); err != nil {
			return fail(result, err)
		}
		snapshot := copyObjectSearchRecord(record)
		snapshot.Resource.Body["location"] = bytes.Clone(location)
		raw, err := json.Marshal(snapshot.Resource.Body)
		if err != nil {
			return fail(result, err)
		}
		result.AllRecords = append(result.AllRecords, snapshot)
		rows = append(rows, raw)
	}
	selection, err := cloudfilter.SelectLoaded(rows, cfg.Name, cfg.Filters, guard)
	if err != nil {
		return fail(result, err)
	}
	result.Value = bytes.Clone(selection.Value)
	result.Expression = selection.Expression
	if !selection.Expression {
		result.Records = make([]*objects.ObjectRecord, len(selection.Indices))
		for i, index := range selection.Indices {
			result.Records[i] = copyObjectSearchRecord(result.AllRecords[index])
		}
	}
	if err := guard(); err != nil {
		result.Value = nil
		result.Records = nil
		return fail(result, err)
	}
	return result, nil
}

func copyObjectSearchRecord(value *objects.ObjectRecord) *objects.ObjectRecord {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Resource = copyContainerSearchRaw(value.Resource)
	copy.Wire = copyContainerSearchRaw(value.Wire)
	copy.Metadata = copyContainerSearchFields(value.Metadata)
	copy.Envelope = bytes.Clone(value.Envelope)
	copy.Header = value.Header.Clone()
	return &copy
}
func objectSearchRejectedRecord(container string, err error) *objects.ObjectRecord {
	actual := containerSearchRejectedRecord(err)
	if actual == nil {
		return nil
	}
	return &objects.ObjectRecord{Container: container, Envelope: bytes.Clone(actual.Envelope), Header: actual.Header.Clone(), StatusCode: actual.StatusCode}
}
