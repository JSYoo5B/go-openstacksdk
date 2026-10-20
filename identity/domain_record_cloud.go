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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DomainReadCloud binds Keystone v3 domain reads to an SDK-owned record API.
// Construction does not discover an endpoint or make a request.
type DomainReadCloud struct{ api *domains.API }

func NewDomainReadCloud(api *domains.API) *DomainReadCloud { return &DomainReadCloud{api: api} }
func (c *DomainReadCloud) RawAPI() *domains.API {
	if c == nil {
		return nil
	}
	return c.api
}

// DomainReadOpts owns keyword filters and distinguishes absent name_or_id
// from an explicitly empty identifier for search_domains. List keyword filters
// require an object; identifier-present search accepts local expression JSON.
type DomainReadOpts struct {
	NameOrID     *string
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type DomainReadOption func(*DomainReadOpts) error

func domainReadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func copyDomainReadOpts(v DomainReadOpts) DomainReadOpts {
	if v.NameOrID != nil {
		x := *v.NameOrID
		v.NameOrID = &x
	}
	v.Headers = maps.Clone(v.Headers)
	if v.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &raw
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
func WithDomainReadOpts(v DomainReadOpts) DomainReadOption {
	owned := copyDomainReadOpts(v)
	return func(c *DomainReadOpts) error { *c = copyDomainReadOpts(owned); return nil }
}
func WithDomainReadNameOrID(v string) DomainReadOption {
	return func(c *DomainReadOpts) error { x := v; c.NameOrID = &x; return nil }
}
func WithDomainReadFilters(v json.RawMessage) DomainReadOption {
	owned := bytes.Clone(v)
	return func(c *DomainReadOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	}
}
func WithDomainReadHeaders(v map[string]string) DomainReadOption {
	owned := maps.Clone(v)
	return func(c *DomainReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithDomainReadMicroversion(v string) DomainReadOption {
	return func(c *DomainReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithDomainReadLocation(v resource.CloudLocation) DomainReadOption {
	owned := v.Clone()
	return func(c *DomainReadOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// DomainListResult preserves list()'s polymorphic Proxy result. Value is always
// []any on success, including possible opaque generators inside that list. JSON
// is nil for an opaque result. Domains is nonnil only for actual record rows;
// Inventory retains every physically returned record, also on later failure.
type DomainListResult struct {
	Value     []any
	JSON      json.RawMessage
	Domains   []*domains.DomainRecord
	Inventory []*domains.DomainRecord
}

// DomainSearchResult retains local selection independently of actual rows.
type DomainSearchResult struct {
	Expression   bool
	ValueObjects []any
	Value        json.RawMessage
	Domains      []*domains.DomainRecord
	Inventory    []*domains.DomainRecord
}

func (c *DomainReadCloud) prepare(ctx context.Context, options []DomainReadOption) (context.Context, *domains.API, DomainReadOpts, func() error, error) {
	policy := DomainReadOpts{}
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, policy, nil, err
	}
	if c == nil || c.api == nil {
		return ctx, nil, policy, nil, domainReadInvalid("Keystone v3 domain record API is required")
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
			observed = domainReadInvalid("selected domain API changed")
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
			return ctx, nil, policy, guard, domainReadInvalid("nil domain cloud read option")
		}
		next := copyDomainReadOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return ctx, nil, policy, guard, err
		}
		policy = copyDomainReadOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, nil); err != nil {
			return ctx, nil, policy, guard, err
		}
	}
	// The owned source retains selected headers before caller callbacks. The
	// compound operation guard still watches the original API/client throughout.
	return ctx, domains.New(&source.Client), policy, guard, nil
}
func domainReadListOptions(policy DomainReadOpts) ([]domains.RecordListOption, error) {
	options := make([]domains.RecordListOption, 0)
	if policy.Filters != nil {
		members, err := cloudfilter.ObjectMembers(*policy.Filters)
		if err != nil {
			return nil, errors.Join(domainReadInvalid("list_domains keyword filters must be a JSON object"), err)
		}
		for _, member := range members {
			options = append(options, domains.WithRecordListQuery(member.Key, member.Value))
		}
	}
	if policy.Headers != nil {
		options = append(options, domains.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		options = append(options, domains.WithRecordListMicroversion(*policy.Microversion))
	}
	if policy.Location != nil {
		options = append(options, domains.WithRecordListLocation(*policy.Location))
	}
	return options, nil
}
func (c *DomainReadCloud) listPrepared(ctx context.Context, api *domains.API, policy DomainReadOpts, guard func() error) (*DomainListResult, error) {
	options, err := domainReadListOptions(policy)
	if err != nil {
		return nil, err
	}
	proxy, err := api.ProxyListRecords(ctx, options...)
	if err != nil {
		return nil, err
	}
	result := &DomainListResult{Value: make([]any, 0), Inventory: make([]*domains.DomainRecord, 0)}
	switch v := proxy.Value.(type) {
	case *domains.RecordListGenerator:
		result.Domains = make([]*domains.DomainRecord, 0)
		for record, err := range v.Records {
			if err != nil {
				return result, err
			}
			result.Inventory = append(result.Inventory, record)
			result.Domains = append(result.Domains, record)
			result.Value = append(result.Value, record.Resource)
		}
	case []any:
		result.Value = slices.Clone(v)
	case map[string]any:
		// Go map iteration has no stable order; make the interoperability policy
		// explicit and deterministic for an object-valued Proxy expression.
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			result.Value = append(result.Value, key)
		}
	case string:
		for _, r := range v {
			result.Value = append(result.Value, string(r))
		}
	default:
		return result, domainReadInvalid("list_domains cannot iterate Proxy result %T", proxy.Value)
	}
	if err := guard(); err != nil {
		return result, err
	}
	// json.Marshal invokes the Source generator's explicit failure for an opaque
	// value. Keep Value rather than discarding or eagerly iterating that generator.
	if raw, err := json.Marshal(result.Value); err == nil {
		result.JSON = bytes.Clone(raw)
	}
	return result, nil
}

// ListDomains mirrors cloud.list_domains: ensure an already selected v3
// binding, expand arbitrary keyword filters, then eagerly consume Proxy list().
func (c *DomainReadCloud) ListDomains(ctx context.Context, options ...DomainReadOption) (*DomainListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListDomains", "domains", cloudread.ContextError(ctx, err))
	}
	owned, api, policy, guard, err := c.prepare(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result, err := c.listPrepared(owned, api, policy, guard)
	return result, wrap(errors.Join(err, guard()))
}

// SearchDomains preserves Source's name_or_id presence branch. With no
// identifier it expands filters into list_domains kwargs. With an explicit
// identifier (including empty) it materializes an unfiltered list first, then
// evaluates exact/glob identifiers plus local mapping or JMESPath filters.
func (c *DomainReadCloud) SearchDomains(ctx context.Context, options ...DomainReadOption) (*DomainSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchDomains", "domains", cloudread.ContextError(ctx, err))
	}
	owned, api, policy, guard, err := c.prepare(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result := &DomainSearchResult{Inventory: make([]*domains.DomainRecord, 0)}
	originalFilters := policy.Filters
	// Source replaces exactly None with {}, not every falsey value.
	if policy.Filters != nil && bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		policy.Filters = nil
		originalFilters = nil
	}
	if policy.NameOrID != nil {
		policy.Filters = nil
	}
	listed, err := c.listPrepared(owned, api, policy, guard)
	if listed != nil {
		result.Inventory = listed.Inventory
	}
	if err != nil {
		return result, wrap(errors.Join(err, guard()))
	}
	if policy.NameOrID == nil {
		result.Value = bytes.Clone(listed.JSON)
		result.ValueObjects = slices.Clone(listed.Value)
		if listed.Domains != nil {
			result.Domains = make([]*domains.DomainRecord, len(listed.Domains))
			for i, r := range listed.Domains {
				result.Domains[i] = r.Clone()
			}
		}
		return result, wrap(guard())
	}
	rows := make([]json.RawMessage, 0, len(listed.Value))
	for _, v := range listed.Value {
		if err := guard(); err != nil {
			return result, wrap(err)
		}
		raw, cause := json.Marshal(v)
		if cause != nil || !utf8.Valid(raw) {
			return result, wrap(errors.Join(domainReadInvalid("opaque domain search row"), cause))
		}
		rows = append(rows, raw)
	}
	selected, err := cloudfilter.SelectLoaded(rows, *policy.NameOrID, originalFilters, guard)
	if err != nil {
		return result, wrap(errors.Join(domainReadInvalid("local domain search"), err))
	}
	if err := guard(); err != nil {
		return result, wrap(err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression && listed.Domains != nil {
		result.Domains = make([]*domains.DomainRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Domains[i] = listed.Domains[index].Clone()
		}
	}
	return result, wrap(guard())
}
