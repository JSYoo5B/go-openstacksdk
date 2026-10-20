package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RoleReadCloud binds cloud.search_roles to an already selected Keystone v3
// API. Construction performs no HTTP; SDK users implement no builders.
type RoleReadCloud struct{ api *roles.API }

func NewRoleReadCloud(api *roles.API) *RoleReadCloud { return &RoleReadCloud{api: api} }
func (c *RoleReadCloud) RawAPI() *roles.API {
	if c == nil {
		return nil
	}
	return c.api
}

// RoleSearchOpts separates the server domain query from local filters. Source
// consumes every page before inspecting filters, including invalid expressions.
type RoleSearchOpts struct {
	DomainID     *string
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type RoleSearchOption func(*RoleSearchOpts) error

func copyRoleSearchOpts(v RoleSearchOpts) RoleSearchOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.DomainID != nil {
		x := *v.DomainID
		v.DomainID = &x
	}
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
func WithRoleSearchOpts(v RoleSearchOpts) RoleSearchOption {
	owned := copyRoleSearchOpts(v)
	return func(c *RoleSearchOpts) error { *c = copyRoleSearchOpts(owned); return nil }
}
func WithRoleSearchDomainID(v string) RoleSearchOption {
	return func(c *RoleSearchOpts) error { x := v; c.DomainID = &x; return nil }
}
func WithRoleSearchFilters(v json.RawMessage) RoleSearchOption {
	owned := bytes.Clone(v)
	return func(c *RoleSearchOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithRoleSearchHeaders(v map[string]string) RoleSearchOption {
	owned := maps.Clone(v)
	return func(c *RoleSearchOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRoleSearchMicroversion(v string) RoleSearchOption {
	return func(c *RoleSearchOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRoleSearchLocation(v resource.CloudLocation) RoleSearchOption {
	owned := v.Clone()
	return func(c *RoleSearchOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}
func roleSearchInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// RoleSearchResult owns arbitrary expression JSON separately from selected
// records. Inventory preserves each actual row on later HTTP/local failure.
// Expression output has no invented association with an HTTP record.
type RoleSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Roles      []*roles.RoleRecord
	Inventory  []*roles.RoleRecord
}

func (c *RoleReadCloud) prepareRoleSearch(ctx context.Context, options []RoleSearchOption) (context.Context, *roles.API, RoleSearchOpts, func() error, error) {
	policy := RoleSearchOpts{}
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, policy, nil, err
	}
	if c == nil || c.api == nil {
		return ctx, nil, policy, nil, roleSearchInvalid("Keystone v3 role API is required")
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
			observed = roleSearchInvalid("selected role API changed")
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
			return ctx, nil, policy, guard, roleSearchInvalid("nil role search option")
		}
		next := copyRoleSearchOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return ctx, nil, policy, guard, err
		}
		policy = copyRoleSearchOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return ctx, nil, policy, guard, err
		}
	}
	return ctx, roles.New(&source.Client), policy, guard, guard()
}

// SearchRoles mirrors cloud.search_roles: list every role with only the
// domain server query, then exact/glob identifier and ordered local filtering.
// Truthy string filters run JMESPath on materialized Source views; the Proxy's
// opaque generator expression policy does not apply to this Cloud method.
func (c *RoleReadCloud) SearchRoles(ctx context.Context, nameOrID string, options ...RoleSearchOption) (*RoleSearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchRoles", "roles", cloudread.ContextError(ctx, err)) }
	owned, api, policy, guard, err := c.prepareRoleSearch(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	opts := make([]roles.RecordListOption, 0)
	var domain any
	if policy.DomainID != nil {
		domain = *policy.DomainID
	}
	opts = append(opts, roles.WithRecordListQuery("domain_id", domain))
	if policy.Headers != nil {
		opts = append(opts, roles.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		opts = append(opts, roles.WithRecordListMicroversion(*policy.Microversion))
	}
	if policy.Location != nil {
		opts = append(opts, roles.WithRecordListLocation(*policy.Location))
	}
	result := &RoleSearchResult{Inventory: make([]*roles.RoleRecord, 0)}
	rows := make([]json.RawMessage, 0)
	for record, err := range api.ListRecords(owned, opts...) {
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
			return result, wrap(errors.Join(roleSearchInvalid("role search row projection"), err))
		}
		rows = append(rows, bytes.Clone(raw))
	}
	selected, err := cloudfilter.SelectLoaded(rows, nameOrID, policy.Filters, guard)
	if err != nil {
		return result, wrap(errors.Join(roleSearchInvalid("local role search"), err))
	}
	if err := guard(); err != nil {
		return result, wrap(err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Roles = make([]*roles.RoleRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Roles[i] = result.Inventory[index].Clone()
		}
	}
	return result, wrap(guard())
}

// RoleListResult retains cloud.list_roles' list() value, including JSON literal
// and opaque-generator branches of Source Proxy._list. JSON is present only
// when the whole value is representable; no HTTP association is synthesized.
type RoleListResult struct {
	Value     []any
	JSON      json.RawMessage
	Roles     []*roles.RoleRecord
	Inventory []*roles.RoleRecord
}

// ListRoles consumes Source Proxy.roles with library-owned list options. Real
// Role rows retain independently copied views/evidence; a later failure keeps
// prior Inventory, while a literal Proxy branch has no fabricated Role receipt.
func (c *RoleReadCloud) ListRoles(ctx context.Context, options ...roles.RecordListOption) (*RoleListResult, error) {
	wrap := func(err error) error { return request.Wrap("ListRoles", "roles", cloudread.ContextError(ctx, err)) }
	owned, api, _, guard, err := c.prepareRoleSearch(ctx, nil)
	if err != nil {
		return nil, wrap(err)
	}
	proxy, err := api.ProxyListRecords(owned, slices.Clone(options)...)
	if err != nil {
		return nil, wrap(errors.Join(err, guard()))
	}
	result := &RoleListResult{Value: make([]any, 0), Inventory: make([]*roles.RoleRecord, 0)}
	switch value := proxy.Value.(type) {
	case *roles.RecordListGenerator:
		result.Roles = make([]*roles.RoleRecord, 0)
		for record, err := range value.Records {
			if record != nil {
				result.Inventory = append(result.Inventory, record)
			}
			if err != nil {
				return result, wrap(errors.Join(err, guard()))
			}
			if err := guard(); err != nil {
				return result, wrap(err)
			}
			result.Roles = append(result.Roles, record.Clone())
			result.Value = append(result.Value, record.Resource.Clone())
		}
	case []any:
		result.Value = slices.Clone(value)
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			result.Value = append(result.Value, key)
		}
	case string:
		for _, char := range value {
			result.Value = append(result.Value, string(char))
		}
	default:
		return result, wrap(roleSearchInvalid("list_roles cannot iterate Source Proxy result %T", proxy.Value))
	}
	if err := guard(); err != nil {
		return result, wrap(err)
	}
	if raw, err := json.Marshal(result.Value); err == nil {
		result.JSON = bytes.Clone(raw)
	}
	return result, wrap(guard())
}
