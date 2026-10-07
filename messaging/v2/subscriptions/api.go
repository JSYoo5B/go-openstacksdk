package subscriptions

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

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "messaging.subscriptions"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type QueueScope struct {
	client *gophercloud.ServiceClient
	queue  string
}

func (a *API) InQueue(ctx context.Context, queue string) (*QueueScope, error) {
	if err := validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("InQueue", kind, err)
	}
	if err := segment(queue); err != nil {
		return nil, request.Wrap("InQueue", kind, err)
	}
	return &QueueScope{client: a.client, queue: queue}, nil
}
func (s *QueueScope) QueueName() string {
	if s == nil {
		return ""
	}
	return s.queue
}
func (s *QueueScope) RawClient() *gophercloud.ServiceClient {
	if s == nil {
		return nil
	}
	return s.client
}
func (s *QueueScope) validate(ctx context.Context) error {
	if err := validate(ctx, s.RawClient()); err != nil {
		return err
	}
	return segment(s.queue)
}
func (s *QueueScope) endpoint(id string) string {
	if id == "" {
		return s.client.ServiceURL("queues", url.PathEscape(s.queue), "subscriptions")
	}
	return s.client.ServiceURL("queues", url.PathEscape(s.queue), "subscriptions", url.PathEscape(id))
}
func validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: message service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "message" {
		return fmt.Errorf("%w: message service client type is required", resource.ErrUnsupported)
	}
	return nil
}
func segment(value string) error {
	if err := resource.ID(value).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: path segment must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%w: invalid path segment", resource.ErrInvalidOption)
		}
	}
	return nil
}
func validateHeaders(headers map[string]string, extension bool) error {
	for key, value := range headers {
		if key == "" {
			return fmt.Errorf("%w: empty header", resource.ErrInvalidOption)
		}
		for _, r := range key {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", r)) {
				return fmt.Errorf("%w: invalid header name", resource.ErrInvalidOption)
			}
		}
		for _, r := range value {
			if r < 32 && r != '\t' || r == 127 {
				return fmt.Errorf("%w: invalid header value", resource.ErrInvalidOption)
			}
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "openstack-api-version":
			return fmt.Errorf("%w: SDK-owned header %q", resource.ErrInvalidOption, key)
		case "client-id", "x-project-id":
			if extension {
				return fmt.Errorf("%w: use typed request identity for %q", resource.ErrInvalidOption, key)
			}
		}
	}
	return nil
}

// headers freezes the effective header map. Typed identity overrides configured
// values on a per-request clone, without changing the shared service client.
func (s *QueueScope) headers(identity RequestIdentity, extra map[string]string) (map[string]string, error) {
	if err := validateHeaders(s.client.MoreHeaders, false); err != nil {
		return nil, err
	}
	if err := validateHeaders(extra, true); err != nil {
		return nil, err
	}
	owned := make(map[string]string)
	for key, value := range s.client.MoreHeaders {
		canonical := http.CanonicalHeaderKey(key)
		if prior, present := owned[canonical]; present && prior != value {
			return nil, fmt.Errorf("%w: conflicting source header %q", resource.ErrInvalidOption, key)
		}
		owned[canonical] = value
	}
	extensions := make(map[string]string)
	for key, value := range extra {
		canonical := http.CanonicalHeaderKey(key)
		if prior, present := extensions[canonical]; present && prior != value {
			return nil, fmt.Errorf("%w: conflicting request header %q", resource.ErrInvalidOption, key)
		}
		extensions[canonical] = value
	}
	for key, value := range extensions {
		owned[key] = value
	}
	if identity.ClientID != "" {
		owned["Client-Id"] = identity.ClientID
	}
	if identity.ProjectID != "" {
		owned["X-Project-Id"] = identity.ProjectID
	}
	if err := validateHeaders(owned, false); err != nil {
		return nil, err
	}
	if strings.TrimSpace(owned["Client-Id"]) == "" {
		return nil, fmt.Errorf("%w: configure Client-ID or use a typed client ID option", resource.ErrInvalidOption)
	}
	return owned, nil
}
func (s *QueueScope) do(ctx context.Context, method, endpoint string, body any, headers map[string]string, code int) (*rest.Response, error) {
	if err := s.validate(ctx); err != nil {
		return nil, err
	}
	client := *s.client
	client.MoreHeaders = maps.Clone(headers)
	return rest.DoJSON(ctx, &client, method, endpoint, body, nil, code)
}
func (s *QueueScope) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Subscription, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	config, err := request.Apply(CreateOpts{}, append([]CreateOption{WithCreateOptions(opts)}, options...)...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	if err == nil && strings.TrimSpace(config.Options.Subscriber) == "" {
		err = fmt.Errorf("%w: subscriber is required", resource.ErrInvalidOption)
	}
	if err == nil && config.Options.TTL.IsNull() {
		err = fmt.Errorf("%w: ttl requires an integer", resource.ErrInvalidOption)
	}
	if err == nil && len(config.Options.Options) > 0 {
		var object map[string]json.RawMessage
		if decode := json.Unmarshal(config.Options.Options, &object); decode != nil || object == nil {
			err = fmt.Errorf("%w: options must be a JSON object", resource.ErrInvalidOption)
		}
	}
	var body map[string]json.RawMessage
	if err == nil {
		for key := range config.Fields {
			switch strings.ToLower(key) {
			case "subscriber", "ttl", "options", "id", "subscription_id", "source", "age", "queue_name", "client_id", "project_id":
				err = fmt.Errorf("%w: field %q is owned by the SDK", resource.ErrInvalidOption, key)
			}
		}
	}
	if err == nil {
		encoded, marshal := json.Marshal(config.Options)
		err = marshal
		if err == nil {
			err = json.Unmarshal(encoded, &body)
		}
		if err == nil {
			for key, raw := range config.Fields {
				body[key] = append(json.RawMessage(nil), raw...)
			}
		}
	}
	var headers map[string]string
	if err == nil {
		headers, err = s.headers(config.Options.RequestIdentity, config.Headers)
	}
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	response, err := s.do(ctx, "POST", s.endpoint(""), body, headers, 201)
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	value, err := rest.Decode(response, "", func(value *Subscription) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Create", kind, err)
}
func (s *QueueScope) Get(ctx context.Context, id string, options ...GetOption) (*Subscription, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	if err := segment(id); err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	config, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	var headers map[string]string
	if err == nil {
		headers, err = s.headers(config.Options.RequestIdentity, config.Headers)
	}
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	response, err := s.do(ctx, "GET", s.endpoint(id), nil, headers, 200)
	if err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	value, err := rest.Decode(response, "", func(value *Subscription) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap("Get", kind, err)
}
func (s *QueueScope) Delete(ctx context.Context, id string, options ...DeleteOption) error {
	if err := s.validate(ctx); err != nil {
		return request.Wrap("Delete", kind, err)
	}
	if err := segment(id); err != nil {
		return request.Wrap("Delete", kind, err)
	}
	config, err := request.Apply(DeleteOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	var headers map[string]string
	if err == nil {
		headers, err = s.headers(config.Options.RequestIdentity, config.Headers)
	}
	if err != nil {
		return request.Wrap("Delete", kind, err)
	}
	ignore := config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing
	_, err = s.do(ctx, "DELETE", s.endpoint(id), nil, headers, 204)
	if err != nil && ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	var accepted *resource.ResponseError
	var transport *url.Error
	if ignore && !errors.As(err, &accepted) && !errors.As(err, &transport) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && gophercloud.ResponseCodeIs(err, 404) {
		return nil
	}
	return request.Wrap("Delete", kind, err)
}
