package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
	"slices"
)

type ProjectGetOpts struct {
	DomainID     json.RawMessage
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type ProjectGetOption func(*ProjectGetOpts) error

func copyProjectGetOpts(v ProjectGetOpts) ProjectGetOpts {
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
func WithProjectGetOpts(v ProjectGetOpts) ProjectGetOption {
	owned := copyProjectGetOpts(v)
	return func(c *ProjectGetOpts) error { *c = copyProjectGetOpts(owned); return nil }
}
func WithProjectGetDomainID(v string) ProjectGetOption {
	raw, _ := json.Marshal(v)
	return WithProjectGetRawDomainID(raw)
}
func WithProjectGetRawDomainID(v json.RawMessage) ProjectGetOption {
	owned := bytes.Clone(v)
	return func(c *ProjectGetOpts) error { c.DomainID = bytes.Clone(owned); return nil }
}
func WithProjectGetFilters(v json.RawMessage) ProjectGetOption {
	owned := bytes.Clone(v)
	return func(c *ProjectGetOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithProjectGetHeaders(v map[string]string) ProjectGetOption {
	owned := maps.Clone(v)
	return func(c *ProjectGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithProjectGetMicroversion(v string) ProjectGetOption {
	return func(c *ProjectGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithProjectGetLocation(v resource.CloudLocation) ProjectGetOption {
	owned := v.Clone()
	return func(c *ProjectGetOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

type ProjectGetResult struct {
	Value             json.RawMessage
	Project           *projects.ProjectRecord
	Inventory         []*projects.ProjectRecord
	DeprecatedFilters bool
}

func cloneProjectGetRecord(v *projects.ProjectRecord) *projects.ProjectRecord {
	if v == nil {
		return nil
	}
	return &projects.ProjectRecord{Resource: v.Resource.Clone(), Wire: v.Wire.Clone(), Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
}

// projectGetListFilters preserves list_projects' truthy domain override without
// adding it to _filter_list's original local filters. Object insertion order is
// retained, and a truthy nonmapping filters argument fails before HTTP.
func projectGetListFilters(policy ProjectGetOpts) (*json.RawMessage, error) {
	var members []cloudfilter.JSONMember
	if policy.Filters != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(*policy.Filters)
		if err != nil {
			return nil, err
		}
		if truthy {
			members, err = cloudfilter.ObjectMembers(*policy.Filters)
			if err != nil {
				return nil, errors.Join(projectReadInvalid("get_project list dict(**filters)"), err)
			}
		}
	}
	if policy.DomainID == nil {
		return policy.Filters, nil
	}
	truthy, err := cloudfilter.PythonJSONTruthy(policy.DomainID)
	if err != nil {
		return nil, err
	}
	if !truthy {
		return policy.Filters, nil
	}
	found := false
	for i := range members {
		if members[i].Key == "domain_id" {
			members[i].Value = bytes.Clone(policy.DomainID)
			found = true
		}
	}
	if !found {
		members = append(members, cloudfilter.JSONMember{Key: "domain_id", Value: bytes.Clone(policy.DomainID)})
	}
	var buffer bytes.Buffer
	buffer.WriteByte('{')
	for i, member := range members {
		if i > 0 {
			buffer.WriteByte(',')
		}
		key, _ := json.Marshal(member.Key)
		buffer.Write(key)
		buffer.WriteByte(':')
		buffer.Write(member.Value)
	}
	buffer.WriteByte('}')
	value := json.RawMessage(buffer.Bytes())
	return &value, nil
}
func (c *ProjectReadCloud) projectGetSearch(ctx context.Context, api *projects.API, id string, policy ProjectGetOpts, guard func() error) (*ProjectSearchResult, error) {
	effective, err := projectGetListFilters(policy)
	if err != nil {
		return nil, errors.Join(projectReadInvalid("get_project search arguments"), err)
	}
	listed, err := c.listPrepared(ctx, api, ProjectReadOpts{Filters: effective, Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location}, false, guard)
	result := &ProjectSearchResult{Inventory: make([]*projects.ProjectRecord, 0)}
	if listed != nil {
		for _, row := range listed.Inventory {
			result.Inventory = append(result.Inventory, cloneProjectGetRecord(row))
		}
	}
	if err != nil {
		return result, errors.Join(err, guard())
	}
	filterTruthy := false
	if policy.Filters != nil {
		filterTruthy, err = cloudfilter.PythonJSONTruthy(*policy.Filters)
		if err != nil {
			return result, err
		}
	}
	if id == "" && !filterTruthy {
		result.Value = bytes.Clone(listed.JSON)
		if listed.Projects != nil {
			result.Projects = make([]*projects.ProjectRecord, len(listed.Projects))
			for i, row := range listed.Projects {
				result.Projects[i] = cloneProjectGetRecord(row)
			}
		}
		return result, guard()
	}
	rows := make([]json.RawMessage, len(listed.Value))
	for i, value := range listed.Value {
		if err := guard(); err != nil {
			return result, err
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return result, errors.Join(projectReadInvalid("get_project opaque search row"), err)
		}
		rows[i] = bytes.Clone(raw)
	}
	selectionRows := rows
	selectionFilters := policy.Filters
	// With no identifier, Source _dict_filter can discard falsey scalar literal
	// rows. A private wrapper reuses the common lazy nested mapping predicate.
	if id == "" && filterTruthy {
		selectionRows = make([]json.RawMessage, len(rows))
		for i, row := range rows {
			selectionRows[i] = append(append(json.RawMessage(`{"value":`), row...), '}')
		}
		wrapped := append(append(json.RawMessage(`{"value":`), *policy.Filters...), '}')
		selectionFilters = &wrapped
	}
	selected, err := cloudfilter.SelectLoaded(selectionRows, id, selectionFilters, guard)
	if err != nil {
		return result, errors.Join(projectReadInvalid("get_project local search"), err)
	}
	chosen := make([]json.RawMessage, len(selected.Indices))
	for i, index := range selected.Indices {
		chosen[i] = bytes.Clone(rows[index])
	}
	result.Value, err = json.Marshal(chosen)
	if listed.Projects != nil {
		result.Projects = make([]*projects.ProjectRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Projects[i] = cloneProjectGetRecord(listed.Projects[index])
		}
	}
	return result, errors.Join(err, guard())
}

// GetProject mirrors the actual pinned cloud function, including list_projects'
// kwargs-expansion failure before any deprecated truthy string JMES filter.
func (c *ProjectReadCloud) GetProject(ctx context.Context, nameOrID string, options ...ProjectGetOption) (*ProjectGetResult, error) {
	wrap := func(err error) error { return request.Wrap("GetProject", "projects", cloudread.ContextError(ctx, err)) }
	owned, api, _, guard, err := c.prepare(ctx, nil)
	if err != nil {
		return nil, wrap(err)
	}
	policy := copyProjectGetOpts(ProjectGetOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(projectReadInvalid("nil Project get option"))
		}
		next := copyProjectGetOpts(policy)
		if err := errors.Join(apply(&next), guard()); err != nil {
			return nil, wrap(err)
		}
		policy = copyProjectGetOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, policy.Location.Zone); err != nil {
			return nil, wrap(errors.Join(err, guard()))
		}
	}
	result := &ProjectGetResult{Inventory: make([]*projects.ProjectRecord, 0)}
	if policy.Filters == nil || bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		record, err := api.ResolveRecord(owned, nameOrID, projects.WithRecordResolveOpts(projects.RecordResolveOpts{DomainID: policy.DomainID, Headers: policy.Headers, Microversion: policy.Microversion, Location: policy.Location}))
		if record != nil {
			result.Inventory = append(result.Inventory, cloneProjectGetRecord(record))
			result.Project = cloneProjectGetRecord(record)
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
	selected, err := c.projectGetSearch(owned, api, nameOrID, policy, guard)
	if selected != nil {
		result.Inventory = selected.Inventory
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
			err = errors.Join(projectReadInvalid("get_project truth/len/index"), err)
		}
		return result, wrap(errors.Join(err, guard()))
	}
	if result.Value != nil && len(selected.Projects) == 1 {
		result.Project = cloneProjectGetRecord(selected.Projects[0])
	}
	return result, wrap(guard())
}
