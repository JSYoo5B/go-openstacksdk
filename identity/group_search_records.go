package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GroupReadCloud binds cloud.search_groups to an already selected Keystone v3
// API. Construction performs no HTTP; SDK users implement no builders.
type GroupReadCloud struct{ api *groups.API }

func NewGroupReadCloud(api *groups.API) *GroupReadCloud { return &GroupReadCloud{api: api} }
func (c *GroupReadCloud) RawAPI() *groups.API {
	if c == nil {
		return nil
	}
	return c.api
}

// GroupSearchOpts separates the server domain query from local filters. Source
// consumes every page before inspecting filters, including invalid expressions.
type GroupSearchOpts struct {
	DomainID     *string
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type GroupSearchOption func(*GroupSearchOpts) error

func copyGroupSearchOpts(v GroupSearchOpts) GroupSearchOpts {
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
func WithGroupSearchOpts(v GroupSearchOpts) GroupSearchOption {
	owned := copyGroupSearchOpts(v)
	return func(c *GroupSearchOpts) error { *c = copyGroupSearchOpts(owned); return nil }
}
func WithGroupSearchDomainID(v string) GroupSearchOption {
	return func(c *GroupSearchOpts) error { x := v; c.DomainID = &x; return nil }
}
func WithGroupSearchFilters(v json.RawMessage) GroupSearchOption {
	owned := bytes.Clone(v)
	return func(c *GroupSearchOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithGroupSearchHeaders(v map[string]string) GroupSearchOption {
	owned := maps.Clone(v)
	return func(c *GroupSearchOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithGroupSearchMicroversion(v string) GroupSearchOption {
	return func(c *GroupSearchOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithGroupSearchLocation(v resource.CloudLocation) GroupSearchOption {
	owned := v.Clone()
	return func(c *GroupSearchOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}
func groupSearchInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// GroupSearchResult owns arbitrary expression JSON separately from selected
// records. Inventory preserves each actual row on later HTTP/local failure.
// Expression output has no invented association with an HTTP record.
type GroupSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Groups     []*groups.GroupRecord
	Inventory  []*groups.GroupRecord
}

func (c *GroupReadCloud) prepareGroupSearch(ctx context.Context, options []GroupSearchOption) (context.Context, *groups.API, GroupSearchOpts, func() error, error) {
	policy := GroupSearchOpts{}
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, policy, nil, err
	}
	if c == nil || c.api == nil {
		return ctx, nil, policy, nil, groupSearchInvalid("Keystone v3 group API is required")
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
			observed = groupSearchInvalid("selected group API changed")
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
			return ctx, nil, policy, guard, groupSearchInvalid("nil group search option")
		}
		next := copyGroupSearchOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return ctx, nil, policy, guard, err
		}
		policy = copyGroupSearchOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return ctx, nil, policy, guard, err
		}
	}
	return ctx, groups.New(&source.Client), policy, guard, guard()
}

// SearchGroups mirrors cloud.search_groups: list every group with only the
// domain server query, then exact/glob identifier and ordered local filtering.
// Truthy string filters run JMESPath on materialized Source views; the Proxy's
// opaque generator expression policy does not apply to this Cloud method.
func (c *GroupReadCloud) SearchGroups(ctx context.Context, nameOrID string, options ...GroupSearchOption) (*GroupSearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchGroups", "groups", cloudread.ContextError(ctx, err)) }
	owned, api, policy, guard, err := c.prepareGroupSearch(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	opts := make([]groups.RecordListOption, 0)
	var domain any
	if policy.DomainID != nil {
		domain = *policy.DomainID
	}
	opts = append(opts, groups.WithRecordListQuery("domain_id", domain))
	if policy.Headers != nil {
		opts = append(opts, groups.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		opts = append(opts, groups.WithRecordListMicroversion(*policy.Microversion))
	}
	if policy.Location != nil {
		opts = append(opts, groups.WithRecordListLocation(*policy.Location))
	}
	result := &GroupSearchResult{Inventory: make([]*groups.GroupRecord, 0)}
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
			return result, wrap(errors.Join(groupSearchInvalid("group search row projection"), err))
		}
		rows = append(rows, bytes.Clone(raw))
	}
	selected, err := cloudfilter.SelectLoaded(rows, nameOrID, policy.Filters, guard)
	if err != nil {
		return result, wrap(errors.Join(groupSearchInvalid("local group search"), err))
	}
	if err := guard(); err != nil {
		return result, wrap(err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Groups = make([]*groups.GroupRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Groups[i] = result.Inventory[index].Clone()
		}
	}
	return result, wrap(guard())
}
