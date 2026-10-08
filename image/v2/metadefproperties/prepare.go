package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedSource struct {
	scope  *NamespaceScope
	client *gophercloud.ServiceClient
}

func (s *NamespaceScope) check(ctx context.Context) error {
	if s == nil || s.api == nil || s.source == nil {
		return invalid("namespace scope is required")
	}
	if s.api.RawClient() != s.source || s.source.ProviderClient != s.provider || s.source.Type != s.serviceType || s.source.Endpoint != s.endpoint || s.source.ServiceURL() != s.base || s.source.Microversion != s.microversion {
		return invalid("property scope source or service target changed")
	}
	_, err := validateSource(ctx, s.source)
	return err
}

func (s *NamespaceScope) capture(ctx context.Context, name *string) (*preparedSource, error) {
	if err := s.check(ctx); err != nil {
		return nil, err
	}
	if name != nil {
		if err := literal(*name); err != nil {
			return nil, err
		}
	}
	headers, err := validateSource(ctx, s.source)
	if err != nil {
		return nil, err
	}
	client := *s.source
	client.MoreHeaders = headers
	return &preparedSource{scope: s, client: &client}, nil
}

// operationGuard freezes the namespace and keeps any observed source or outer
// guard failure sticky across owned record fetch/list physical boundaries.
func (p *preparedSource) operationGuard(ctx context.Context) func(context.Context) error {
	namespace := p.scope.namespace
	outer := rest.OperationGuard(ctx)
	var observed error
	return func(checkCtx context.Context) error {
		if observed != nil {
			return observed
		}
		var parent, binding error
		if outer != nil {
			parent = outer(checkCtx)
		}
		if p.scope.namespace != namespace {
			binding = invalid("property namespace changed")
		}
		observed = errors.Join(p.check(checkCtx), parent, binding)
		return observed
	}
}

func (p *preparedSource) check(ctx context.Context) error { return p.scope.check(ctx) }
func (p *preparedSource) finish(ctx context.Context, headers map[string]string, err error) error {
	if checkErr := p.check(ctx); checkErr != nil {
		err = joinErrors(err, checkErr)
	}
	if err != nil {
		return err
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	return nil
}
func (p *preparedSource) url(name *string) string {
	endpoint := p.scope.base + "metadefs/namespaces/" + url.PathEscape(p.scope.namespace) + "/properties"
	if name != nil {
		endpoint += "/" + url.PathEscape(*name)
	}
	return endpoint
}

func validateSource(ctx context.Context, client *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || client.ProviderClient == nil {
		return nil, invalid("image service client is required")
	}
	if client.Type != "image" {
		return nil, fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if !utf8.ValidString(client.Endpoint) || !utf8.ValidString(base) {
		return nil, invalid("image target must be valid UTF-8")
	}
	if err := rest.ValidateTarget(client, base); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return nil, invalid("image service base must be query-free and end in a slash")
	}
	if !utf8.ValidString(client.Microversion) || !headerValue(client.Microversion) {
		return nil, invalid("invalid image microversion")
	}
	return validateHeaders(client.MoreHeaders, true, client.Microversion)
}

func validateHeaders(values map[string]string, source bool, version string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if key == "" || !utf8.ValidString(value) || !headerValue(value) {
			return nil, invalid("invalid property header")
		}
		for _, char := range []byte(key) {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
				continue
			}
			return nil, invalid("invalid property header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := result[name]; exists && old != value {
			return nil, invalid("conflicting property header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version", "x-openstack-image-size":
			return nil, invalid("header %q is owned by the SDK", key)
		case "accept", "content-type":
			if !source {
				return nil, invalid("header %q is owned by the SDK", key)
			}
		case "openstack-api-version":
			if !source || version == "" || value != "image "+version {
				return nil, invalid("image version header conflicts with selected microversion")
			}
		}
		result[name] = value
	}
	return result, nil
}

func headerValue(value string) bool {
	for _, ch := range []byte(value) {
		if ch < 32 && ch != '\t' || ch == 127 {
			return false
		}
	}
	return true
}

func text(value string, max int) error {
	if !utf8.ValidString(value) || max > 0 && utf8.RuneCountInString(value) > max {
		return invalid("text must be valid UTF-8 and within its rune limit")
	}
	return nil
}

func literal(value string) error {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, "/\\%?#") {
		return invalid("name must be a nonempty safe literal name")
	}
	if err := text(value, 80); err != nil {
		return err
	}
	return queryText(value)
}

func queryText(value string) error {
	if err := text(value, 0); err != nil {
		return err
	}
	for _, ch := range []byte(value) {
		if ch < 32 || ch == 127 {
			return invalid("query text must not contain ASCII controls")
		}
	}
	return nil
}

func payload(propertyType, title, description *string, attributes map[string]json.RawMessage) error {
	if propertyType == nil || title == nil {
		return invalid("type and title are required")
	}
	if *propertyType == "" {
		return invalid("type must be nonempty")
	}
	for _, value := range []*string{propertyType, title, description} {
		if value != nil {
			if err := text(*value, 0); err != nil {
				return err
			}
		}
	}
	for key, raw := range attributes {
		if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
			return invalid("attribute %q must be valid UTF-8 JSON", key)
		}
		switch key {
		case "name", "type", "title", "description", "self", "schema", "created_at", "updated_at", "namespace_name":
			return invalid("attribute %q is owned by the SDK", key)
		}
	}
	return nil
}
func propertyBody(name string, propertyType, title, description *string, attributes map[string]json.RawMessage) map[string]any {
	body := map[string]any{"name": name, "type": *propertyType, "title": *title}
	if description != nil {
		body["description"] = *description
	}
	for key, raw := range attributes {
		body[key] = raw
	}
	return body
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func joinErrors(causes ...error) error {
	var values []error
	for _, cause := range causes {
		if cause != nil {
			values = append(values, cause)
		}
	}
	if len(values) == 1 {
		return values[0]
	}
	return errors.Join(values...)
}
func wrap(ctx context.Context, operation string, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = joinErrors(err, cause)
			}
		}
	}
	return request.Wrap(operation, kind, err)
}
