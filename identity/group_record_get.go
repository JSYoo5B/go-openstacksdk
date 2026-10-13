package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
	"slices"
)

type GroupGetOpts struct {
	DomainID     json.RawMessage
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type GroupGetOption func(*GroupGetOpts) error

func copyGroupGetOpts(v GroupGetOpts) GroupGetOpts {
	v.DomainID = bytes.Clone(v.DomainID)
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.Filters != nil {
		x := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	return v
}
func WithGroupGetOpts(v GroupGetOpts) GroupGetOption {
	owned := copyGroupGetOpts(v)
	return func(c *GroupGetOpts) error { *c = copyGroupGetOpts(owned); return nil }
}
func WithGroupGetDomainID(v string) GroupGetOption {
	raw, _ := json.Marshal(v)
	return WithGroupGetRawDomainID(raw)
}
func WithGroupGetRawDomainID(v json.RawMessage) GroupGetOption {
	owned := bytes.Clone(v)
	return func(c *GroupGetOpts) error { c.DomainID = bytes.Clone(owned); return nil }
}
func WithGroupGetFilters(v json.RawMessage) GroupGetOption {
	owned := bytes.Clone(v)
	return func(c *GroupGetOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithGroupGetHeaders(v map[string]string) GroupGetOption {
	owned := maps.Clone(v)
	return func(c *GroupGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithGroupGetMicroversion(v string) GroupGetOption {
	return func(c *GroupGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithGroupGetLocation(v resource.CloudLocation) GroupGetOption {
	owned := v.Clone()
	return func(c *GroupGetOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

type GroupGetResult struct {
	Value                         json.RawMessage
	Group                         *groups.GroupRecord
	Inventory                     []*groups.GroupRecord
	DeprecatedFilters, Expression bool
}

func groupGetSearch(ctx context.Context, api *groups.API, id string, policy GroupGetOpts, guard func() error) (*GroupSearchResult, error) {
	domain := json.RawMessage(bytes.Clone(policy.DomainID))
	if domain == nil {
		domain = json.RawMessage("null")
	}
	options := []groups.RecordListOption{groups.WithRecordListQuery("domain_id", domain), groups.WithRecordListHeaders(policy.Headers)}
	if policy.Microversion != nil {
		options = append(options, groups.WithRecordListMicroversion(*policy.Microversion))
	}
	if policy.Location != nil {
		options = append(options, groups.WithRecordListLocation(*policy.Location))
	}
	result := &GroupSearchResult{Inventory: make([]*groups.GroupRecord, 0)}
	rows := make([]json.RawMessage, 0)
	for record, err := range api.ListRecords(ctx, options...) {
		if record != nil {
			result.Inventory = append(result.Inventory, record.Clone())
		}
		if err != nil {
			return result, errors.Join(err, guard())
		}
		if err := guard(); err != nil {
			return result, err
		}
		raw, err := json.Marshal(record.Resource)
		if err != nil {
			return result, err
		}
		rows = append(rows, bytes.Clone(raw))
	}
	selected, err := cloudfilter.SelectLoaded(rows, id, policy.Filters, guard)
	if err != nil {
		return result, errors.Join(groupSearchInvalid("local get_group search"), err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Groups = make([]*groups.GroupRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Groups[i] = result.Inventory[index].Clone()
		}
	}
	return result, guard()
}

// GetGroup preserves cloud get_group's filters-is-not-None branch. Only the
// default/null branch uses the closed domain-scoped GET-first resolver.
func (c *GroupReadCloud) GetGroup(ctx context.Context, nameOrID string, options ...GroupGetOption) (*GroupGetResult, error) {
	wrap := func(err error) error { return request.Wrap("GetGroup", "groups", cloudread.ContextError(ctx, err)) }
	owned, api, _, guard, err := c.prepareGroupSearch(ctx, nil)
	if err != nil {
		return nil, wrap(err)
	}
	policy := copyGroupGetOpts(GroupGetOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(groupSearchInvalid("nil Group get option"))
		}
		next := copyGroupGetOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return nil, wrap(err)
		}
		policy = copyGroupGetOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return nil, wrap(errors.Join(err, guard()))
		}
	}
	result := &GroupGetResult{Inventory: make([]*groups.GroupRecord, 0)}
	if policy.Filters == nil || bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		record, err := api.ResolveRecord(owned, nameOrID, groups.WithRecordResolveOpts(groups.RecordResolveOpts{DomainID: policy.DomainID, Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location}))
		if record != nil {
			result.Inventory = append(result.Inventory, record.Clone())
			result.Group = record.Clone()
		}
		if err != nil {
			return result, wrap(errors.Join(err, guard()))
		}
		if record != nil {
			result.Value, err = json.Marshal(record.Resource)
		}
		return result, wrap(errors.Join(err, guard()))
	}
	result.DeprecatedFilters = true
	selected, err := groupGetSearch(owned, api, nameOrID, policy, guard)
	if selected != nil {
		result.Inventory = selected.Inventory
		result.Expression = selected.Expression
	}
	if err != nil {
		return result, wrap(errors.Join(err, guard()))
	}
	truthy, err := cloudfilter.PythonJSONTruthy(selected.Value)
	if err == nil && truthy {
		result.Value, err = cloudfilter.First(selected.Value)
	}
	if err != nil {
		var duplicate *cloudfilter.MultipleError
		if errors.As(err, &duplicate) {
			err = errors.Join(resource.ErrAmbiguous, err)
		} else {
			err = errors.Join(groupSearchInvalid("get_group truth/len/index"), err)
		}
		return result, wrap(errors.Join(err, guard()))
	}
	if result.Value != nil && !selected.Expression && len(selected.Groups) == 1 {
		result.Group = selected.Groups[0].Clone()
	}
	return result, wrap(guard())
}
