// Package actions reads Senlin asynchronous actions and their execution state.
package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[Action]
}

func New(client *gophercloud.ServiceClient) *API {
	return &API{client: client, Resources: rest.Collection(spec(client))}
}

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// Action keeps epoch fractions and other JSON numbers without float64 loss.
// Its ID is independent of TargetID and ClusterID.
type Action struct {
	resource.Metadata
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	TargetID     string                     `json:"target"`
	Action       string                     `json:"action"`
	Cause        string                     `json:"cause"`
	OwnerID      *string                    `json:"owner"`
	UserID       string                     `json:"user"`
	ProjectID    string                     `json:"project"`
	DomainID     string                     `json:"domain"`
	Interval     *json.Number               `json:"interval"`
	StartAt      *json.Number               `json:"start_time"`
	EndAt        *json.Number               `json:"end_time"`
	Timeout      *json.Number               `json:"timeout"`
	Status       string                     `json:"status"`
	StatusReason string                     `json:"status_reason"`
	Inputs       map[string]json.RawMessage `json:"inputs"`
	Outputs      map[string]json.RawMessage `json:"outputs"`
	Data         map[string]json.RawMessage `json:"data"`
	DependsOn    []string                   `json:"depends_on"`
	DependedBy   []string                   `json:"depended_by"`
	ClusterID    *string                    `json:"cluster_id"`
}

func (value *Action) UnmarshalJSON(data []byte) error {
	type plain Action
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Action] {
	return rest.CollectionSpec[Action]{
		Client: client, Path: "actions", Kind: "clustering.actions",
		SingleKey: "action", PluralKey: "actions", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID:        func(value *Action) string { return value.ID },
		Name:      func(value *Action) string { return value.Name },
		NameQuery: func(name string) string { return name },
		Status:    func(value *Action) string { return value.Status },
		Metadata:  func(value *Action) *resource.Metadata { return &value.Metadata },
		Validate:  func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateQuery: func(ctx context.Context, query url.Values) error {
			return senlin.Sort(query.Get("sort"), "name", "target", "action", "created_at", "status")
		},
		ValidateID: senlin.Identifier,
		Paging: rest.PagePolicy[Action]{HTTPLink: true, MarkerFallback: true,
			Marker: func(value *Action) (string, error) {
				if value == nil {
					return "", fmt.Errorf("%w: action pagination row is missing", resource.ErrInvalidOption)
				}
				return value.ID, senlin.Identifier(value.ID)
			}},
	}
}

// Get requests a name, UUID or short-ID directly; no UUID heuristic is used.
func (a *API) Get(ctx context.Context, id string) (*Action, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, id)
}

// ListOpts omits zero/empty parameters; GlobalProject nil leaves the false
// server default untouched, while a non-nil false is explicitly serialized.
type ListOpts struct {
	Limit         int
	Marker        string
	Name          string
	TargetID      string
	Action        string
	Status        string
	Sort          string
	GlobalProject *bool
	// ClusterID is inherited from pinned Python, beyond the published query
	// table. The deployment validates its support.
	ClusterID string
}

type ListOption = request.Option[ListOpts]

func cloneOptions(value ListOpts) ListOpts {
	value.GlobalProject = senlin.Bool(value.GlobalProject)
	return value
}

func WithListOptions(value ListOpts) ListOption {
	value = cloneOptions(value)
	return func(config *request.Config[ListOpts]) error {
		config.Options = cloneOptions(value)
		return nil
	}
}

func WithListGlobalProject(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	}
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	if err := request.ValidateCapabilities(config, false, true, false); err != nil {
		return nil, err
	}
	value := cloneOptions(config.Options)
	if err := senlin.Sort(value.Sort, "name", "target", "action", "created_at", "status"); err != nil {
		return nil, err
	}
	query, err := senlin.Query(request.Config[senlin.ListOpts]{Options: senlin.ListOpts{Limit: value.Limit, Marker: value.Marker}})
	if err != nil {
		return nil, err
	}
	for key, field := range map[string]string{"name": value.Name, "target": value.TargetID, "action": value.Action, "status": value.Status, "sort": value.Sort, "cluster_id": value.ClusterID} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "name", "target", "target_id", "action", "status", "sort", "global_project", "cluster_id":
			return nil, fmt.Errorf("%w: extension %q is a concrete action list option", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Action, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Action, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var query url.Values
		if err == nil {
			query, err = listQuery(config)
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.actions", err))
			return
		}
		for value, err := range rest.List(ctx, spec(a.RawClient()), query) {
			if !yield(value, request.Wrap("List", "clustering.actions", err)) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Action, error) {
	return senlin.All(a.List(ctx, options...))
}
