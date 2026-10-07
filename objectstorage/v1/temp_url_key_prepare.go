package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

type preparedTempURLKey struct {
	service                       *Service
	source, client                *gophercloud.ServiceClient
	provider                      *gophercloud.ProviderClient
	endpoint, base, kind, version string
	apiFieldsMatch                func() bool
}

func (s *Service) captureTempURLKey(ctx context.Context) (*preparedTempURLKey, error) {
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateTempURLKeySource(ctx, source)
	if err != nil {
		return nil, err
	}
	client, fields := *source, *s
	client.MoreHeaders = headers
	return &preparedTempURLKey{
		service: s, source: source, client: &client, provider: source.ProviderClient,
		endpoint: source.Endpoint, base: source.ResourceBase, kind: source.Type, version: source.Microversion,
		apiFieldsMatch: func() bool {
			return s.Accounts == fields.Accounts && s.Containers == fields.Containers && s.Objects == fields.Objects && s.Swauth == fields.Swauth
		},
	}, nil
}
func (p *preparedTempURLKey) check(ctx context.Context) error {
	if p.service.client != p.source || p.source.ProviderClient != p.provider || p.source.Endpoint != p.endpoint || p.source.ResourceBase != p.base || p.source.Type != p.kind || p.source.Microversion != p.version || !p.apiFieldsMatch() {
		return tempURLKeyContextError(ctx, tempURLKeyInvalid("Temp URL key source or API identity changed"))
	}
	_, err := validateTempURLKeySource(ctx, p.source)
	return err
}
func (p *preparedTempURLKey) finish(ctx context.Context, cfg GetTempURLKeyOpts, err error) error {
	err = joinTempURLKeyErrors(err, p.check(ctx))
	if err != nil {
		return tempURLKeyContextError(ctx, err)
	}
	headers, err := validateTempURLKeyHeaders(cfg.Headers)
	if err != nil {
		return tempURLKeyContextError(ctx, err)
	}
	if cfg.Container != "" {
		if err := validateTempURLKeyContainer(p.client, cfg.Container); err != nil {
			return tempURLKeyContextError(ctx, err)
		}
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	return nil
}
func (p *preparedTempURLKey) observe(ctx context.Context, response *rest.Response, err error) (error, bool) {
	guardErr := p.check(ctx)
	if guardErr == nil {
		return err, false
	}
	err = joinTempURLKeyErrors(err, guardErr)
	if response != nil {
		err = response.Fail(err)
	}
	return err, true
}

func validateTempURLKeySource(ctx context.Context, source *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, tempURLKeyInvalid("context is required")
	}
	if ctx.Err() != nil {
		return nil, tempURLKeyContextError(ctx, ctx.Err())
	}
	if source == nil || source.ProviderClient == nil {
		return nil, tempURLKeyInvalid("object-store service client is required")
	}
	if source.Type != "" && source.Type != "object-store" {
		return nil, fmt.Errorf("%w: object-store service client is required", resource.ErrUnsupported)
	}
	if !utf8.ValidString(source.Endpoint) || !tempURLKeyFieldValue(source.Microversion) {
		return nil, tempURLKeyInvalid("invalid endpoint or microversion")
	}
	if err := rest.ValidateTarget(source, source.Endpoint); err != nil {
		return nil, err
	}
	return validateTempURLKeyHeaders(source.MoreHeaders)
}
func validateTempURLKeyContainer(source *gophercloud.ServiceClient, name string) error {
	if err := swift.CheckContainerName(name); err != nil {
		return fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	if !utf8.ValidString(name) || name == "." || name == ".." || strings.ContainsRune(name, '\\') {
		return tempURLKeyInvalid("invalid literal container name")
	}
	for _, b := range []byte(name) {
		if b < 32 || b == 127 {
			return tempURLKeyInvalid("invalid control in container name")
		}
	}
	base := source.ResourceBaseURL()
	parsed, err := url.Parse(base)
	if err != nil || !utf8.ValidString(base) || parsed.User != nil || parsed.Host == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || !strings.HasSuffix(base, "/") {
		return tempURLKeyInvalid("invalid container resource base")
	}
	if err := rest.ValidateTarget(source, base); err != nil {
		return err
	}
	return rest.ValidateTarget(source, source.ServiceURL(url.PathEscape(name)))
}
func tempURLKeyToken(value string) bool {
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
func tempURLKeyFieldValue(value string) bool {
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
func tempURLKeyReserved(key string) bool {
	name := strings.ToLower(key)
	for _, prefix := range []string{"x-account-meta-", "x-remove-account-meta-", "x-container-meta-", "x-remove-container-meta-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	switch name {
	case "authorization", "x-auth-token", "host", "content-length", "transfer-encoding", "connection", "proxy-connection", "proxy-authorization", "upgrade", "trailer", "te", "x-newest":
		return true
	}
	return false
}
func validateTempURLKeyHeaders(headers map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		if !tempURLKeyToken(key) || !tempURLKeyFieldValue(value) {
			return nil, tempURLKeyInvalid("invalid Temp URL key header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if _, exists := result[name]; exists {
			return nil, tempURLKeyInvalid("Temp URL key header aliases %q", key)
		}
		if tempURLKeyReserved(key) {
			return nil, tempURLKeyInvalid("Temp URL key header %q is SDK owned", key)
		}
		result[name] = value
	}
	return result, nil
}
func mergeTempURLKeyHeaders(dst, values map[string]string) error {
	owned, err := validateTempURLKeyHeaders(values)
	if err != nil {
		return err
	}
	for key, value := range owned {
		for previous := range dst {
			if tempURLKeyToken(previous) && http.CanonicalHeaderKey(previous) == key {
				delete(dst, previous)
			}
		}
		dst[key] = value
	}
	return nil
}
func tempURLKeyInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func joinTempURLKeyErrors(values ...error) error {
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
func tempURLKeyContextError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = joinTempURLKeyErrors(err, cause)
			}
		}
	}
	return err
}
