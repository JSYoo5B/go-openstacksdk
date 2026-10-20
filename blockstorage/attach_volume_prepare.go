package blockstorage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	volumes "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	servers "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Each service keeps its selected route and request settings while its original
// provider continues supplying current authentication to physical requests.
type attachSource struct {
	source               *gophercloud.ServiceClient
	client               gophercloud.ServiceClient
	provider             *gophercloud.ProviderClient
	endpoint, base, kind string
	version, role        string
}

type preparedAttach struct {
	nova, cinder             attachSource
	serverID, volumeID       string
	volumeURL, attachmentURL string
	options                  preparedAttachOptions
	observed                 error
}

func captureAttach(ctx context.Context, nova, cinder *gophercloud.ServiceClient, input AttachVolumeRequest, options []AttachVolumeOption) (*preparedAttach, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	if err := validateAttachRef(input.Server); err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}
	if err := validateAttachRef(input.Volume); err != nil {
		return nil, fmt.Errorf("volume: %w", err)
	}
	novaSource, err := captureAttachSource(ctx, nova, "compute")
	if err != nil {
		return nil, err
	}
	cinderSource, err := captureAttachSource(ctx, cinder, "volume")
	if err != nil {
		return nil, err
	}
	p := &preparedAttach{nova: novaSource, cinder: cinderSource}
	p.options, err = applyAttachOptions(options, func() error { return p.guard(ctx) })
	if err != nil {
		return nil, attachContextError(ctx, errors.Join(err, p.guard(ctx)))
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	p.serverID, err = servers.New(&p.nova.client).Resources.ResolveID(ctx, input.Server)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return nil, fmt.Errorf("resolve server: %w", err)
	}
	if err := validateAttachRef(resource.ID(p.serverID)); err != nil {
		return nil, fmt.Errorf("resolve server: %w", err)
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	p.volumeID, err = volumes.New(&p.cinder.client).Resources.ResolveID(ctx, input.Volume)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return nil, fmt.Errorf("resolve volume: %w", err)
	}
	if err := validateAttachRef(resource.ID(p.volumeID)); err != nil {
		return nil, fmt.Errorf("resolve volume: %w", err)
	}
	p.volumeURL = p.cinder.client.ServiceURL("volumes", url.PathEscape(p.volumeID))
	p.attachmentURL = p.nova.client.ServiceURL("servers", url.PathEscape(p.serverID), "os-volume_attachments")
	if err := validateAttachTarget(&p.cinder.client, p.volumeURL); err != nil {
		return nil, err
	}
	if err := validateAttachTarget(&p.nova.client, p.attachmentURL); err != nil {
		return nil, err
	}
	return p, p.guard(ctx)
}

func captureAttachSource(ctx context.Context, source *gophercloud.ServiceClient, role string) (attachSource, error) {
	headers, err := validateAttachSource(ctx, source, role)
	if err != nil {
		return attachSource{}, err
	}
	client := *source
	client.MoreHeaders = maps.Clone(headers)
	return attachSource{
		source: source, client: client, provider: source.ProviderClient,
		endpoint: source.Endpoint, base: source.ResourceBase,
		kind: source.Type, version: source.Microversion, role: role,
	}, nil
}

func (source *attachSource) check(ctx context.Context) error {
	if source.source == nil || source.source.ProviderClient != source.provider || source.source.Endpoint != source.endpoint || source.source.ResourceBase != source.base || source.source.Type != source.kind || source.source.Microversion != source.version {
		return attachInvalid("%s attachment source or route changed", source.role)
	}
	_, err := validateAttachSource(ctx, source.source, source.role)
	return err
}

func (p *preparedAttach) guard(ctx context.Context) error {
	if p.observed != nil {
		return attachContextError(ctx, p.observed)
	}
	err := errors.Join(p.nova.check(ctx), p.cinder.check(ctx))
	if err != nil {
		// Keep an observed source violation terminal even if a callback later
		// restores the original values.
		p.observed = err
	}
	return attachContextError(ctx, err)
}

func (p *preparedAttach) exchange(ctx context.Context, method, target string, body any) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	var client *gophercloud.ServiceClient
	switch {
	case method == http.MethodGet && target == p.volumeURL:
		client = &p.cinder.client
	case method == http.MethodPost && target == p.attachmentURL:
		client = &p.nova.client
	default:
		return nil, attachInvalid("attachment exchange changes its fixed phase target or method")
	}
	response, err := rest.DoJSON(ctx, client, method, target, body, nil, http.StatusOK)
	if observed := p.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	// The actual accepted response must reach the caller even when reading,
	// closing, cancellation or the post-send source check failed.
	return response, attachContextError(ctx, err)
}

func attachContext(ctx context.Context) error {
	if ctx == nil {
		return attachInvalid("context is required")
	}
	return attachContextError(ctx, ctx.Err())
}

func attachContextError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return err
}

func validateAttachRef(ref resource.Ref) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if !attachText(ref.String()) {
		return attachInvalid("attachment reference must be valid UTF-8 without controls")
	}
	if !ref.IsName() {
		for _, character := range ref.String() {
			if unicode.IsSpace(character) {
				return attachInvalid("attachment ID must be a single unescaped URL path segment without whitespace")
			}
		}
	}
	return nil
}

func validateAttachSource(ctx context.Context, source *gophercloud.ServiceClient, role string) (map[string]string, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	if source == nil || source.ProviderClient == nil {
		return nil, attachInvalid("%s service client is required", role)
	}
	allowed := source.Type == ""
	switch role {
	case "compute":
		allowed = allowed || source.Type == "compute"
	case "volume":
		allowed = allowed || source.Type == "block-storage" || source.Type == "block-store" || source.Type == "volume" || source.Type == "volumev3"
	default:
		return nil, attachInvalid("unknown attachment service role")
	}
	if !allowed {
		return nil, fmt.Errorf("%w: %s attachment service client is required", resource.ErrUnsupported, role)
	}
	if !attachText(source.Microversion) {
		return nil, attachInvalid("invalid attachment service microversion")
	}
	for index, endpoint := range []string{source.Endpoint, source.ResourceBaseURL()} {
		if err := validateAttachEndpoint(endpoint, index == 1); err != nil {
			return nil, err
		}
	}
	if err := rest.ValidateTarget(source, source.ResourceBaseURL()); err != nil {
		return nil, err
	}
	return canonicalAttachHeaders(source.MoreHeaders, role, source.Microversion)
}

func validateAttachEndpoint(endpoint string, requireSlash bool) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || !attachText(endpoint) || parsed.User != nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || !attachText(parsed.Path) || requireSlash && !strings.HasSuffix(endpoint, "/") {
		return attachInvalid("attachment endpoint or resource base must be an absolute HTTP(S) URL without credentials, query, fragment or controls; effective base must end in a slash")
	}
	return nil
}

func validateAttachTarget(source *gophercloud.ServiceClient, target string) error {
	if err := rest.ValidateTarget(source, target); err != nil {
		return err
	}
	parsed, err := url.Parse(target)
	if err != nil || !attachText(target) || parsed.RawQuery != "" || parsed.ForceQuery || !attachText(parsed.Path) {
		return attachInvalid("invalid fixed attachment request target")
	}
	return nil
}

func canonicalAttachHeaders(values map[string]string, role, version string) (map[string]string, error) {
	headers := make(map[string]string, len(values))
	for key, value := range values {
		if !attachHeaderToken(key) || !attachHeaderValue(value) {
			return nil, attachInvalid("invalid attachment service header")
		}
		name := http.CanonicalHeaderKey(key)
		if previous, exists := headers[name]; exists && previous != value {
			return nil, attachInvalid("conflicting attachment header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade":
			return nil, attachInvalid("attachment service header %q is SDK owned", key)
		case "openstack-api-version":
			if version == "" || value != role+" "+version {
				return nil, attachInvalid("attachment service version header conflicts with selected microversion")
			}
		case "x-openstack-nova-api-version":
			if role != "compute" || version == "" || value != version {
				return nil, attachInvalid("Nova version header conflicts with attachment service or microversion")
			}
		case "x-openstack-volume-api-version":
			if role != "volume" || version == "" || value != version {
				return nil, attachInvalid("Cinder version header conflicts with attachment service or microversion")
			}
		}
		headers[name] = value
	}
	return headers, nil
}

func attachHeaderToken(value string) bool {
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

func attachHeaderValue(value string) bool {
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
