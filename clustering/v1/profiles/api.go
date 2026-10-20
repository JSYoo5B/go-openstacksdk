package profiles

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"regexp"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[Profile]
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

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Profile] {
	return rest.CollectionSpec[Profile]{
		Client: client, Path: "profiles", Kind: "clustering.profiles", SingleKey: "profile", PluralKey: "profiles",
		Get: true, Delete: true, GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK}, DeleteCodes: []int{http.StatusNoContent},
		ID: func(value *Profile) string { return value.ID }, Name: func(value *Profile) string { return value.Name },
		NameQuery: func(name string) string { return name },
		Metadata:  func(value *Profile) *resource.Metadata { return &value.Metadata },
		Validate:  func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateQuery: func(ctx context.Context, query url.Values) error {
			return senlin.SortGrammar(query.Get("sort"))
		},
		ValidateID: senlin.Identifier,
		Paging: rest.PagePolicy[Profile]{MaxItemsLimitHint: true, StopOnEmptyPage: true, HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *Profile) (string, error) {
				if value == nil {
					return "", fmt.Errorf("%w: missing profile pagination row", resource.ErrInvalidOption)
				}
				return value.ID, senlin.Identifier(value.ID)
			}},
	}
}

// Get accepts a name, UUID, or short ID as the direct Senlin route identity.
func (a *API) Get(ctx context.Context, identity string) (*Profile, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, identity)
}

func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Profile, error) {
	options = append([]resource.LookupOption{resource.WithIgnoreMissing()}, options...)
	return rest.Collection(spec(a.RawClient())).Find(ctx, ref, options...)
}

func (a *API) Delete(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return rest.Collection(spec(a.RawClient())).Delete(ctx, ref, options...)
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Profile, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Profile, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var client *gophercloud.ServiceClient
		if err == nil {
			client, err = senlin.PrepareListClient(ctx, a.RawClient(), config)
		}
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, err = listQuery(config)
		}
		if err == nil {
			filters, err = prepareFilters(senlin.ListQueryConfig(config))
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.profiles", err))
			return
		}
		for value, err := range rest.ListWithControl(ctx, senlin.ListSpec(a.RawClient(), client, spec), query, rest.ListControl{MaxItems: config.Options.MaxItems, SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated, LimitHint: true}) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.profiles", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.profiles", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Profile, error) {
	return senlin.All(a.List(ctx, options...))
}

var profileName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func validateName(name string) error {
	if err := senlin.Required(name); err != nil {
		return err
	}
	if len(name) >= 255 || !profileName.MatchString(name) {
		return fmt.Errorf("%w: profile name must start with an ASCII letter and contain fewer than 255 ASCII letters, digits, underscores, periods or hyphens", resource.ErrInvalidOption)
	}
	return nil
}

var responseFields = []string{"id", "type", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at", "profile"}

func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Profile, error) {
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Create", "clustering.profiles", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = validateName(config.Options.Name)
	}
	if err == nil {
		err = senlin.Object(config.Options.Spec, "spec")
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	if err != nil {
		return nil, request.Wrap("Create", "clustering.profiles", err)
	}
	body, err := senlin.Body(config, "profile", responseFields...)
	if err != nil {
		return nil, request.Wrap("Create", "clustering.profiles", err)
	}
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Create", "clustering.profiles", err)
	}
	response, err := rest.DoJSON(ctx, a.RawClient(), http.MethodPost, a.RawClient().ServiceURL("profiles"), body, config.Headers, http.StatusCreated)
	if err != nil {
		return nil, request.Wrap("Create", "clustering.profiles", err)
	}
	value, err := rest.Decode(response, "profile", func(value *Profile) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Create", "clustering.profiles", err)
}

// Update resolves explicit names once. Empty stateless updates are invalid;
// this API does not track dirty fields on an existing model.
func (a *API) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Profile, error) {
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Update", "clustering.profiles", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil && config.Options.Name != nil {
		err = validateName(*config.Options.Name)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	if err != nil {
		return nil, request.Wrap("Update", "clustering.profiles", err)
	}
	forbidden := append(append([]string(nil), responseFields...), "spec")
	body, err := senlin.Body(config, "profile", forbidden...)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.profiles", err)
	}
	headers := maps.Clone(config.Headers)
	id, err := rest.Collection(spec(a.RawClient())).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.profiles", err)
	}
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Update", "clustering.profiles", err)
	}
	response, err := rest.DoJSON(ctx, a.RawClient(), http.MethodPatch, a.RawClient().ServiceURL("profiles", url.PathEscape(id)), body, headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.profiles", err)
	}
	value, err := rest.Decode(response, "profile", func(value *Profile) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Update", "clustering.profiles", err)
}

func (a *API) Validate(ctx context.Context, opts ValidateOpts, options ...ValidateOption) (*Profile, error) {
	if err := senlin.RequireVersion(ctx, a.RawClient(), 2); err != nil {
		return nil, request.Wrap("Validate", "clustering.profiles", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = senlin.Object(config.Options.Spec, "spec")
	}
	if err != nil {
		return nil, request.Wrap("Validate", "clustering.profiles", err)
	}
	body, err := senlin.Body(config, "profile", responseFields...)
	if err != nil {
		return nil, request.Wrap("Validate", "clustering.profiles", err)
	}
	if err := senlin.RequireVersion(ctx, a.RawClient(), 2); err != nil {
		return nil, request.Wrap("Validate", "clustering.profiles", err)
	}
	response, err := rest.DoJSON(ctx, a.RawClient(), http.MethodPost, a.RawClient().ServiceURL("profiles", "validate"), body, config.Headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Validate", "clustering.profiles", err)
	}
	value, err := rest.Decode(response, "profile", func(value *Profile) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Validate", "clustering.profiles", err)
}
