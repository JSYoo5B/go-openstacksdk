package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RoleGetOpts owns filters presence, raw domain_id and concrete transport facts.
// Omitted/literal null filters use the fixed finder; other falsey filters use
// the deprecated search branch. Nil domain_id means forwarded Python None.
type RoleGetOpts struct {
	DomainID     json.RawMessage
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type RoleGetOption func(*RoleGetOpts) error

func copyRoleGetOpts(v RoleGetOpts) RoleGetOpts {
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
func WithRoleGetOpts(v RoleGetOpts) RoleGetOption {
	owned := copyRoleGetOpts(v)
	return func(c *RoleGetOpts) error { *c = copyRoleGetOpts(owned); return nil }
}
func WithRoleGetDomainID(v string) RoleGetOption {
	raw, _ := json.Marshal(v)
	return WithRoleGetDomainIDJSON(raw)
}
func WithRoleGetDomainIDJSON(v json.RawMessage) RoleGetOption {
	owned := bytes.Clone(v)
	return func(c *RoleGetOpts) error { c.DomainID = bytes.Clone(owned); return nil }
}
func WithRoleGetFilters(v json.RawMessage) RoleGetOption {
	owned := bytes.Clone(v)
	return func(c *RoleGetOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithRoleGetExpression(v string) RoleGetOption {
	raw, _ := json.Marshal(v)
	return WithRoleGetFilters(raw)
}
func WithRoleGetHeaders(v map[string]string) RoleGetOption {
	owned := maps.Clone(v)
	return func(c *RoleGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRoleGetMicroversion(v string) RoleGetOption {
	return func(c *RoleGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRoleGetLocation(v resource.CloudLocation) RoleGetOption {
	owned := v.Clone()
	return func(c *RoleGetOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// RoleGetResult exposes actual rows/evidence independently of returned Value.
// Inventory is the deprecated search observation; Lookup is only the actual
// final closed finder record. No expression result receives a fake receipt.
type RoleGetResult struct {
	Value             json.RawMessage
	Role              *roles.RoleRecord
	Inventory         []*roles.RoleRecord
	Lookup            *roles.RoleRecord
	SearchValue       json.RawMessage
	Expression        bool
	DeprecatedFilters bool
}

// GetRole preserves the Source's return of entities[0] for non-None filters;
// omitted/null filters instead call the fixed domain-aware GET-first finder.
func (c *RoleReadCloud) GetRole(ctx context.Context, nameOrID string, options ...RoleGetOption) (*RoleGetResult, error) {
	wrap := func(err error) error { return request.Wrap("GetRole", "roles", cloudread.ContextError(ctx, err)) }
	owned, api, _, guard, err := c.prepareRoleSearch(ctx, nil)
	if err != nil {
		return nil, wrap(err)
	}
	policy := copyRoleGetOpts(RoleGetOpts{})
	for _, apply := range slices.Clone(options) {
		if err = guard(); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(roleSearchInvalid("nil Role get option"))
		}
		next := copyRoleGetOpts(policy)
		if err = errors.Join(apply(&next), guard()); err != nil {
			return nil, wrap(err)
		}
		policy = copyRoleGetOpts(next)
	}
	if policy.Location != nil {
		if _, err = policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return nil, wrap(errors.Join(err, guard()))
		}
	}
	result := &RoleGetResult{Inventory: make([]*roles.RoleRecord, 0)}
	if policy.Filters != nil && !bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		result.DeprecatedFilters = true
		domain := bytes.Clone(policy.DomainID)
		if domain == nil {
			domain = json.RawMessage("null")
		}
		opts := []roles.RecordListOption{roles.WithRecordListQuery("domain_id", json.RawMessage(domain)), roles.WithRecordListHeaders(policy.Headers)}
		if policy.Microversion != nil {
			opts = append(opts, roles.WithRecordListMicroversion(*policy.Microversion))
		}
		if policy.Location != nil {
			opts = append(opts, roles.WithRecordListLocation(*policy.Location))
		}
		rows := make([]json.RawMessage, 0)
		for record, err := range api.ListRecords(owned, opts...) {
			if record != nil {
				result.Inventory = append(result.Inventory, record)
			}
			if err != nil {
				return result, wrap(errors.Join(err, guard()))
			}
			if err = guard(); err != nil {
				return result, wrap(err)
			}
			raw, cause := json.Marshal(record.Resource.Body)
			if cause != nil {
				return result, wrap(errors.Join(cause, guard()))
			}
			rows = append(rows, raw)
		}
		selected, err := cloudfilter.SelectLoaded(rows, nameOrID, policy.Filters, guard)
		if err != nil {
			return result, wrap(errors.Join(roleSearchInvalid("local Role get search"), err, guard()))
		}
		result.SearchValue = bytes.Clone(selected.Value)
		result.Expression = selected.Expression
		// Use loaded Python truth first; First supplies Source len and index rules.
		truthy, err := cloudfilter.PythonJSONTruthy(selected.Value)
		if err != nil {
			return result, wrap(errors.Join(err, guard()))
		}
		if !truthy {
			result.Value = json.RawMessage("null")
			return result, wrap(guard())
		}
		first, err := cloudfilter.First(selected.Value)
		if err != nil {
			var multiple *cloudfilter.MultipleError
			if errors.As(err, &multiple) {
				err = errors.Join(resource.ErrAmbiguous, err)
			} else {
				err = errors.Join(roleSearchInvalid("Source Role len/index"), err)
			}
			return result, wrap(errors.Join(err, guard()))
		}
		if first == nil {
			result.Value = json.RawMessage("null")
		} else {
			result.Value = bytes.Clone(first)
		}
		if !selected.Expression && len(selected.Indices) == 1 {
			result.Role = result.Inventory[selected.Indices[0]].Clone()
		}
		if err = guard(); err != nil {
			result.Value = nil
			result.Role = nil
			return result, wrap(err)
		}
		return result, nil
	}
	resolved, err := api.ResolveRecord(owned, nameOrID, roles.WithClosedLookupOpts(roles.ClosedLookupOpts{DomainID: policy.DomainID, Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location}))
	if err != nil {
		return result, wrap(errors.Join(err, guard()))
	}
	if resolved != nil {
		result.Lookup = resolved
		result.Role = resolved.Clone()
		result.Value, err = json.Marshal(result.Role.Resource.Body)
	} else {
		result.Value = json.RawMessage("null")
	}
	if err = errors.Join(err, guard()); err != nil {
		result.Value = nil
		result.Role = nil
		return result, wrap(err)
	}
	return result, nil
}
