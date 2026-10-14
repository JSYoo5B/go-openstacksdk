package roles

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/gophercloud/gophercloud/v2"
)

// assignmentSpec is the pinned openstacksdk identity v3 RoleAssignment
// Resource: Body id, name, links, role, scope, user and group; the dotted
// QueryParameters below plus limit and marker. Python accepts both the
// client name and the server name of each query; the client name wins when
// both are given. The remaining Body attributes filter rows locally, and a
// dictionary filter matches a nested subset as Resource.list _dict_filter does.
var assignmentSpec = &osfederation.Spec{
	Kind: "role_assignments", ResourceKey: "role_assignment", ResourcesKey: "role_assignments",
	Fields: []string{"id", "name", "links", "role", "scope", "user", "group"},
	Attrs:  map[string]string{},
	Query: map[string]string{
		"group_id": "group.id", "group.id": "group.id",
		"role_id": "role.id", "role.id": "role.id",
		"user_id": "user.id", "user.id": "user.id",
		"scope_project_id": "scope.project.id", "scope.project.id": "scope.project.id",
		"scope_domain_id": "scope.domain.id", "scope.domain.id": "scope.domain.id",
		"scope_system": "scope.system", "scope.system": "scope.system",
		"inherited_to": "scope.OS-INHERIT:inherited_to", "scope.OS-INHERIT:inherited_to": "scope.OS-INHERIT:inherited_to",
		"effective": "effective", "include_names": "include_names", "include_subtree": "include_subtree",
	},
	Filters: map[string]string{"id": "id", "name": "name", "links": "links", "role": "role", "scope": "scope", "user": "user", "group": "group"},
}

// AssignmentRecord is the Python RoleAssignment Resource view. Role, Scope,
// User, Group and Links are the raw JSON objects Keystone returned, so the
// scope OS-INHERIT:inherited_to key and include_names fields are kept. Body
// keeps the seven declared Body keys as raw JSON (null when unknown); Wire
// is the actual row, Envelope the actual page body, and Header and
// StatusCode the actual page response.
type AssignmentRecord struct {
	Role, Scope, User, Group, Links json.RawMessage

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func assignmentRecord(r *osfederation.Record) *AssignmentRecord {
	if r == nil {
		return nil
	}
	r = r.Clone()
	raw := func(key string) json.RawMessage {
		if value := r.View[key]; string(value) != "null" {
			return value
		}
		return nil
	}
	return &AssignmentRecord{Role: raw("role"), Scope: raw("scope"), User: raw("user"), Group: raw("group"), Links: raw("links"),
		Body: r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func (a *API) assignmentTarget() osfederation.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return osfederation.Target{Spec: assignmentSpec, Client: client, Path: "role_assignments"}
}

// ListAssignmentRecords implements role_assignments(**query) as a lazy
// iterator over GET role_assignments.
func (a *API) ListAssignmentRecords(ctx context.Context, options ...AssignmentRecordListOption) iter.Seq2[*AssignmentRecord, error] {
	source := osfederation.List(ctx, a.assignmentTarget(), options)
	return func(yield func(*AssignmentRecord, error) bool) {
		for record, err := range source {
			if !yield(assignmentRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// AllAssignmentRecords collects ListAssignmentRecords; a nonnil empty slice on success.
func (a *API) AllAssignmentRecords(ctx context.Context, options ...AssignmentRecordListOption) ([]*AssignmentRecord, error) {
	values := make([]*AssignmentRecord, 0)
	for value, err := range a.ListAssignmentRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// AssignmentRecordListOpts carries Python **query. role_id, user_id,
// group_id, scope_project_id, scope_domain_id, scope_system, inherited_to
// (or their dotted server names), effective, include_names and
// include_subtree go to the server encoded like Requests; id, name, links,
// role, scope, user and group filter rows locally; other names are ignored.
type AssignmentRecordListOpts = osfederation.ListOpts
type AssignmentRecordListOption func(*AssignmentRecordListOpts) error

func WithAssignmentRecordListOpts(opts AssignmentRecordListOpts) AssignmentRecordListOption {
	return AssignmentRecordListOption(osfederation.ListSnapshot(opts))
}
func WithAssignmentRecordListQuery(key string, value any) AssignmentRecordListOption {
	return AssignmentRecordListOption(assignmentSpec.ListQuery(key, value))
}
func WithAssignmentRecordListLimit(limit int) AssignmentRecordListOption {
	return AssignmentRecordListOption(osfederation.ListLimit(limit))
}
func WithAssignmentRecordListMarker(marker string) AssignmentRecordListOption {
	return AssignmentRecordListOption(osfederation.ListMarker(marker))
}
func WithAssignmentRecordListMaxItems(count int) AssignmentRecordListOption {
	return AssignmentRecordListOption(osfederation.ListMaxItems(count))
}
func WithAssignmentRecordListPaginated(value bool) AssignmentRecordListOption {
	return AssignmentRecordListOption(osfederation.ListPaginated(value))
}
func WithAssignmentRecordListHeaders(values map[string]string) AssignmentRecordListOption {
	return AssignmentRecordListOption(osfederation.ListHeaders(values))
}
