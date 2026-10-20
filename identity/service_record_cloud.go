package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServiceReadCloud binds cloud.search_services/get_service to an already selected Keystone v3
// API. Construction performs no HTTP; SDK users implement no builders.
type ServiceReadCloud struct{ api *services.API }

func NewServiceReadCloud(api *services.API) *ServiceReadCloud { return &ServiceReadCloud{api: api} }
func (c *ServiceReadCloud) RawAPI() *services.API {
	if c == nil {
		return nil
	}
	return c.api
}

// ServiceReadOpts owns local filters and explicit transport conveniences. Source
// consumes every page before inspecting filters, including invalid expressions.
type ServiceReadOpts struct {
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type ServiceReadOption func(*ServiceReadOpts) error

func copyServiceReadOpts(v ServiceReadOpts) ServiceReadOpts {
	v.Headers = maps.Clone(v.Headers)

	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Filters != nil {
		x := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &x
	}
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	return v
}
func WithServiceReadOpts(v ServiceReadOpts) ServiceReadOption {
	owned := copyServiceReadOpts(v)
	return func(c *ServiceReadOpts) error { *c = copyServiceReadOpts(owned); return nil }
}
func WithServiceReadFilters(v json.RawMessage) ServiceReadOption {
	owned := bytes.Clone(v)
	return func(c *ServiceReadOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithServiceReadHeaders(v map[string]string) ServiceReadOption {
	owned := maps.Clone(v)
	return func(c *ServiceReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithServiceReadMicroversion(v string) ServiceReadOption {
	return func(c *ServiceReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithServiceReadLocation(v resource.CloudLocation) ServiceReadOption {
	owned := v.Clone()
	return func(c *ServiceReadOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}
func serviceReadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// ServiceSearchResult owns arbitrary expression JSON separately from selected
// records. Inventory preserves each actual row on later HTTP/local failure.
// Expression output has no invented association with an HTTP record.
type ServiceSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Services   []*services.ServiceRecord
	Inventory  []*services.ServiceRecord
}

func (c *ServiceReadCloud) prepareServiceRead(ctx context.Context, options []ServiceReadOption) (context.Context, *services.API, ServiceReadOpts, func() error, error) {
	policy := ServiceReadOpts{}
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, policy, nil, err
	}
	if c == nil || c.api == nil {
		return ctx, nil, policy, nil, serviceReadInvalid("Keystone v3 service API is required")
	}
	selected := c.api
	client := selected.RawClient()
	source, err := cloudread.Capture(ctx, client, "identity")
	if err != nil {
		return ctx, nil, policy, nil, err
	}
	previous := rest.OperationGuard(ctx)
	var observed error
	sourceGuard := func(current context.Context) error {
		if observed == nil && (c.api != selected || selected.RawClient() != client) {
			observed = serviceReadInvalid("selected service API changed")
		}
		var outer error
		if previous != nil {
			outer = previous(current)
		}
		return errors.Join(cloudread.ContextError(current, observed), source.Guard(current), outer)
	}
	ctx = rest.WithOperationGuard(ctx, sourceGuard)
	guard := func() error { return sourceGuard(ctx) }
	for _, apply := range slices.Clone(options) {
		if err := guard(); err != nil {
			return ctx, nil, policy, guard, err
		}
		if apply == nil {
			return ctx, nil, policy, guard, serviceReadInvalid("nil service read option")
		}
		next := copyServiceReadOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return ctx, nil, policy, guard, err
		}
		policy = copyServiceReadOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return ctx, nil, policy, guard, err
		}
	}
	return ctx, services.New(&source.Client), policy, guard, guard()
}

// SearchServices mirrors cloud.search_services: consume a queryless list, then exact/glob identifier and ordered local filtering.
// Truthy string filters run JMESPath on materialized Source views; the Proxy's
// opaque generator expression policy does not apply to this Cloud method.
func (c *ServiceReadCloud) SearchServices(ctx context.Context, nameOrID string, options ...ServiceReadOption) (*ServiceSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServices", "services", cloudread.ContextError(ctx, err))
	}
	owned, api, policy, guard, err := c.prepareServiceRead(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	return searchServicesPrepared(owned, api, nameOrID, policy, guard)
}

func searchServicesPrepared(ctx context.Context, api *services.API, nameOrID string, policy ServiceReadOpts, guard func() error) (*ServiceSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServices", "services", cloudread.ContextError(ctx, err))
	}
	opts := make([]services.RecordListOption, 0)

	if policy.Headers != nil {
		opts = append(opts, services.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		opts = append(opts, services.WithRecordListMicroversion(*policy.Microversion))
	}
	if policy.Location != nil {
		opts = append(opts, services.WithRecordListLocation(*policy.Location))
	}
	result := &ServiceSearchResult{Inventory: make([]*services.ServiceRecord, 0)}
	rows := make([]json.RawMessage, 0)
	for record, err := range api.ListRecords(ctx, opts...) {
		if record != nil {
			result.Inventory = append(result.Inventory, record)
		}
		if err != nil {
			return result, wrap(errors.Join(err, guard()))
		}
		if err := guard(); err != nil {
			return result, wrap(err)
		}
		raw, err := json.Marshal(record.Resource)
		if err != nil {
			return result, wrap(errors.Join(serviceReadInvalid("service read row projection"), err))
		}
		rows = append(rows, bytes.Clone(raw))
	}
	selected, err := cloudfilter.SelectLoaded(rows, nameOrID, policy.Filters, guard)
	if err != nil {
		return result, wrap(errors.Join(serviceReadInvalid("local service read"), err))
	}
	if err := guard(); err != nil {
		return result, wrap(err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Services = make([]*services.ServiceRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Services[i] = result.Inventory[index].Clone()
		}
	}
	return result, wrap(guard())
}

// ServiceGetResult keeps the selected record separate from arbitrary deprecated
// expression output. Inventory belongs only to actual HTTP rows. Value is JSON
// null when Source returns None. DeprecatedFilters records the non-None branch;
// warnings are represented by this field rather than process-global output.
type ServiceGetResult struct {
	Value             json.RawMessage
	Service           *services.ServiceRecord
	Inventory         []*services.ServiceRecord
	Expression        bool
	DeprecatedFilters bool
}

// GetService preserves the pinned Cloud's two paths. Omitted/null filters use
// the closed-signature GET-first finder. Every other value, including {}, uses
// queryless search before Source truthiness, len and numeric-index selection.
func (c *ServiceReadCloud) GetService(ctx context.Context, nameOrID string, options ...ServiceReadOption) (*ServiceGetResult, error) {
	wrap := func(err error) error { return request.Wrap("GetService", "services", cloudread.ContextError(ctx, err)) }
	owned, api, policy, guard, err := c.prepareServiceRead(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result := &ServiceGetResult{Inventory: make([]*services.ServiceRecord, 0)}
	if policy.Filters == nil || bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		opts := []services.RecordResolveOption{services.WithRecordResolveOpts(services.RecordResolveOpts{Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location})}
		record, err := api.ResolveRecord(owned, nameOrID, opts...)
		if record != nil {
			result.Inventory = append(result.Inventory, record)
			result.Service = record.Clone()
		}
		if err != nil {
			return result, wrap(errors.Join(err, guard()))
		}
		result.Value = json.RawMessage("null")
		if record != nil {
			result.Value, err = json.Marshal(record.Resource)
			if err != nil {
				return result, wrap(errors.Join(serviceReadInvalid("service result JSON"), err, guard()))
			}
		}
		return result, wrap(guard())
	}
	result.DeprecatedFilters = true
	searched, err := searchServicesPrepared(owned, api, nameOrID, policy, guard)
	if searched != nil {
		result.Inventory = searched.Inventory
		result.Expression = searched.Expression
	}
	if err != nil {
		return result, wrap(errors.Join(err, guard()))
	}
	truthy, err := cloudfilter.PythonJSONTruthy(searched.Value)
	if err != nil {
		return result, wrap(errors.Join(serviceReadInvalid("service result truthiness"), err, guard()))
	}
	if !truthy {
		result.Value = json.RawMessage("null")
		return result, wrap(guard())
	}
	raw := bytes.TrimSpace(searched.Value)
	var first json.RawMessage
	amount := 0
	switch raw[0] {
	case '[':
		var values []json.RawMessage
		if err = json.Unmarshal(raw, &values); err != nil {
			return result, wrap(errors.Join(serviceReadInvalid("service result array"), err, guard()))
		}
		amount = len(values)
		if amount == 1 {
			first = bytes.Clone(values[0])
		}
	case '"':
		var value string
		if err = json.Unmarshal(raw, &value); err != nil {
			return result, wrap(errors.Join(err, guard()))
		}
		amount = utf8.RuneCountInString(value)
		if amount == 1 {
			first, _ = json.Marshal(value)
		}
	case '{':
		values, cause := cloudfilter.ObjectMembers(raw)
		if cause != nil {
			return result, wrap(errors.Join(cause, guard()))
		}
		amount = len(values)
		if amount == 1 {
			return result, wrap(errors.Join(serviceReadInvalid("Source get_service cannot index an expression mapping with integer zero"), guard()))
		}
	default:
		return result, wrap(errors.Join(serviceReadInvalid("Source get_service cannot take len of scalar expression result"), guard()))
	}
	if amount > 1 {
		ids := make([]string, 0, len(searched.Services))
		for _, record := range searched.Services {
			var id string
			_ = json.Unmarshal(record.Resource.Body["id"], &id)
			ids = append(ids, id)
		}
		return result, wrap(errors.Join(&resource.AmbiguousError{Resource: "services", Name: nameOrID, IDs: ids}, guard()))
	}
	result.Value = bytes.Clone(first)
	if !searched.Expression && len(searched.Services) == 1 {
		result.Service = searched.Services[0].Clone()
	}
	return result, wrap(guard())
}
