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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ProjectReadCloud binds Keystone v3 project reads to an SDK-owned record API.
// Construction does not discover an endpoint or make a request.
type ProjectReadCloud struct{ api *projects.API }

func NewProjectReadCloud(api *projects.API) *ProjectReadCloud { return &ProjectReadCloud{api: api} }
func (c *ProjectReadCloud) RawAPI() *projects.API {
	if c == nil {
		return nil
	}
	return c.api
}

// ProjectReadOpts owns list_projects/search_projects arguments. Falsey filters
// become {}; truthy filters must be an object, exactly as dict(**filters) does.
// NameOrID is the list's server name filter; SearchProjects has its own argument.
type ProjectReadOpts struct {
	DomainID     string
	NameOrID     string
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type ProjectReadOption func(*ProjectReadOpts) error

func projectReadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func copyProjectReadOpts(v ProjectReadOpts) ProjectReadOpts {
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
func WithProjectReadOpts(v ProjectReadOpts) ProjectReadOption {
	owned := copyProjectReadOpts(v)
	return func(c *ProjectReadOpts) error { *c = copyProjectReadOpts(owned); return nil }
}
func WithProjectReadDomainID(v string) ProjectReadOption {
	return func(c *ProjectReadOpts) error { c.DomainID = v; return nil }
}
func WithProjectReadNameOrID(v string) ProjectReadOption {
	return func(c *ProjectReadOpts) error { c.NameOrID = v; return nil }
}
func WithProjectReadFilters(v json.RawMessage) ProjectReadOption {
	owned := bytes.Clone(v)
	return func(c *ProjectReadOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	}
}
func WithProjectReadHeaders(v map[string]string) ProjectReadOption {
	owned := maps.Clone(v)
	return func(c *ProjectReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithProjectReadMicroversion(v string) ProjectReadOption {
	return func(c *ProjectReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithProjectReadLocation(v resource.CloudLocation) ProjectReadOption {
	owned := v.Clone()
	return func(c *ProjectReadOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// ProjectListResult preserves list()'s polymorphic Proxy result. Value is always
// []any on success, including possible opaque generators inside that list. JSON
// is nil for an opaque result. Projects is nonnil only for actual record rows;
// Inventory retains every physically returned record, also on later failure.
type ProjectListResult struct {
	Value     []any
	JSON      json.RawMessage
	Projects  []*projects.ProjectRecord
	Inventory []*projects.ProjectRecord
}

// ProjectSearchResult retains local selection independently of actual rows.
type ProjectSearchResult struct {
	Value     json.RawMessage
	Projects  []*projects.ProjectRecord
	Inventory []*projects.ProjectRecord
}

func (c *ProjectReadCloud) prepare(ctx context.Context, options []ProjectReadOption) (context.Context, *projects.API, ProjectReadOpts, func() error, error) {
	policy := ProjectReadOpts{}
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, policy, nil, err
	}
	if c == nil || c.api == nil {
		return ctx, nil, policy, nil, projectReadInvalid("Keystone v3 project record API is required")
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
			observed = projectReadInvalid("selected project API changed")
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
			return ctx, nil, policy, guard, projectReadInvalid("nil project cloud read option")
		}
		next := copyProjectReadOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return ctx, nil, policy, guard, err
		}
		policy = copyProjectReadOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, nil); err != nil {
			return ctx, nil, policy, guard, err
		}
	}
	// The owned source retains selected headers before caller callbacks. The
	// compound operation guard still watches the original API/client throughout.
	return ctx, projects.New(&source.Client), policy, guard, nil
}
func projectReadListOptions(policy ProjectReadOpts, includeName bool) ([]projects.RecordListOption, error) {
	options := make([]projects.RecordListOption, 0)
	if policy.Filters != nil {
		raw := bytes.TrimSpace(*policy.Filters)
		truthy, err := cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return nil, errors.Join(projectReadInvalid("project filters must be complete UTF-8 JSON"), err)
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return nil, errors.Join(projectReadInvalid("truthy project filters must be an object for dict(**filters)"), err)
			}
			for _, member := range members {
				options = append(options, projects.WithRecordListQuery(member.Key, member.Value))
			}
		}
	}
	if includeName && policy.NameOrID != "" {
		options = append(options, projects.WithRecordListQuery("name", policy.NameOrID))
	}
	if policy.DomainID != "" {
		options = append(options, projects.WithRecordListQuery("domain_id", policy.DomainID))
	}
	if policy.Headers != nil {
		options = append(options, projects.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		options = append(options, projects.WithRecordListMicroversion(*policy.Microversion))
	}
	return options, nil
}
func (c *ProjectReadCloud) listPrepared(ctx context.Context, api *projects.API, policy ProjectReadOpts, includeName bool, guard func() error) (*ProjectListResult, error) {
	options, err := projectReadListOptions(policy, includeName)
	if err != nil {
		return nil, err
	}
	proxy, err := api.ProxyListRecords(ctx, options...)
	if err != nil {
		return nil, err
	}
	result := &ProjectListResult{Value: make([]any, 0), Inventory: make([]*projects.ProjectRecord, 0)}
	switch v := proxy.Value.(type) {
	case *projects.RecordListGenerator:
		result.Projects = make([]*projects.ProjectRecord, 0)
		for record, err := range v.Records {
			if err != nil {
				return result, err
			}
			if policy.Location != nil {
				raw, err := policy.Location.ForResource(record.Resource.Body["project_id"], record.Resource.Body["availability_zone"])
				if err != nil {
					return result, err
				}
				record.Resource.Body["location"] = raw
			}
			result.Inventory = append(result.Inventory, record)
			result.Projects = append(result.Projects, record)
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
		return result, projectReadInvalid("list_projects cannot iterate Proxy result %T", proxy.Value)
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

// ListProjects mirrors cloud.list_projects: ensure the v3 binding, expand
// filters, override truthy name/domain arguments, then eagerly consume list().
func (c *ProjectReadCloud) ListProjects(ctx context.Context, options ...ProjectReadOption) (*ProjectListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListProjects", "projects", cloudread.ContextError(ctx, err))
	}
	owned, api, policy, guard, err := c.prepare(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result, err := c.listPrepared(owned, api, policy, true, guard)
	return result, wrap(errors.Join(err, guard()))
}

// SearchProjects calls list_projects without a name hint, then matches exact
// ID/name or a glob and ordered nested filters on the projected Resource views.
// A truthy string filter fails in the preceding list_projects kwargs expansion,
// as it does in the pinned Source despite that method's JMESPath documentation.
func (c *ProjectReadCloud) SearchProjects(ctx context.Context, nameOrID string, options ...ProjectReadOption) (*ProjectSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchProjects", "projects", cloudread.ContextError(ctx, err))
	}
	owned, api, policy, guard, err := c.prepare(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	listed, err := c.listPrepared(owned, api, policy, false, guard)
	result := &ProjectSearchResult{Inventory: make([]*projects.ProjectRecord, 0)}
	if listed != nil {
		result.Inventory = listed.Inventory
	}
	if err != nil {
		return result, wrap(errors.Join(err, guard()))
	}
	// _filter_list returns an unfiltered list verbatim for falsey filters and
	// absent name. Preserve scalar/opaque literal rows in that exact branch.
	filters := policy.Filters
	truthy := false
	if filters != nil {
		truthy, err = cloudfilter.PythonJSONTruthy(*filters)
		if err != nil {
			return result, wrap(err)
		}
	}
	if nameOrID == "" && !truthy {
		if listed.JSON == nil {
			return result, wrap(projectReadInvalid("opaque project list cannot be represented as search JSON"))
		}
		result.Value = bytes.Clone(listed.JSON)
		result.Projects = listed.Projects
		return result, wrap(guard())
	}
	rows := make([]json.RawMessage, 0, len(listed.Value))
	for _, value := range listed.Value {
		if err := guard(); err != nil {
			return result, wrap(err)
		}
		raw, cause := json.Marshal(value)
		if cause != nil || !utf8.Valid(raw) {
			return result, wrap(errors.Join(projectReadInvalid("project search row is an opaque Source value"), cause))
		}
		rows = append(rows, raw)
	}
	selected, err := cloudfilter.SelectLoaded(rows, nameOrID, filters, guard)
	if err != nil {
		return result, wrap(errors.Join(projectReadInvalid("local project search"), err))
	}
	result.Value = bytes.Clone(selected.Value)
	if listed.Projects != nil && !selected.Expression {
		result.Projects = make([]*projects.ProjectRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Projects[i] = listed.Projects[index]
		}
	}
	return result, wrap(guard())
}
