package containers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

type preparedMetadata struct {
	api                                         *API
	source, client                              *gophercloud.ServiceClient
	provider                                    *gophercloud.ProviderClient
	endpoint, base, target, name, kind, version string
}

func (a *API) captureMetadata(ctx context.Context, container string) (*preparedMetadata, error) {
	var source *gophercloud.ServiceClient
	if a != nil {
		source = a.client
	}
	headers, err := validateMetadataSource(ctx, source)
	if err != nil {
		return nil, err
	}
	if err := validateMetadataContainer(container); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	target := source.ServiceURL(url.PathEscape(container))
	if err := rest.ValidateTarget(source, target); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	client := *source
	client.MoreHeaders = headers
	return &preparedMetadata{api: a, source: source, client: &client, provider: source.ProviderClient, endpoint: source.Endpoint, base: source.ResourceBase, target: target, name: container, kind: source.Type, version: source.Microversion}, nil
}
func (p *preparedMetadata) check(ctx context.Context) error {
	if p.api.client != p.source || p.source.ProviderClient != p.provider || p.source.Endpoint != p.endpoint || p.source.ResourceBase != p.base || p.source.ServiceURL(url.PathEscape(p.name)) != p.target || p.source.Type != p.kind || p.source.Microversion != p.version {
		return metadataContextError(ctx, metadataInvalid("container metadata source or target changed"))
	}
	_, err := validateMetadataSource(ctx, p.source)
	return err
}
func (p *preparedMetadata) finish(ctx context.Context, headers map[string]string, err error) error {
	err = joinMetadataErrors(err, p.check(ctx))
	if err != nil {
		return metadataContextError(ctx, err)
	}
	owned, err := validateMetadataHeaders(headers)
	if err != nil {
		return metadataContextError(ctx, err)
	}
	for key, value := range owned {
		p.client.MoreHeaders[key] = value
	}
	return nil
}
func (p *preparedMetadata) observe(ctx context.Context, response *rest.Response, err error) error {
	checkErr := p.check(ctx)
	if checkErr == nil {
		return err
	}
	err = joinMetadataErrors(err, checkErr)
	if response != nil {
		return response.Fail(err)
	}
	return err
}
func validateMetadataSource(ctx context.Context, source *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, metadataInvalid("context is required")
	}
	if ctx.Err() != nil {
		return nil, metadataContextError(ctx, ctx.Err())
	}
	if source == nil || source.ProviderClient == nil {
		return nil, metadataInvalid("container service client is required")
	}
	if source.Type != "" && source.Type != "object-store" {
		return nil, fmt.Errorf("%w: object-store service client is required", resource.ErrUnsupported)
	}
	if !utf8.ValidString(source.Endpoint) || !metadataFieldValue(source.Microversion) {
		return nil, metadataInvalid("invalid container endpoint or microversion")
	}
	base := source.ResourceBaseURL()
	for _, endpoint := range []string{source.Endpoint, base} {
		parsed, err := url.Parse(endpoint)
		if err != nil || !utf8.ValidString(endpoint) || parsed.User != nil || parsed.Host == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, metadataInvalid("invalid container endpoint or resource base")
		}
	}
	if !strings.HasSuffix(base, "/") {
		return nil, metadataInvalid("container resource base must end with a slash")
	}
	if err := rest.ValidateTarget(source, base); err != nil {
		return nil, err
	}
	return validateMetadataHeaders(source.MoreHeaders)
}
func validateMetadataContainer(name string) error {
	if err := swift.CheckContainerName(name); err != nil {
		return fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	if !utf8.ValidString(name) || name == "." || name == ".." || strings.ContainsRune(name, '\\') {
		return metadataInvalid("invalid literal container name")
	}
	for _, b := range []byte(name) {
		if b < 32 || b == 127 {
			return metadataInvalid("invalid control in container name")
		}
	}
	return nil
}
func metadataToken(value string) bool {
	if value == "" {
		return false
	}
	for _, b := range []byte(value) {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
			continue
		}
		return false
	}
	return true
}
func metadataFieldValue(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, b := range []byte(value) {
		if b < 32 && b != '\t' || b == 127 {
			return false
		}
	}
	return true
}
func metadataReserved(key string) bool {
	name := strings.ToLower(key)
	if strings.HasPrefix(name, "x-container-meta-") || strings.HasPrefix(name, "x-remove-container-meta-") {
		return true
	}
	switch name {
	case "authorization", "x-auth-token", "host", "content-length", "transfer-encoding", "connection", "proxy-connection", "proxy-authorization", "upgrade", "trailer", "te", "x-newest":
		return true
	}
	return false
}
func validateMetadataHeaders(headers map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		if !metadataToken(key) || !metadataFieldValue(value) {
			return nil, metadataInvalid("invalid container header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if _, exists := result[name]; exists {
			return nil, metadataInvalid("container header aliases %q", key)
		}
		if metadataReserved(key) {
			return nil, metadataInvalid("container header %q is SDK owned", key)
		}
		result[name] = value
	}
	return result, nil
}
func mergeMetadataHeaders(dst, values map[string]string) error {
	owned, err := validateMetadataHeaders(values)
	if err != nil {
		return err
	}
	for key, value := range owned {
		for previous := range dst {
			if metadataToken(previous) && http.CanonicalHeaderKey(previous) == key {
				delete(dst, previous)
			}
		}
		dst[key] = value
	}
	return nil
}
func metadataSuffix(key string) error {
	if !metadataToken(key) || strings.HasPrefix(strings.ToLower(key), "x-container-meta-") {
		return metadataInvalid("metadata key must be a suffix-only HTTP token")
	}
	return nil
}
func metadataInput(input map[string]string) func() (map[string]string, error) {
	snapshot := cloneMetadataHeaders(input)
	return func() (map[string]string, error) {
		result := make(map[string]string, len(snapshot))
		seen := make(map[string]bool)
		for key, value := range snapshot {
			if err := metadataSuffix(key); err != nil {
				return nil, err
			}
			name := strings.ToLower(key)
			if seen[name] {
				return nil, metadataInvalid("metadata key aliases %q", key)
			}
			seen[name] = true
			if !metadataFieldValue(value) {
				return nil, metadataInvalid("invalid metadata value for %q", key)
			}
			result[http.CanonicalHeaderKey("X-Container-Meta-"+key)] = value
		}
		return result, nil
	}
}
func metadataKeys(keys []string) func() (map[string]string, error) {
	return func() (map[string]string, error) {
		result := make(map[string]string, len(keys))
		seen := make(map[string]bool)
		for _, key := range keys {
			if err := metadataSuffix(key); err != nil {
				return nil, err
			}
			name := strings.ToLower(key)
			if seen[name] {
				return nil, metadataInvalid("duplicate metadata key %q", key)
			}
			seen[name] = true
			result[http.CanonicalHeaderKey("X-Container-Meta-"+key)] = ""
		}
		return result, nil
	}
}
func metadataInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func joinMetadataErrors(values ...error) error {
	var result []error
	for _, value := range values {
		if value != nil {
			result = append(result, value)
		}
	}
	if len(result) == 1 {
		return result[0]
	}
	return errors.Join(result...)
}
func metadataContextError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = joinMetadataErrors(err, cause)
			}
		}
	}
	return err
}
