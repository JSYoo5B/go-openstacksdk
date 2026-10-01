// Package events reads Senlin's immutable event records.
package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[Event]
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

type Event struct {
	resource.Metadata
	ID           string          `json:"id"`
	GeneratedAt  *string         `json:"timestamp"`
	ObjectID     string          `json:"oid"`
	ObjectName   string          `json:"oname"`
	ObjectType   string          `json:"otype"`
	ClusterID    *string         `json:"cluster_id"`
	Level        string          `json:"level"`
	UserID       string          `json:"user"`
	ProjectID    string          `json:"project"`
	Action       string          `json:"action"`
	Status       string          `json:"status"`
	StatusReason string          `json:"status_reason"`
	MetaData     json.RawMessage `json:"meta_data"`
}

func (value *Event) UnmarshalJSON(data []byte) error {
	type plain Event
	var decoded plain
	wire := struct {
		*plain
		Level json.RawMessage `json:"level"`
	}{plain: &decoded}
	if err := resource.DecodeObject(data, &wire, &decoded.Metadata); err != nil {
		return err
	}
	level := bytes.TrimSpace(wire.Level)
	if len(level) != 0 && !bytes.Equal(level, []byte("null")) {
		if err := json.Unmarshal(level, &decoded.Level); err != nil {
			var number json.Number
			if err := json.Unmarshal(level, &number); err != nil {
				return fmt.Errorf("event level must be a JSON string or number: %w", err)
			}
			decoded.Level = number.String()
		}
	}
	*value = Event(decoded)
	return nil
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Event] {
	return rest.CollectionSpec[Event]{
		Client: client, Path: "events", Kind: "clustering.events",
		SingleKey: "event", PluralKey: "events", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID:       func(value *Event) string { return value.ID },
		Metadata: func(value *Event) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateQuery: func(ctx context.Context, query url.Values) error {
			return senlin.Sort(query.Get("sort"), "timestamp", "level", "otype", "oname", "action", "status", "oid", "cluster_id")
		},
		ValidateID: senlin.Identifier,
		Paging: rest.PagePolicy[Event]{MaxItemsLimitHint: true, StopOnEmptyPage: true, HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *Event) (string, error) {
				if value == nil {
					return "", fmt.Errorf("%w: event pagination row is missing", resource.ErrInvalidOption)
				}
				return value.ID, senlin.Identifier(value.ID)
			}},
	}
}

func (a *API) Get(ctx context.Context, id string) (*Event, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, id)
}

// ListOpts supports repeated object/action filters. Nil GlobalProject omits
// the false-default flag. Empty slices and strings omit their filters.
type ListOpts struct {
	// MaxItems counts wire rows before local filtering; zero is unlimited.
	MaxItems int
	// Paginated nil uses all pages; an explicit false returns one page.
	Paginated     *bool
	Limit         int
	Marker        string
	ObjectIDs     []string
	ObjectNames   []string
	ObjectTypes   []string
	Actions       []string
	ClusterID     string
	Level         string
	Sort          string
	GlobalProject *bool
}

type ListOption = request.Option[ListOpts]

func cloneOptions(value ListOpts) ListOpts {
	value.ObjectIDs = slices.Clone(value.ObjectIDs)
	value.ObjectNames = slices.Clone(value.ObjectNames)
	value.ObjectTypes = slices.Clone(value.ObjectTypes)
	value.Actions = slices.Clone(value.Actions)
	value.GlobalProject = senlin.Bool(value.GlobalProject)
	value.Paginated = senlin.Bool(value.Paginated)
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

// WithListMaxItems limits wire rows before local filtering. Zero is unlimited.
func WithListMaxItems(value int) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.MaxItems = value; return nil }
}

// WithListPaginated controls continuation without changing server query fields.
func WithListPaginated(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.Paginated = &copy
		return nil
	}
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	if err := request.ValidateCapabilities(config, false, true, false, filterSpec.Namespace); err != nil {
		return nil, err
	}
	value := cloneOptions(config.Options)
	if value.MaxItems < 0 {
		return nil, fmt.Errorf("%w: max items must be non-negative", resource.ErrInvalidOption)
	}
	if err := senlin.Sort(value.Sort, "timestamp", "level", "otype", "oname", "action", "status", "oid", "cluster_id"); err != nil {
		return nil, err
	}
	query, err := senlin.Query(request.Config[senlin.ListOpts]{Options: senlin.ListOpts{Limit: value.Limit, Marker: value.Marker}})
	if err != nil {
		return nil, err
	}
	for key, values := range map[string][]string{"oid": value.ObjectIDs, "oname": value.ObjectNames, "otype": value.ObjectTypes, "action": value.Actions} {
		for _, field := range values {
			if field == "" {
				return nil, fmt.Errorf("%w: event filter %q cannot contain empty values", resource.ErrInvalidOption, key)
			}
			query.Add(key, field)
		}
	}
	for key, field := range map[string]string{"cluster_id": value.ClusterID, "level": value.Level, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "max_items", "paginated", "oid", "obj_id", "oname", "obj_name", "otype", "obj_type", "action", "cluster_id", "level", "sort", "global_project":
			return nil, fmt.Errorf("%w: extension %q is a concrete event list option", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	if err := senlin.RejectBodyFilterQuery(config.Query, filterSpec); err != nil {
		return nil, err
	}
	return query, nil
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Event, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Event, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, err = listQuery(config)
		}
		if err == nil {
			filters, err = senlin.PrepareBodyFilters(config, filterSpec)
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.events", err))
			return
		}
		for value, err := range rest.ListWithControl(ctx, spec(a.RawClient()), query, rest.ListControl{MaxItems: config.Options.MaxItems, SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated, LimitHint: true}) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.events", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.events", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Event, error) {
	return senlin.All(a.List(ctx, options...))
}
