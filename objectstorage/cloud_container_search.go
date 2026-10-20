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
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ContainerSearchOpts owns the cloud name/glob, JSON-domain mapping or JMESPath
// filter and location. Filters nil omits filtering; explicit JSON null, false,
// zero, empty string/list/dict follow Python truthiness after all pages load.
// Location nil uses already recorded native token scope without authentication.
type ContainerSearchOpts struct {
	Name     string
	Filters  *json.RawMessage
	Location *resource.CloudLocation
	Headers  map[string]string
}
type ContainerSearchOption func(*ContainerSearchOpts) error

func copyContainerSearchOpts(c ContainerSearchOpts) ContainerSearchOpts {
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
func WithContainerSearchOpts(value ContainerSearchOpts) ContainerSearchOption {
	value = copyContainerSearchOpts(value)
	return func(c *ContainerSearchOpts) error { *c = copyContainerSearchOpts(value); return nil }
}
func WithContainerSearchName(value string) ContainerSearchOption {
	return func(c *ContainerSearchOpts) error { c.Name = value; return nil }
}
func WithContainerSearchFiltersJSON(value json.RawMessage) ContainerSearchOption {
	owned := bytes.Clone(value)
	return func(c *ContainerSearchOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		c.Filters = &raw
		return nil
	}
}
func WithoutContainerSearchFilters() ContainerSearchOption {
	return func(c *ContainerSearchOpts) error { c.Filters = nil; return nil }
}
func WithContainerSearchExpression(value string) ContainerSearchOption {
	raw, _ := json.Marshal(value)
	return WithContainerSearchFiltersJSON(raw)
}
func WithContainerSearchLocation(value resource.CloudLocation) ContainerSearchOption {
	owned := value.Clone()
	return func(c *ContainerSearchOpts) error { snapshot := owned.Clone(); c.Location = &snapshot; return nil }
}
func WithContainerSearchHeaders(value map[string]string) ContainerSearchOption {
	owned := maps.Clone(value)
	return func(c *ContainerSearchOpts) error {
		if c.Headers == nil {
			c.Headers = map[string]string{}
		}
		maps.Copy(c.Headers, owned)
		return nil
	}
}

// ContainerSearchResult separates all actual rows/last page evidence from a
// successful selected view. Expression output may be a scalar, null, list or
// object and never invents an association with HTTP source rows.
type ContainerSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Records    []*containers.ContainerRecord
	AllRecords []*containers.ContainerRecord
	LastPage   *containers.ContainerRecord
}

func (c *Cloud) SearchContainers(ctx context.Context, options ...ContainerSearchOption) (*ContainerSearchResult, error) {
	const operation = "SearchContainers"
	fail := func(result *ContainerSearchResult, err error) (*ContainerSearchResult, error) {
		return result, containerCloudError(ctx, operation, err)
	}
	if ctx == nil {
		return fail(nil, fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if c == nil || c.containers == nil {
		return fail(nil, fmt.Errorf("%w: object-store cloud is required", resource.ErrInvalidOption))
	}
	ctx = rest.WithOperationSources(ctx)
	guard := func() error { return errors.Join(ctx.Err(), context.Cause(ctx), rest.CheckOperationGuard(ctx)) }
	owned := slices.Clone(options)
	cfg := copyContainerSearchOpts(ContainerSearchOpts{})
	var location json.RawMessage
	// The child captures and registers its account source before running this
	// library-owned option. No fake container identity or validation request is used.
	configure := func(list *containers.RecordListOpts) error {
		id, err := cloudlocation.ProjectID(c.client.ProviderClient)
		if err != nil {
			return err
		}
		defaultLocation := resource.CloudLocation{Project: resource.CloudProject{ID: id}}
		for _, option := range owned {
			if err := guard(); err != nil {
				return err
			}
			if option == nil {
				return fmt.Errorf("%w: nil container search option", resource.ErrInvalidOption)
			}
			next := copyContainerSearchOpts(cfg)
			if err := errors.Join(option(&next), guard()); err != nil {
				return err
			}
			cfg = copyContainerSearchOpts(next)
		}
		selected := defaultLocation
		if cfg.Location != nil {
			selected = cfg.Location.Clone()
		}
		location, err = selected.ForResource(nil, selected.Zone)
		if err != nil {
			return err
		}
		list.Headers = maps.Clone(cfg.Headers)
		return guard()
	}
	result := &ContainerSearchResult{}
	rows := []json.RawMessage{}
	for record, err := range c.containers.ListRecords(ctx, configure) {
		if record != nil {
			result.LastPage = copyContainerSearchRecord(record)
		}
		if err != nil {
			if record == nil {
				result.LastPage = containerSearchRejectedRecord(err)
			}
			return fail(result, err)
		}
		if err := guard(); err != nil {
			return fail(result, err)
		}
		snapshot := copyContainerSearchRecord(record)
		// The existing listing's bool descriptors retain exact-decimal policy.
		// Search uses Source JSON-loaded float truthiness without changing it.
		for _, key := range []string{"is_newest", "is_content_type_detected"} {
			if raw, present := snapshot.SeedHeaders[key]; present && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				truthy, err := cloudfilter.PythonJSONTruthy(raw)
				if err != nil {
					return fail(result, err)
				}
				snapshot.Resource.Body[key], _ = json.Marshal(truthy)
			}
		}
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
		result.Records = make([]*containers.ContainerRecord, len(selection.Indices))
		for i, index := range selection.Indices {
			result.Records[i] = copyContainerSearchRecord(result.AllRecords[index])
		}
	}
	if err := guard(); err != nil {
		result.Value = nil
		result.Records = nil
		return fail(result, err)
	}
	return result, nil
}

func copyContainerSearchRecord(value *containers.ContainerRecord) *containers.ContainerRecord {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Resource = copyContainerSearchRaw(value.Resource)
	copy.Wire = copyContainerSearchRaw(value.Wire)
	copy.Envelope = bytes.Clone(value.Envelope)
	copy.Header = value.Header.Clone()
	copy.Metadata = maps.Clone(value.Metadata)
	copy.SeedBody = copyContainerSearchFields(value.SeedBody)
	copy.SeedHeaders = copyContainerSearchFields(value.SeedHeaders)
	return &copy
}
func copyContainerSearchFields(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(value))
	for key, raw := range value {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func copyContainerSearchRaw(value *resource.RawResource) *resource.RawResource {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Body = copyContainerSearchFields(value.Body)
	copy.Header = value.Header.Clone()
	return &copy
}

func containerSearchRejectedRecord(err error) *containers.ContainerRecord {
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) {
		var renewal *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &renewal) || !errors.As(renewal.ErrOriginal, &native) {
			return nil
		}
	}
	return &containers.ContainerRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
}
