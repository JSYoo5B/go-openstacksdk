package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UserReadCloud binds finite Source list_users/search_users to selected v3.
// SDK users implement no class/session builders or response decoders.
type UserReadCloud struct{ api *users.API }

func NewUserReadCloud(api *users.API) *UserReadCloud { return &UserReadCloud{api: api} }
func (c *UserReadCloud) RawAPI() *users.API {
	if c == nil {
		return nil
	}
	return c.api
}
func userReadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// UserListOpts owns the two actual Source valid_kwargs keys, domain_id/name.
// Transport and explicit location fields are Go conveniences, not extra Python
// cloud list kwargs. Query retains omission/null/false/list JSON values.
type UserListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
	order        []string
}
type UserListOption func(*UserListOpts) error

func copyUserListOpts(v UserListOpts) UserListOpts {
	query := make(map[string]json.RawMessage, len(v.Query))
	for key, raw := range v.Query {
		query[key] = bytes.Clone(raw)
	}
	v.Query = query
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	v.order = slices.Clone(v.order)
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
func WithUserListOpts(v UserListOpts) UserListOption {
	owned := copyUserListOpts(v)
	return func(c *UserListOpts) error { *c = copyUserListOpts(owned); return nil }
}
func WithUserListQuery(key string, value any) UserListOption {
	if key != "name" && key != "domain_id" {
		return func(*UserListOpts) error { return userReadInvalid("%q is not a Source list_users valid kwarg", key) }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *UserListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithUserListDomainID(v string) UserListOption { return WithUserListQuery("domain_id", v) }
func WithUserListName(v string) UserListOption     { return WithUserListQuery("name", v) }
func WithUserListHeaders(v map[string]string) UserListOption {
	owned := maps.Clone(v)
	return func(c *UserListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithUserListMicroversion(v string) UserListOption {
	return func(c *UserListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithUserListLocation(v resource.CloudLocation) UserListOption {
	owned := v.Clone()
	return func(c *UserListOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// UserListResult associates Source declared JSON with only actually read rows.
// Inventory remains available on a late failure; successful Values/Users are
// published only after all pages complete. A local transform has no receipt.
type UserListResult struct {
	Value     json.RawMessage
	Users     []*users.UserRecord
	Inventory []*users.UserRecord
}

type userReadPrepared struct {
	ctx   context.Context
	api   *users.API
	guard func() error
}

func (c *UserReadCloud) captureUserRead(ctx context.Context) (*userReadPrepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.api == nil {
		return nil, userReadInvalid("selected Keystone v3 User API required")
	}
	selected := c.api
	client := selected.RawClient()
	source, err := cloudread.Capture(ctx, client, "identity")
	if err != nil {
		return nil, err
	}
	previous := rest.OperationGuard(ctx)
	var observed error
	guard := func(current context.Context) error {
		if observed == nil && (c.api != selected || selected.RawClient() != client) {
			observed = userReadInvalid("selected User API changed")
		}
		var outer error
		if previous != nil {
			outer = previous(current)
		}
		return errors.Join(cloudread.ContextError(current, observed), source.Guard(current), outer)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	return &userReadPrepared{ctx: ctx, api: users.New(&source.Client), guard: func() error { return guard(ctx) }}, guard(ctx)
}
func applyUserRead[T any, O ~func(*T) error](p *userReadPrepared, zero T, clone func(T) T, options []O) (T, error) {
	value := clone(zero)
	for _, apply := range slices.Clone(options) {
		if err := p.guard(); err != nil {
			return value, err
		}
		if apply == nil {
			return value, userReadInvalid("nil User read option")
		}
		next := clone(value)
		if err := errors.Join(apply(&next), p.guard()); err != nil {
			return value, err
		}
		value = clone(next)
	}
	return value, p.guard()
}
func userListOptions(v UserListOpts) ([]users.RecordListOption, error) {
	order := slices.Clone(v.order)
	remainder := make([]string, 0)
	for key := range v.Query {
		if key != "name" && key != "domain_id" {
			return nil, userReadInvalid("%q is not a Source list_users valid kwarg", key)
		}
		if !slices.Contains(order, key) {
			remainder = append(remainder, key)
		}
	}
	slices.Sort(remainder)
	order = append(order, remainder...)
	opts := make([]users.RecordListOption, 0, len(order)+3)
	for _, key := range order {
		if raw, exists := v.Query[key]; exists {
			opts = append(opts, users.WithRecordListQuery(key, json.RawMessage(bytes.Clone(raw))))
		}
	}
	if v.Headers != nil {
		opts = append(opts, users.WithRecordListHeaders(v.Headers))
	}
	if v.Microversion != nil {
		opts = append(opts, users.WithRecordListMicroversion(*v.Microversion))
	}
	if v.Location != nil {
		opts = append(opts, users.WithRecordListLocation(*v.Location))
	}
	return opts, nil
}
func (p *userReadPrepared) listUserRecords(v UserListOpts) (*UserListResult, error) {
	opts, err := userListOptions(v)
	if err != nil {
		return nil, err
	}
	result := &UserListResult{Inventory: make([]*users.UserRecord, 0)}
	records := make([]*users.UserRecord, 0)
	rows := make([]json.RawMessage, 0)
	for row, err := range p.api.ListRecords(p.ctx, opts...) {
		if row != nil {
			result.Inventory = append(result.Inventory, row)
		}
		if err != nil {
			return result, errors.Join(err, p.guard())
		}
		if err := p.guard(); err != nil {
			return result, err
		}
		raw, err := json.Marshal(row.Resource.Body)
		if err != nil {
			return result, errors.Join(err, p.guard())
		}
		rows = append(rows, bytes.Clone(raw))
		records = append(records, row.Clone())
	}
	if err := p.guard(); err != nil {
		return result, err
	}
	result.Value, err = json.Marshal(rows)
	if err != nil {
		return result, err
	}
	result.Users = records
	return result, p.guard()
}

// ListUsers mirrors Source's decorated cloud helper: only name/domain_id
// Source kwargs are accepted, then the selected v3 Resource list is consumed.
func (c *UserReadCloud) ListUsers(ctx context.Context, options ...UserListOption) (*UserListResult, error) {
	wrap := func(err error) error { return request.Wrap("ListUsers", "users", cloudread.ContextError(ctx, err)) }
	p, err := c.captureUserRead(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	opts, err := applyUserRead(p, UserListOpts{}, copyUserListOpts, options)
	if err != nil {
		return nil, wrap(err)
	}
	result, err := p.listUserRecords(opts)
	return result, wrap(errors.Join(err, p.guard()))
}

// UserSearchOpts separates only the domain server scope from local filters.
// NameOrID is a concrete string: non-UUID names also become a server name hint.
type UserSearchOpts struct {
	DomainID     string
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type UserSearchOption func(*UserSearchOpts) error

func copyUserSearchOpts(v UserSearchOpts) UserSearchOpts {
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
func WithUserSearchOpts(v UserSearchOpts) UserSearchOption {
	owned := copyUserSearchOpts(v)
	return func(c *UserSearchOpts) error { *c = copyUserSearchOpts(owned); return nil }
}
func WithUserSearchDomainID(v string) UserSearchOption {
	return func(c *UserSearchOpts) error { c.DomainID = v; return nil }
}
func WithUserSearchFilters(v json.RawMessage) UserSearchOption {
	owned := bytes.Clone(v)
	return func(c *UserSearchOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithUserSearchHeaders(v map[string]string) UserSearchOption {
	owned := maps.Clone(v)
	return func(c *UserSearchOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithUserSearchMicroversion(v string) UserSearchOption {
	return func(c *UserSearchOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithUserSearchLocation(v resource.CloudLocation) UserSearchOption {
	owned := v.Clone()
	return func(c *UserSearchOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// UserSearchResult keeps arbitrary expression JSON separate from actual rows.
type UserSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Users      []*users.UserRecord
	Inventory  []*users.UserRecord
}

// sourceUserUUID follows Source _is_uuid_like's UUID hex normalization, not
// an RFC-version restriction: upper hex, braces, urn/uuid prefixes and hyphens
// anywhere in a 32-hex spelling are accepted. Non-ASCII hex remains a name.
func sourceUserUUID(value string) bool {
	normalized := strings.ReplaceAll(strings.ReplaceAll(value, "urn:", ""), "uuid:", "")
	normalized = strings.ReplaceAll(strings.Trim(normalized, "{}"), "-", "")
	if len(normalized) != 32 {
		return false
	}
	for _, c := range normalized {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// SearchUsers preserves the non-UUID server-name hint, materializes every page,
// then performs exact/glob and ordered local mapping/JMESPath filtering. Filter
// failures do not occur until all list responses have been read successfully.
func (c *UserReadCloud) SearchUsers(ctx context.Context, nameOrID string, options ...UserSearchOption) (*UserSearchResult, error) {
	wrap := func(err error) error { return request.Wrap("SearchUsers", "users", cloudread.ContextError(ctx, err)) }
	p, err := c.captureUserRead(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyUserRead(p, UserSearchOpts{}, copyUserSearchOpts, options)
	if err != nil {
		return nil, wrap(err)
	}
	list := UserListOpts{Query: make(map[string]json.RawMessage), Headers: maps.Clone(policy.Headers), Microversion: policy.Microversion, Location: policy.Location}
	if nameOrID != "" && !sourceUserUUID(nameOrID) {
		list.Query["name"], _ = json.Marshal(nameOrID)
		list.order = append(list.order, "name")
	}
	if policy.DomainID != "" {
		list.Query["domain_id"], _ = json.Marshal(policy.DomainID)
		list.order = append(list.order, "domain_id")
	}
	listed, err := p.listUserRecords(list)
	result := &UserSearchResult{Inventory: make([]*users.UserRecord, 0)}
	if listed != nil {
		result.Inventory = listed.Inventory
	}
	if err != nil {
		return result, wrap(err)
	}
	rows := make([]json.RawMessage, 0, len(listed.Users))
	for _, row := range listed.Users {
		raw, cause := json.Marshal(row.Resource.Body)
		if cause != nil {
			return result, wrap(cause)
		}
		rows = append(rows, bytes.Clone(raw))
	}
	selected, err := cloudfilter.SelectLoaded(rows, nameOrID, policy.Filters, p.guard)
	if err != nil {
		return result, wrap(errors.Join(userReadInvalid("local User search"), err))
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Users = make([]*users.UserRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Users[i] = listed.Users[index].Clone()
		}
	}
	return result, wrap(p.guard())
}
