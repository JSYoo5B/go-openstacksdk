package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServerGroupCloudReadOpts owns local filters and SDK transport/location
// conveniences. Filters nil/null selects GetServerGroup's default find;
// any nonnull value, even false/0/empty, selects its deprecated search branch.
type ServerGroupCloudReadOpts struct {
	Filters         *json.RawMessage
	Headers         map[string]string
	CurrentLocation *resource.CloudLocation
}
type ServerGroupCloudReadOption func(*ServerGroupCloudReadOpts) error

func cloneServerGroupCloudOptions(v ServerGroupCloudReadOpts) ServerGroupCloudReadOpts {
	if v.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &raw
	}
	v.Headers = maps.Clone(v.Headers)
	if v.CurrentLocation != nil {
		loc := v.CurrentLocation.Clone()
		v.CurrentLocation = &loc
	}
	return v
}
func WithServerGroupCloudReadOpts(v ServerGroupCloudReadOpts) ServerGroupCloudReadOption {
	owned := cloneServerGroupCloudOptions(v)
	return func(c *ServerGroupCloudReadOpts) error { *c = cloneServerGroupCloudOptions(owned); return nil }
}
func WithServerGroupCloudFilters(v json.RawMessage) ServerGroupCloudReadOption {
	owned := bytes.Clone(v)
	return func(c *ServerGroupCloudReadOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	}
}
func WithServerGroupCloudExpression(v string) ServerGroupCloudReadOption {
	raw, _ := json.Marshal(v)
	return WithServerGroupCloudFilters(raw)
}
func WithServerGroupCloudHeaders(v map[string]string) ServerGroupCloudReadOption {
	owned := maps.Clone(v)
	return func(c *ServerGroupCloudReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithServerGroupCloudCurrentLocation(v resource.CloudLocation) ServerGroupCloudReadOption {
	owned := v.Clone()
	return func(c *ServerGroupCloudReadOpts) error { loc := owned.Clone(); c.CurrentLocation = &loc; return nil }
}

// Value is arbitrary Source JSON. Groups associate ordinary local matches with
// actual owned records; JMESPath results never acquire fabricated HTTP proof.
type ServerGroupCloudSearchResult struct {
	Value             json.RawMessage
	Groups, Inventory []*servergroups.ServerGroupRecord
	Expression        bool
}
type ServerGroupCloudGetResult struct {
	Value           json.RawMessage
	Group, Observed *servergroups.ServerGroupRecord
	Inventory       []*servergroups.ServerGroupRecord
	Filtered        bool
}
type serverGroupCloudWorkflow struct {
	ctx   context.Context
	api   *servergroups.API
	cfg   ServerGroupCloudReadOpts
	check func(context.Context) error
}

func serverGroupCloudWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "servergroups", cloudread.ContextError(ctx, err))
}
func (s *Service) serverGroupCloudStart(ctx context.Context, options []ServerGroupCloudReadOption) (*serverGroupCloudWorkflow, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.API.ServerGroups == nil || s.client == nil || s.API.RawClient() != s.client || s.API.ServerGroups.RawClient() != s.client {
		return nil, invalid("configured compute/servergroups service required")
	}
	selectedAPI, selectedGroup, selectedClient := s.API, s.API.ServerGroups, s.client
	source, err := cloudread.Capture(ctx, selectedClient, "compute")
	if err != nil {
		return nil, err
	}
	var observed error
	guard := func(ctx context.Context) error {
		if observed == nil && (s.API != selectedAPI || selectedAPI.ServerGroups != selectedGroup || s.client != selectedClient || selectedAPI.RawClient() != selectedClient || selectedGroup.RawClient() != selectedClient) {
			observed = invalid("selected compute/servergroups binding changed")
		}
		return errors.Join(observed, source.Guard(ctx))
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := cloneServerGroupCloudOptions(ServerGroupCloudReadOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, invalid("nil server group cloud option")
		}
		next := cloneServerGroupCloudOptions(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, err
		}
		cfg = cloneServerGroupCloudOptions(next)
	}
	if cfg.Filters != nil && (!utf8.Valid(*cfg.Filters) || !json.Valid(*cfg.Filters)) {
		return nil, invalid("server group filters require complete UTF-8 JSON")
	}
	if err := source.WithPolicy(ctx, nil, cfg.Headers); err != nil {
		return nil, err
	}
	if cfg.CurrentLocation != nil {
		if _, err := serverGroupCloudLocation(nil, cfg.CurrentLocation); err != nil {
			return nil, err
		}
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	return &serverGroupCloudWorkflow{ctx: ctx, api: servergroups.New(&source.Client), cfg: cfg, check: guard}, nil
}
func serverGroupCloudRecord(v *servergroups.ServerGroupRecord) *servergroups.ServerGroupRecord {
	if v == nil {
		return nil
	}
	out := *v
	out.Resource = v.Resource.Clone()
	out.Wire = v.Wire.Clone()
	out.Envelope = bytes.Clone(v.Envelope)
	out.Header = v.Header.Clone()
	return &out
}
func serverGroupCloudLocation(body map[string]json.RawMessage, location *resource.CloudLocation) (json.RawMessage, error) {
	if location == nil {
		return json.RawMessage("null"), nil
	}
	project := body["project_id"]
	if project != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(project)
		if err != nil {
			return nil, err
		}
		if !truthy {
			project = nil
		} else {
			current := location.Project.ID
			if current == nil {
				current = json.RawMessage("null")
			}
			equal, err := jsonfilter.EqualPythonLoadedJSON(project, current)
			if err != nil {
				return nil, err
			}
			if equal {
				project = nil
			}
		}
	}
	return location.ForResource(project, nil)
}
func (p *serverGroupCloudWorkflow) locate(v *servergroups.ServerGroupRecord) (*servergroups.ServerGroupRecord, error) {
	v = serverGroupCloudRecord(v)
	if v == nil || v.Resource == nil {
		return v, p.check(p.ctx)
	}
	raw, err := serverGroupCloudLocation(v.Resource.Body, p.cfg.CurrentLocation)
	if err == nil {
		v.Resource.Body["location"] = bytes.Clone(raw)
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		v.Resource = nil
		return v, (&rest.Response{Body: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}).Fail(err)
	}
	return v, nil
}
func serverGroupCloudProof(inventory []*servergroups.ServerGroupRecord, cause error) error {
	if len(inventory) == 0 {
		return cause
	}
	row := inventory[len(inventory)-1]
	return (&rest.Response{Body: bytes.Clone(row.Envelope), Header: row.Header.Clone(), StatusCode: row.StatusCode}).Fail(cause)
}
func (p *serverGroupCloudWorkflow) search(name string) (*ServerGroupCloudSearchResult, error) {
	result := &ServerGroupCloudSearchResult{}
	views := make([]json.RawMessage, 0)
	for value, err := range p.api.ListCloudRecords(p.ctx) {
		row, locErr := p.locate(value)
		if row != nil {
			result.Inventory = append(result.Inventory, row)
		}
		if err = errors.Join(err, locErr, p.check(p.ctx)); err != nil {
			return result, err
		}
		if row == nil || row.Resource == nil {
			return result, invalid("server group list yielded no Resource")
		}
		raw, err := json.Marshal(row.Resource)
		if err != nil {
			return result, serverGroupCloudProof(result.Inventory, err)
		}
		views = append(views, bytes.Clone(raw))
	}
	selected, err := cloudfilter.SelectSource(views, name, p.cfg.Filters, func() error { return p.check(p.ctx) })
	if err != nil {
		return result, serverGroupCloudProof(result.Inventory, fmt.Errorf("%w: local server group filters: %w", resource.ErrInvalidOption, err))
	}
	if err := p.check(p.ctx); err != nil {
		return result, serverGroupCloudProof(result.Inventory, err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Groups = make([]*servergroups.ServerGroupRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Groups[i] = serverGroupCloudRecord(result.Inventory[index])
		}
	}
	return result, nil
}

// SearchServerGroups eagerly consumes all default project rows before applying
// exact/glob name/ID and local ordered dictionary/JMESPath filters. No filter is
// sent to Nova, and expression output is not fabricated as a ServerGroup.
func (s *Service) SearchServerGroups(ctx context.Context, nameOrID string, options ...ServerGroupCloudReadOption) (*ServerGroupCloudSearchResult, error) {
	p, err := s.serverGroupCloudStart(ctx, options)
	if err != nil {
		return nil, serverGroupCloudWrap(ctx, "SearchServerGroups", err)
	}
	result, err := p.search(nameOrID)
	return result, serverGroupCloudWrap(ctx, "SearchServerGroups", err)
}

// GetServerGroup chooses direct default find only for omitted/null filters.
// Every nonnull filter, including {}, false, 0, [] and "", selects complete
// cloud search followed by Source truthiness, len and integer index0 behavior.
func (s *Service) GetServerGroup(ctx context.Context, nameOrID string, options ...ServerGroupCloudReadOption) (*ServerGroupCloudGetResult, error) {
	const op = "GetServerGroup"
	p, err := s.serverGroupCloudStart(ctx, options)
	if err != nil {
		return nil, serverGroupCloudWrap(ctx, op, err)
	}
	result := &ServerGroupCloudGetResult{}
	filtered := p.cfg.Filters != nil && !bytes.Equal(bytes.TrimSpace(*p.cfg.Filters), []byte("null"))
	result.Filtered = filtered
	if !filtered {
		found, cause := p.api.FindCloudRecord(p.ctx, nameOrID)
		err = cause
		if found != nil {
			result.Observed = serverGroupCloudRecord(found.Observed)
			for _, row := range found.Inventory {
				result.Inventory = append(result.Inventory, serverGroupCloudRecord(row))
			}
			if err == nil {
				result.Group, err = p.locate(found.Group)
			}
		}
		if err == nil && result.Group != nil {
			result.Value, err = json.Marshal(result.Group.Resource)
		}
	} else {
		searched, cause := p.search(nameOrID)
		err = cause
		if searched != nil {
			result.Inventory = searched.Inventory
		}
		if err == nil {
			result.Value, err = cloudfilter.FirstLoaded(searched.Value)
			var multiple *cloudfilter.MultipleError
			if errors.As(err, &multiple) {
				err = fmt.Errorf("%w: server group %q has %d matches", resource.ErrAmbiguous, nameOrID, multiple.Length)
			} else if err != nil {
				err = fmt.Errorf("%w: filtered server group index0: %w", resource.ErrInvalidOption, err)
			}
			if err == nil && result.Value != nil && len(searched.Groups) == 1 {
				result.Group = serverGroupCloudRecord(searched.Groups[0])
			}
			if err != nil {
				err = serverGroupCloudProof(result.Inventory, err)
			}
		}
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		result.Value = nil
		result.Group = nil
	}
	return result, serverGroupCloudWrap(ctx, op, err)
}
