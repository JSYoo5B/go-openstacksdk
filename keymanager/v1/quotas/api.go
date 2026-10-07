package quotas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// Get reads effective quotas for the provider token's current project.
func (a *API) Get(ctx context.Context, options ...GetOption) (*Quota, error) {
	client := a.RawClient()
	if err := validate(ctx, client); err != nil {
		return nil, request.Wrap("Get", "quotas", err)
	}
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = validateConfig(config)
	}
	if err != nil {
		return nil, request.Wrap("Get", "quotas", err)
	}
	return get(ctx, client, client.ServiceURL("quotas"), "quotas", config.Headers)
}

// ProjectScope owns one project ID independently of response fields and the
// provider's current project. Constructing an explicit ID scope performs no HTTP.
type ProjectScope struct {
	client    *gophercloud.ServiceClient
	projectID string
}

func (a *API) InProject(ctx context.Context, ref resource.Ref) (*ProjectScope, error) {
	if err := validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("InProject", "project_quotas", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InProject", "project_quotas", err)
	}
	if ref.IsName() {
		return nil, request.Wrap("InProject", "project_quotas", fmt.Errorf("%w: quota scopes require an explicit project ID", resource.ErrUnsupported))
	}
	if err := projectID(ref.String()); err != nil {
		return nil, request.Wrap("InProject", "project_quotas", err)
	}
	return &ProjectScope{client: a.client, projectID: ref.String()}, nil
}

func (s *ProjectScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}
func (s *ProjectScope) RawClient() *gophercloud.ServiceClient {
	if s == nil {
		return nil
	}
	return s.client
}
func (s *ProjectScope) validate(ctx context.Context) error {
	if err := validate(ctx, s.RawClient()); err != nil {
		return err
	}
	return projectID(s.projectID)
}

func (s *ProjectScope) Get(ctx context.Context, options ...GetOption) (*Quota, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Get", "project_quotas", err)
	}
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = validateConfig(config)
	}
	if err != nil {
		return nil, request.Wrap("Get", "project_quotas", err)
	}
	return get(ctx, s.client, s.client.ServiceURL("project-quotas", url.PathEscape(s.projectID)), "project_quotas", config.Headers)
}

// Update replaces all configured overrides. It never reads current values to
// fill omitted fields and returns only the actual accepted HTTP response.
func (s *ProjectScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*UpdateResult, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Update", "project_quotas", err)
	}
	config, err := request.Apply(UpdateOpts{}, append([]UpdateOption{snapshot(opts)}, options...)...)
	if err == nil {
		err = validateConfig(config)
	}
	if err == nil {
		for _, field := range []request.Optional[int64]{config.Options.Secrets, config.Options.Orders, config.Options.Containers, config.Options.Consumers, config.Options.CAs} {
			if field.IsNull() {
				err = fmt.Errorf("%w: quota update fields require integers, not null", resource.ErrInvalidOption)
				break
			}
		}
	}
	if err != nil {
		return nil, request.Wrap("Update", "project_quotas", err)
	}
	body, err := json.Marshal(map[string]UpdateOpts{"project_quotas": config.Options})
	if err != nil {
		return nil, request.Wrap("Update", "project_quotas", err)
	}
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Update", "project_quotas", err)
	}
	response, err := rest.DoJSON(ctx, s.client, "PUT", s.client.ServiceURL("project-quotas", url.PathEscape(s.projectID)), json.RawMessage(body), maps.Clone(config.Headers), 204)
	if err != nil {
		return nil, request.Wrap("Update", "project_quotas", err)
	}
	return &UpdateResult{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

func (s *ProjectScope) Delete(ctx context.Context, options ...DeleteOption) error {
	if err := s.validate(ctx); err != nil {
		return request.Wrap("Delete", "project_quotas", err)
	}
	config, err := request.Apply(DeleteOpts{}, options...)
	if err == nil {
		err = validateConfig(config)
	}
	if err != nil {
		return request.Wrap("Delete", "project_quotas", err)
	}
	ignoreMissing := config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing
	if err := s.validate(ctx); err != nil {
		return request.Wrap("Delete", "project_quotas", err)
	}
	_, err = rest.DoJSON(ctx, s.client, "DELETE", s.client.ServiceURL("project-quotas", url.PathEscape(s.projectID)), nil, maps.Clone(config.Headers), 204)
	var accepted *resource.ResponseError
	var transport *url.Error
	if ignoreMissing && !errors.As(err, &accepted) && !errors.As(err, &transport) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && gophercloud.ResponseCodeIs(err, 404) {
		return nil
	}
	return request.Wrap("Delete", "project_quotas", err)
}

func get(ctx context.Context, client *gophercloud.ServiceClient, endpoint, envelope string, headers map[string]string) (*Quota, error) {
	if err := validate(ctx, client); err != nil {
		return nil, request.Wrap("Get", envelope, err)
	}
	response, err := rest.DoJSON(ctx, client, "GET", endpoint, nil, maps.Clone(headers), 200)
	if err != nil {
		return nil, request.Wrap("Get", envelope, err)
	}
	value, err := rest.Decode(response, envelope, func(value *Quota) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Get", envelope, err)
}

func validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if err := cloudread.Context(ctx); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: key-manager service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "key-manager" {
		return fmt.Errorf("%w: service client type must be key-manager", resource.ErrInvalidOption)
	}
	return validateHeaders(client.MoreHeaders)
}

func projectID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return fmt.Errorf("%w: project ID must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, r := range id {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%w: project ID cannot contain whitespace/control characters", resource.ErrInvalidOption)
		}
	}
	return nil
}

func validateConfig[T any](config request.Config[T]) error {
	if err := request.ValidateCapabilities(config, false, false, true); err != nil {
		return err
	}
	return validateHeaders(config.Headers)
}

func validateHeaders(headers map[string]string) error {
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid quota header", resource.ErrInvalidOption)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "openstack-api-version":
			return fmt.Errorf("%w: header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}
