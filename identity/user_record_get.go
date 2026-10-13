package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UserGetOpts owns filters presence, raw domain_id and concrete transport facts.
// Omitted/literal null filters use the fixed finder; other falsey filters use
// the deprecated search branch. Nil domain_id means forwarded Python None.
type UserGetOpts struct {
	DomainID     json.RawMessage
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type UserGetOption func(*UserGetOpts) error

func copyUserGetOpts(v UserGetOpts) UserGetOpts {
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
func WithUserGetOpts(v UserGetOpts) UserGetOption {
	owned := copyUserGetOpts(v)
	return func(c *UserGetOpts) error { *c = copyUserGetOpts(owned); return nil }
}
func WithUserGetDomainID(v string) UserGetOption {
	raw, _ := json.Marshal(v)
	return WithUserGetDomainIDJSON(raw)
}
func WithUserGetDomainIDJSON(v json.RawMessage) UserGetOption {
	owned := bytes.Clone(v)
	return func(c *UserGetOpts) error { c.DomainID = bytes.Clone(owned); return nil }
}
func WithUserGetFilters(v json.RawMessage) UserGetOption {
	owned := bytes.Clone(v)
	return func(c *UserGetOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithUserGetExpression(v string) UserGetOption {
	raw, _ := json.Marshal(v)
	return WithUserGetFilters(raw)
}
func WithUserGetHeaders(v map[string]string) UserGetOption {
	owned := maps.Clone(v)
	return func(c *UserGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithUserGetMicroversion(v string) UserGetOption {
	return func(c *UserGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithUserGetLocation(v resource.CloudLocation) UserGetOption {
	owned := v.Clone()
	return func(c *UserGetOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// UserGetResult exposes actual rows/evidence independently of returned Value.
// Inventory is the deprecated search observation; Lookup is only the actual
// final closed finder record. No expression result receives a fake receipt.
type UserGetResult struct {
	Value             json.RawMessage
	User              *users.UserRecord
	Inventory         []*users.UserRecord
	Lookup            *users.UserRecord
	SearchValue       json.RawMessage
	Expression        bool
	DeprecatedFilters bool
}

// GetUser preserves the pinned Source bug: successful deprecated search only
// checks truthiness and len, then still calls find_user with original domain.
// It never returns search entities[0] or uses that row as the final identity.
func (c *UserReadCloud) GetUser(ctx context.Context, nameOrID string, options ...UserGetOption) (*UserGetResult, error) {
	wrap := func(err error) error { return request.Wrap("GetUser", "users", cloudread.ContextError(ctx, err)) }
	p, err := c.captureUserRead(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyUserRead(p, UserGetOpts{}, copyUserGetOpts, options)
	if err != nil {
		return nil, wrap(err)
	}
	if policy.Location != nil {
		if _, err = policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return nil, wrap(errors.Join(err, p.guard()))
		}
	}
	result := &UserGetResult{Inventory: make([]*users.UserRecord, 0)}
	if policy.Filters != nil && !bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		result.DeprecatedFilters = true
		list := UserListOpts{Query: make(map[string]json.RawMessage), Headers: maps.Clone(policy.Headers), Microversion: policy.Microversion, Location: policy.Location}
		if nameOrID != "" && !sourceUserUUID(nameOrID) {
			list.Query["name"], _ = json.Marshal(nameOrID)
			list.order = append(list.order, "name")
		}
		truthy, err := cloudfilter.PythonJSONTruthy(policy.DomainID)
		if err != nil {
			return result, wrap(errors.Join(userReadInvalid("domain truthiness"), err, p.guard()))
		}
		if truthy {
			list.Query["domain_id"] = bytes.Clone(policy.DomainID)
			list.order = append(list.order, "domain_id")
		}
		listed, err := p.listUserRecords(list)
		if listed != nil {
			result.Inventory = listed.Inventory
		}
		if err != nil {
			return result, wrap(errors.Join(err, p.guard()))
		}
		rows := make([]json.RawMessage, 0, len(listed.Users))
		for _, record := range listed.Users {
			raw, cause := json.Marshal(record.Resource.Body)
			if cause != nil {
				return result, wrap(errors.Join(cause, p.guard()))
			}
			rows = append(rows, raw)
		}
		selected, err := cloudfilter.SelectLoaded(rows, nameOrID, policy.Filters, p.guard)
		if err != nil {
			return result, wrap(errors.Join(userReadInvalid("local User get search"), err, p.guard()))
		}
		result.SearchValue = bytes.Clone(selected.Value)
		result.Expression = selected.Expression
		proceed, err := userGetSearchAllowsFind(selected.Value)
		if err != nil {
			return result, wrap(errors.Join(err, p.guard()))
		}
		if !proceed {
			result.Value = json.RawMessage("null")
			return result, wrap(p.guard())
		}
	}
	if err := p.guard(); err != nil {
		return result, wrap(err)
	}
	resolved, err := p.api.ResolveRecord(p.ctx, nameOrID, users.WithClosedLookupOpts(users.ClosedLookupOpts{DomainID: policy.DomainID, Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location}))
	if err != nil {
		return result, wrap(errors.Join(err, p.guard()))
	}
	if resolved != nil {
		result.Lookup = resolved
		result.User = resolved.Clone()
		result.Value, err = json.Marshal(result.User.Resource.Body)
	} else {
		result.Value = json.RawMessage("null")
	}
	if err = errors.Join(err, p.guard()); err != nil {
		result.Value = nil
		result.User = nil
		return result, wrap(err)
	}
	return result, nil
}
func userGetSearchAllowsFind(raw json.RawMessage) (bool, error) {
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return false, errors.Join(userReadInvalid("Source User search truthiness"), err)
	}
	if !truthy {
		return false, nil
	}
	value := bytes.TrimSpace(raw)
	amount := 0
	switch value[0] {
	case '[':
		var rows []json.RawMessage
		if err = json.Unmarshal(value, &rows); err != nil {
			return false, err
		}
		amount = len(rows)
	case '{':
		rows, cause := cloudfilter.ObjectMembers(value)
		if cause != nil {
			return false, cause
		}
		amount = len(rows)
	case '"':
		var text string
		if err = json.Unmarshal(value, &text); err != nil {
			return false, err
		}
		amount = utf8.RuneCountInString(text)
	default:
		return false, userReadInvalid("Source get_user search result has no len")
	}
	if amount > 1 {
		return false, errors.Join(resource.ErrAmbiguous, &cloudfilter.MultipleError{Length: amount})
	}
	return true, nil
}
