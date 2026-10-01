package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	client *gophercloud.ServiceClient
	// Resources supplies shared reads, lookup and polling. Use API.Delete to
	// retain the asynchronous action submission rather than discarding it.
	Resources *resource.Collection[Node]
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

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Node] {
	return rest.CollectionSpec[Node]{
		Client: client, Path: "nodes", Kind: "clustering.nodes", SingleKey: "node", PluralKey: "nodes", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID: func(value *Node) string { return value.ID }, Name: func(value *Node) string { return value.Name },
		NameQuery: func(name string) string { return name }, Status: func(value *Node) string { return value.Status },
		Metadata: func(value *Node) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.Validate(ctx, client) }, ValidateID: senlin.Identifier,
		ValidateQuery: func(_ context.Context, query url.Values) error { return senlin.SortGrammar(query.Get("sort")) },
		Paging: rest.PagePolicy[Node]{HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true, Marker: func(value *Node) (string, error) {
			if value == nil {
				return "", fmt.Errorf("%w: missing node pagination row", resource.ErrInvalidOption)
			}
			return value.ID, senlin.Identifier(value.ID)
		}},
	}
}

var nodeName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func validateName(name string) error {
	if err := senlin.Required(name); err != nil {
		return err
	}
	if len(name) >= 255 || !nodeName.MatchString(name) {
		return fmt.Errorf("%w: node name must start with an ASCII letter and contain fewer than 255 ASCII letters, digits, underscores, periods or hyphens", resource.ErrInvalidOption)
	}
	return nil
}

var responseFields = []string{"node", "id", "physical_id", "project", "project_id", "domain", "domain_id", "user", "user_id", "profile_name", "index", "init_at", "created_at", "updated_at", "status", "status_reason", "data", "details", "dependents"}

func submission(client *gophercloud.ServiceClient, response *rest.Response) (*actions.Submission, error) {
	identity, err := senlin.ActionID(client, response)
	if err != nil {
		return nil, err
	}
	return &actions.Submission{ActionID: identity, Location: senlin.LocationValues(response.Header)[0], Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

func decodeMutation(client *gophercloud.ServiceClient, response *rest.Response, requireLocation bool) (*Node, error) {
	value, err := rest.Decode(response, "node", func(value *Node) *resource.Metadata { return &value.Metadata })
	if err != nil {
		return nil, err
	}
	if requireLocation || len(senlin.LocationValues(response.Header)) != 0 {
		value.Operation, err = submission(client, response)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func (a *API) Create(ctx context.Context, value CreateOpts, options ...CreateOption) (*Node, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Create", "clustering.nodes", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = validateName(config.Options.Name)
	}
	if err == nil {
		err = senlin.Required(config.Options.ProfileID)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "node", append(append([]string(nil), responseFields...), "tainted")...)
	}
	if err != nil {
		return nil, request.Wrap("Create", "clustering.nodes", err)
	}
	extraHeaders := maps.Clone(config.Headers)
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Create", "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPost, client.ServiceURL("nodes"), body, extraHeaders, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Create", "clustering.nodes", err)
	}
	result, err := decodeMutation(client, response, true)
	return result, request.Wrap("Create", "clustering.nodes", err)
}

// Get sends the controller identity directly. WithGetDetails(true) requests
// physical-resource details, while an omitted option leaves server defaults.
func (a *API) Get(ctx context.Context, identity string, options ...GetOption) (*Node, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Get", "clustering.nodes", err)
	}
	if err := senlin.Identifier(identity); err != nil {
		return nil, request.Wrap("Get", "clustering.nodes", err)
	}
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, true)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	if err != nil {
		return nil, request.Wrap("Get", "clustering.nodes", err)
	}
	query := make(url.Values)
	if config.Options.Details != nil {
		query.Set("show_details", strconv.FormatBool(*config.Options.Details))
	}
	for key, values := range config.Query {
		if key == "show_details" || key == "details" {
			return nil, request.Wrap("Get", "clustering.nodes", fmt.Errorf("%w: query %q is a concrete get option", resource.ErrInvalidOption, key))
		}
		query[key] = append([]string(nil), values...)
	}
	target := client.ServiceURL("nodes", url.PathEscape(identity))
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Get", "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, target, nil, config.Headers, http.StatusOK)
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "clustering.nodes", Reference: identity, Cause: err}
	}
	if err != nil {
		return nil, request.Wrap("Get", "clustering.nodes", err)
	}
	result, err := rest.Decode(response, "node", func(value *Node) *resource.Metadata { return &value.Metadata })
	return result, request.Wrap("Get", "clustering.nodes", err)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Node, error) {
	options = append([]resource.LookupOption{resource.WithIgnoreMissing()}, options...)
	return rest.Collection(spec(a.RawClient())).Find(ctx, ref, options...)
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Node, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Node, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, err = listQuery(config)
		}
		if err == nil {
			filters, err = prepareFilters(config)
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.nodes", err))
			return
		}
		for value, err := range rest.List(ctx, spec(a.RawClient()), query) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.nodes", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.nodes", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Node, error) {
	return senlin.All(a.List(ctx, options...))
}

func validateUpdate(ctx context.Context, client *gophercloud.ServiceClient, value UpdateOpts) error {
	if err := senlin.Validate(ctx, client); err != nil {
		return err
	}
	if value.Name.IsSet() && !value.Name.IsNull() {
		name, _ := value.Name.Get()
		if err := validateName(name); err != nil {
			return err
		}
	}
	if err := senlin.OptionalObject(value.Metadata, "metadata"); err != nil {
		return err
	}
	if value.Tainted.IsSet() {
		return senlin.RequireVersion(ctx, client, 13)
	}
	return nil
}

func (a *API) Update(ctx context.Context, ref resource.Ref, value UpdateOpts, options ...UpdateOption) (*Node, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Update", "clustering.nodes", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = validateUpdate(ctx, client, config.Options)
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "node", append(append([]string(nil), responseFields...), "cluster_id")...)
	}
	if err != nil {
		return nil, request.Wrap("Update", "clustering.nodes", err)
	}
	requiresTainted := config.Options.Tainted.IsSet()
	extraHeaders := maps.Clone(config.Headers)
	identity, err := rest.Collection(spec(client)).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.nodes", err)
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Update", "clustering.nodes", err)
	}
	if requiresTainted {
		if err := senlin.RequireVersion(ctx, client, 13); err != nil {
			return nil, request.Wrap("Update", "clustering.nodes", err)
		}
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL("nodes", url.PathEscape(identity)), body, extraHeaders, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.nodes", err)
	}
	result, err := decodeMutation(client, response, true)
	return result, request.Wrap("Update", "clustering.nodes", err)
}

// Delete retains the asynchronous Location action. Missing nodes return nil
// by default. Force true sends the pinned Python force_delete JSON request;
// false uses ordinary deletion, force uses strict missing handling by default,
// and conflicts are never ignored.
func (a *API) Delete(ctx context.Context, ref resource.Ref, options ...DeleteOption) (*actions.Submission, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Delete", "clustering.nodes", err)
	}
	config, err := request.Apply(DeleteOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	if err != nil {
		return nil, request.Wrap("Delete", "clustering.nodes", err)
	}
	ignoreMissing := !config.Options.Force
	if config.Options.IgnoreMissing != nil {
		ignoreMissing = *config.Options.IgnoreMissing
	}
	var body any
	if config.Options.Force {
		body = json.RawMessage(`{"force":true}`)
	}
	extraHeaders := maps.Clone(config.Headers)
	identity, err := rest.Collection(spec(client)).ResolveID(ctx, ref)
	if ignoreMissing && errors.Is(err, resource.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, request.Wrap("Delete", "clustering.nodes", err)
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Delete", "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodDelete, client.ServiceURL("nodes", url.PathEscape(identity)), body, extraHeaders, http.StatusAccepted)
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		if ignoreMissing {
			return nil, nil
		}
		err = &resource.NotFoundError{Resource: "clustering.nodes", Reference: ref.String(), Cause: err}
	}
	if err != nil {
		return nil, request.Wrap("Delete", "clustering.nodes", err)
	}
	result, err := submission(client, response)
	return result, request.Wrap("Delete", "clustering.nodes", err)
}
