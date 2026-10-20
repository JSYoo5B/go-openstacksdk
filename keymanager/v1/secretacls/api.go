package secretacls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "keymanager.secret_acl"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// Opts carries only Go extension headers; the Python helpers take no others.
type Opts struct{}
type Option = request.Option[Opts]

func WithHeader(key, value string) Option { return request.WithHeader[Opts](key, value) }

// DeleteOpts selects Python delete_secret_acl's ignore_missing (nil is true).
type DeleteOpts struct{ IgnoreMissing *bool }
type DeleteOption = request.Option[DeleteOpts]

func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(config *request.Config[DeleteOpts]) error {
		owned := value
		config.Options.IgnoreMissing = &owned
		return nil
	}
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}

// SecretScope fixes the parent secret once; Python passes Resource._get_id.
type SecretScope struct {
	client   *gophercloud.ServiceClient
	secretID string
}

func (s *SecretScope) SecretID() string {
	if s == nil {
		return ""
	}
	return s.secretID
}

// InSecret accepts a safe explicit ID or resolves an explicit Name once
// through the existing Secrets collection. Name lookup is a Go convenience.
func (a *API) InSecret(ctx context.Context, ref resource.Ref) (*SecretScope, error) {
	client := a.RawClient()
	if err := validate(ctx, client); err != nil {
		return nil, request.Wrap("InSecret", kind, err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("InSecret", kind, err)
	}
	id, err := secrets.New(client).Resources.ResolveID(ctx, ref)
	if err == nil {
		err = errors.Join(validate(ctx, client), secretID(id))
	}
	if err != nil {
		return nil, request.Wrap("InSecret", kind, err)
	}
	return &SecretScope{client: client, secretID: id}, nil
}

func validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: key-manager service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "key-manager" {
		return fmt.Errorf("%w: expected key-manager service client", resource.ErrUnsupported)
	}
	return validateHeaders(client.MoreHeaders)
}
func secretID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return fmt.Errorf("%w: secret ID must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("%w: invalid secret ID", resource.ErrInvalidOption)
		}
	}
	return nil
}
func validateHeaders(headers map[string]string) error {
	seen := make(map[string]string, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid secret ACL header", resource.ErrInvalidOption)
		}
		canonical := strings.ToLower(key)
		if previous, exists := seen[canonical]; exists && previous != value {
			return fmt.Errorf("%w: conflicting case variants of header %q", resource.ErrInvalidOption, key)
		}
		seen[canonical] = value
		switch canonical {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "openstack-api-version":
			return fmt.Errorf("%w: header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}
func (s *SecretScope) validate(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("%w: secret scope is required", resource.ErrInvalidOption)
	}
	return errors.Join(validate(ctx, s.client), secretID(s.secretID))
}
func (s *SecretScope) endpoint() string {
	return s.client.ServiceURL("secrets", url.PathEscape(s.secretID), "acl")
}

// pythonCodes is Resource raise_from_response: every status below 400.
func pythonCodes() []int {
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	return codes
}

func (s *SecretScope) seed(response *rest.Response) *SecretACL {
	value := &SecretACL{SecretID: s.secretID, Read: json.RawMessage("null"), ACLRef: json.RawMessage("null")}
	if response != nil {
		value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	}
	return value
}

// Get implements get_secret_acl: Resource.fetch of secrets/{id}/acl.
func (s *SecretScope) Get(ctx context.Context, options ...Option) (*SecretACL, error) {
	config, err := request.Apply(Opts{}, options...)
	if err == nil {
		err = errors.Join(validateHeaders(config.Headers), s.validate(ctx))
	}
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	response, err := rest.DoJSON(ctx, s.client, http.MethodGet, s.endpoint(), nil, maps.Clone(config.Headers), pythonCodes()...)
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	value := s.seed(response)
	if err := value.overlay(response.Body); err != nil {
		return nil, request.Wrap("Get", kind, response.Fail(err))
	}
	return value, nil
}

// Set implements set_secret_acl and Update implements update_secret_acl.
// Both are Python _update/Resource.commit and therefore send PUT. With no
// declared attrs the Resource has nothing dirty and commit sends no request.
func (s *SecretScope) Set(ctx context.Context, input ACLInput, options ...Option) (*SecretACL, error) {
	return s.commit(ctx, "Set", input, options)
}
func (s *SecretScope) Update(ctx context.Context, input ACLInput, options ...Option) (*SecretACL, error) {
	return s.commit(ctx, "Update", input, options)
}
func (s *SecretScope) commit(ctx context.Context, operation string, input ACLInput, options []Option) (*SecretACL, error) {
	config, err := request.Apply(Opts{}, options...)
	var body map[string]json.RawMessage
	if err == nil {
		body, err = input.body()
	}
	if err == nil {
		err = errors.Join(validateHeaders(config.Headers), s.validate(ctx))
	}
	if err != nil {
		return nil, request.Wrap(operation, kind, err)
	}
	value := s.seed(nil)
	if read, present := body["read"]; present {
		value.Read = read
	}
	if ref, present := body["acl_ref"]; present {
		value.ACLRef = ref
	}
	if len(body) == 0 {
		return value, nil
	}
	response, err := rest.DoJSON(ctx, s.client, http.MethodPut, s.endpoint(), body, maps.Clone(config.Headers), pythonCodes()...)
	if err != nil {
		return nil, request.Wrap(operation, kind, err)
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	if err := value.overlay(response.Body); err != nil {
		return nil, request.Wrap(operation, kind, response.Fail(err))
	}
	return value, nil
}

// Delete implements delete_secret_acl. A clean native 404 returns nil,nil
// unless IgnoreMissing is false; otherwise the seeded ACL keeps the response.
func (s *SecretScope) Delete(ctx context.Context, options ...DeleteOption) (*SecretACL, error) {
	config, err := request.Apply(DeleteOpts{}, options...)
	if err == nil {
		err = errors.Join(validateHeaders(config.Headers), s.validate(ctx))
	}
	if err != nil {
		return nil, request.Wrap("Delete", kind, err)
	}
	ignoreMissing := config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing
	response, err := rest.DoJSON(ctx, s.client, http.MethodDelete, s.endpoint(), nil, maps.Clone(config.Headers), pythonCodes()...)
	if err != nil && ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	var transport *url.Error
	var accepted *resource.ResponseError
	if ignoreMissing && !errors.As(err, &transport) && !errors.As(err, &accepted) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, request.Wrap("Delete", kind, err)
	}
	return s.seed(response), nil
}
