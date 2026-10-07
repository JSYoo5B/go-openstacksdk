package serviceinfo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedSource struct {
	api      *API
	source   *gophercloud.ServiceClient
	client   *gophercloud.ServiceClient
	base     string
	provider *gophercloud.ProviderClient
}

func (a *API) capture(ctx context.Context) (*preparedSource, error) {
	source := a.RawClient()
	headers, err := validateSource(ctx, source)
	if err != nil {
		return nil, err
	}
	client := *source
	client.MoreHeaders = headers
	return &preparedSource{api: a, source: source, client: &client, base: source.ServiceURL(), provider: source.ProviderClient}, nil
}

func (prepared *preparedSource) check(ctx context.Context) error {
	if prepared.api.RawClient() != prepared.source || prepared.source.ProviderClient != prepared.provider {
		return infoInvalid("discovery source or provider changed")
	}
	if _, err := validateSource(ctx, prepared.source); err != nil {
		return err
	}
	return rest.ValidateTarget(prepared.source, prepared.base)
}

func (prepared *preparedSource) addHeaders(headers map[string]string) {
	for key, value := range headers {
		prepared.client.MoreHeaders[key] = value
	}
}

func validateSource(ctx context.Context, client *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, infoInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || client.ProviderClient == nil {
		return nil, infoInvalid("image service client is required")
	}
	if client.Type != "image" {
		return nil, fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if !utf8.ValidString(client.Endpoint) || !utf8.ValidString(base) {
		return nil, infoInvalid("image service target must be valid UTF-8")
	}
	if err := rest.ValidateTarget(client, base); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return nil, infoInvalid("image service base must be query-free and end in a slash")
	}
	if !utf8.ValidString(client.Microversion) || !headerValue(client.Microversion) {
		return nil, infoInvalid("invalid image microversion")
	}
	return infoHeaders(client.MoreHeaders, true, client.Microversion)
}

func infoHeaders(values map[string]string, source bool, version string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if key == "" || !utf8.ValidString(value) || !headerValue(value) {
			return nil, infoInvalid("invalid discovery header")
		}
		for _, char := range []byte(key) {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
				continue
			}
			return nil, infoInvalid("invalid discovery header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := result[name]; exists && old != value {
			return nil, infoInvalid("conflicting discovery header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version", "x-openstack-image-size":
			return nil, infoInvalid("header %q is owned by the SDK", key)
		case "accept", "content-type":
			if !source {
				return nil, infoInvalid("header %q is owned by the SDK", key)
			}
		case "openstack-api-version":
			if !source || version == "" || value != "image "+version {
				return nil, infoInvalid("image version header conflicts with selected microversion")
			}
		}
		result[name] = value
	}
	return result, nil
}

func headerValue(value string) bool {
	for _, char := range []byte(value) {
		if char < 32 && char != '\t' || char == 127 {
			return false
		}
	}
	return true
}

func infoInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
