// Package cloudread owns selected service facts across SDK read workflows.
package cloudread

import (
	"context"
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Source struct {
	Client                              gophercloud.ServiceClient
	original                            *gophercloud.ServiceClient
	provider                            *gophercloud.ProviderClient
	endpoint, base, kind, version, role string
	observed                            error
}

func Capture(ctx context.Context, client *gophercloud.ServiceClient, role string) (*Source, error) {
	headers, err := validateSource(ctx, client, role)
	if err != nil {
		return nil, err
	}
	copy := *client
	copy.MoreHeaders = maps.Clone(headers)
	if copy.Type == "" {
		if role == "volume" {
			copy.Type = "volumev3"
		} else {
			copy.Type = role
		}
	}
	return &Source{Client: copy, original: client, provider: client.ProviderClient, endpoint: client.Endpoint, base: client.ResourceBase, kind: client.Type, version: client.Microversion, role: role}, nil
}

func (s *Source) Guard(ctx context.Context) error {
	if s.observed != nil {
		return ContextError(ctx, s.observed)
	}
	if s.original.ProviderClient != s.provider || s.original.Endpoint != s.endpoint || s.original.ResourceBase != s.base || s.original.Type != s.kind || s.original.Microversion != s.version {
		s.observed = invalid("%s read source or route changed", s.role)
	} else {
		_, s.observed = validateSourceFacts(s.original, s.role)
	}
	return ContextError(ctx, s.observed)
}

func (s *Source) Get(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
	return rest.DoJSONGuarded(ctx, &s.Client, s.Guard, http.MethodGet, target, nil, nil, codes...)
}

func Context(ctx context.Context) error {
	if ctx == nil {
		return invalid("context is required")
	}
	return ContextError(ctx, ctx.Err())
}

func ContextError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return err
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func validText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func validateSource(ctx context.Context, source *gophercloud.ServiceClient, role string) (map[string]string, error) {
	if err := Context(ctx); err != nil {
		return nil, err
	}
	return validateSourceFacts(source, role)
}

func validateSourceFacts(source *gophercloud.ServiceClient, role string) (map[string]string, error) {
	if source == nil || source.ProviderClient == nil {
		return nil, invalid("%s service client is required", role)
	}
	allowed := source.Type == ""
	switch role {
	case "compute":
		allowed = allowed || source.Type == "compute"
	case "identity":
		allowed = allowed || source.Type == "identity"
	case "volume":
		allowed = allowed || source.Type == "block-storage" || source.Type == "block-store" || source.Type == "volume" || source.Type == "volumev3"
	default:
		return nil, invalid("unknown read service role")
	}
	if !allowed {
		return nil, fmt.Errorf("%w: %s read service client is required", resource.ErrUnsupported, role)
	}
	if !validText(source.Microversion) {
		return nil, invalid("invalid read service microversion")
	}
	for index, endpoint := range []string{source.Endpoint, source.ResourceBaseURL()} {
		if err := validateEndpoint(endpoint, index == 1); err != nil {
			return nil, err
		}
	}
	if err := rest.ValidateTarget(source, source.ResourceBaseURL()); err != nil {
		return nil, err
	}
	return canonicalHeaders(source.MoreHeaders, role, source.Microversion)
}

func validateEndpoint(endpoint string, requireSlash bool) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || !validText(endpoint) || parsed.User != nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || !validText(parsed.Path) || requireSlash && !strings.HasSuffix(endpoint, "/") {
		return invalid("read endpoint or resource base must be an absolute HTTP(S) URL without credentials, query, fragment or controls; effective base must end in a slash")
	}
	return nil
}

func validateTarget(source *gophercloud.ServiceClient, target string) error {
	if err := rest.ValidateTarget(source, target); err != nil {
		return err
	}
	parsed, err := url.Parse(target)
	if err != nil || !validText(target) || parsed.RawQuery != "" || parsed.ForceQuery || !validText(parsed.Path) {
		return invalid("invalid fixed read request target")
	}
	return nil
}

func canonicalHeaders(values map[string]string, role, version string) (map[string]string, error) {
	headers := make(map[string]string, len(values))
	for key, value := range values {
		if !headerToken(key) || !headerValue(value) {
			return nil, invalid("invalid read service header")
		}
		name := http.CanonicalHeaderKey(key)
		if previous, exists := headers[name]; exists && previous != value {
			return nil, invalid("conflicting read header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade":
			return nil, invalid("read service header %q is SDK owned", key)
		case "openstack-api-version":
			if version == "" || value != role+" "+version {
				return nil, invalid("read service version header conflicts with selected microversion")
			}
		case "x-openstack-nova-api-version":
			if role != "compute" || version == "" || value != version {
				return nil, invalid("Nova version header conflicts with read service or microversion")
			}
		case "x-openstack-volume-api-version":
			if role != "volume" || version == "" || value != version {
				return nil, invalid("Cinder version header conflicts with read service or microversion")
			}
		}
		headers[name] = value
	}
	return headers, nil
}

func headerToken(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range []byte(value) {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(character)) {
			continue
		}
		return false
	}
	return true
}

func headerValue(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) && character != '\t' {
			return false
		}
	}
	return true
}
