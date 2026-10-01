// Package receivers manages synchronous Senlin webhook and message receivers.
package receivers

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[Receiver]
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

// Receiver retains response attributes without interpreting or following the
// channel URL. Action is a command name rather than a Senlin action UUID.
type Receiver struct {
	resource.Metadata
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Type      string                     `json:"type"`
	UserID    string                     `json:"user"`
	ProjectID string                     `json:"project"`
	DomainID  string                     `json:"domain"`
	ClusterID *string                    `json:"cluster_id"`
	Action    *string                    `json:"action"`
	Actor     map[string]json.RawMessage `json:"actor"`
	Params    map[string]json.RawMessage `json:"params"`
	Channel   map[string]json.RawMessage `json:"channel"`
}

func (value *Receiver) UnmarshalJSON(data []byte) error {
	type plain Receiver
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = Receiver(decoded)
	return nil
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Receiver] {
	return rest.CollectionSpec[Receiver]{
		Client: client, Path: "receivers", Kind: "clustering.receivers", SingleKey: "receiver", PluralKey: "receivers", Get: true, Delete: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK}, DeleteCodes: []int{http.StatusNoContent},
		ID: func(value *Receiver) string { return value.ID }, Name: func(value *Receiver) string { return value.Name },
		NameQuery: func(name string) string { return name }, Metadata: func(value *Receiver) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.Validate(ctx, client) }, ValidateID: senlin.Identifier,
		ValidateQuery: func(ctx context.Context, query url.Values) error { return validateQuery(ctx, client, query) },
		Paging: rest.PagePolicy[Receiver]{HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *Receiver) (string, error) {
				if value == nil {
					return "", fmt.Errorf("%w: missing receiver pagination row", resource.ErrInvalidOption)
				}
				return value.ID, senlin.Identifier(value.ID)
			}},
	}
}

var responseFields = []string{"receiver", "id", "user", "user_id", "project", "project_id", "domain", "domain_id", "channel", "created_at", "updated_at"}

func validateCreate(value CreateOpts) error {
	if err := senlin.Required(value.Name, value.Type); err != nil {
		return err
	}
	if value.Type == "webhook" {
		cluster, _ := value.ClusterID.Get()
		action, _ := value.Action.Get()
		if err := senlin.Required(cluster, action); err != nil {
			return err
		}
	}
	if err := senlin.OptionalObject(value.Actor, "actor"); err != nil {
		return err
	}
	return senlin.OptionalObject(value.Params, "params")
}

func (a *API) Create(ctx context.Context, value CreateOpts, options ...CreateOption) (*Receiver, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Create", "clustering.receivers", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = validateCreate(config.Options)
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "receiver", responseFields...)
	}
	if err != nil {
		return nil, request.Wrap("Create", "clustering.receivers", err)
	}
	headers := maps.Clone(config.Headers)
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Create", "clustering.receivers", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPost, client.ServiceURL("receivers"), body, headers, http.StatusCreated)
	if err != nil {
		return nil, request.Wrap("Create", "clustering.receivers", err)
	}
	result, err := rest.Decode(response, "receiver", func(value *Receiver) *resource.Metadata { return &value.Metadata })
	return result, request.Wrap("Create", "clustering.receivers", err)
}

// Get sends the controller identity directly, with no UUID heuristic or lookup.
func (a *API) Get(ctx context.Context, identity string) (*Receiver, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, identity)
}

func (a *API) Update(ctx context.Context, ref resource.Ref, value UpdateOpts, options ...UpdateOption) (*Receiver, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = senlin.OptionalObject(config.Options.Params, "params")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "receiver", append(append([]string(nil), responseFields...), "type", "cluster_id", "actor")...)
	}
	if err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	headers := maps.Clone(config.Headers)
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	identity, err := rest.Collection(spec(client)).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL("receivers", url.PathEscape(identity)), body, headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	result, err := rest.Decode(response, "receiver", func(value *Receiver) *resource.Metadata { return &value.Metadata })
	return result, request.Wrap("Update", "clustering.receivers", err)
}

// Delete ignores missing receivers by default. WithMissingError is strict.
func (a *API) Delete(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return rest.Collection(spec(a.RawClient())).Delete(ctx, ref, options...)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Receiver, error) {
	options = append([]resource.LookupOption{resource.WithIgnoreMissing()}, options...)
	return rest.Collection(spec(a.RawClient())).Find(ctx, ref, options...)
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Receiver, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Receiver, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, err = listQuery(ctx, a.RawClient(), config)
		}
		if err == nil {
			filters, err = prepareFilters(config)
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.receivers", err))
			return
		}
		for value, err := range rest.List(ctx, spec(a.RawClient()), query) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.receivers", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.receivers", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Receiver, error) {
	return senlin.All(a.List(ctx, options...))
}
