package metadefnamespaces

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedSource struct {
	api                          *API
	source, client               *gophercloud.ServiceClient
	provider                     *gophercloud.ProviderClient
	base, endpoint, microversion string
}

func (a *API) capture(ctx context.Context) (*preparedSource, error) {
	source := a.RawClient()
	headers, err := validateSource(ctx, source)
	if err != nil {
		return nil, err
	}
	client := *source
	client.MoreHeaders = headers
	return &preparedSource{api: a, source: source, client: &client, provider: source.ProviderClient, base: source.ServiceURL(), endpoint: source.Endpoint, microversion: source.Microversion}, nil
}

func (p *preparedSource) check(ctx context.Context) error {
	if p.api.RawClient() != p.source || p.source.ProviderClient != p.provider || p.source.Type != "image" || p.source.Endpoint != p.endpoint || p.source.ServiceURL() != p.base || p.source.Microversion != p.microversion {
		return invalid("namespace source or service target changed")
	}
	_, err := validateSource(ctx, p.source)
	return err
}

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

func (p *preparedSource) url(namespace *string) string {
	endpoint := p.base + "metadefs/namespaces"
	if namespace != nil {
		endpoint += "/" + url.PathEscape(*namespace)
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
			return nil, invalid("invalid namespace header")
		}
		for _, char := range []byte(key) {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
				continue
			}
			return nil, invalid("invalid namespace header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := result[name]; exists && old != value {
			return nil, invalid("conflicting namespace header aliases %q", key)
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
		return invalid("namespace must be a nonempty safe literal name")
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

func scalars(display, description, visibility, owner *string) error {
	for _, field := range []struct {
		value *string
		max   int
	}{{display, 80}, {description, 500}, {owner, 255}} {
		if field.value != nil {
			if err := text(*field.value, field.max); err != nil {
				return err
			}
		}
	}
	if visibility != nil && *visibility != "public" && *visibility != "private" {
		return invalid("visibility must be public or private")
	}
	return nil
}

func scalarBody(namespace string, display, description, visibility, owner *string, protected *bool) map[string]any {
	body := map[string]any{"namespace": namespace}
	for _, field := range []struct {
		key   string
		value *string
	}{{"display_name", display}, {"description", description}, {"visibility", visibility}, {"owner", owner}} {
		if field.value != nil {
			body[field.key] = *field.value
		}
	}
	if protected != nil {
		body["protected"] = *protected
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
