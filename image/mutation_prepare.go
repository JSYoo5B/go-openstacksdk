package image

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedImageMutation struct {
	service  *Service
	source   *gophercloud.ServiceClient
	client   *gophercloud.ServiceClient
	base     string
	provider *gophercloud.ProviderClient
	id       string
}

func (s *Service) prepareImageMutation(ctx context.Context, ref resource.Ref, tag *string, options []ImageMutationOption) (*preparedImageMutation, error) {
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateImageMutationSource(ctx, source)
	if err != nil {
		return nil, err
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if !utf8.ValidString(ref.String()) {
		return nil, uploadInvalid("image reference must be valid UTF-8")
	}
	if !ref.IsName() {
		if err := createImportID(ref.String()); err != nil {
			return nil, err
		}
	}
	if tag != nil {
		if err := validateImageMutationTag(*tag); err != nil {
			return nil, err
		}
	}
	client := *source
	client.MoreHeaders = headers
	prepared := &preparedImageMutation{service: s, source: source, client: &client, base: source.ServiceURL(), provider: source.ProviderClient}
	policy, err := parseImageMutationOptions(options)
	if err != nil {
		return nil, err
	}
	if err := prepared.check(ctx); err != nil {
		return nil, err
	}
	for key, value := range policy.Headers {
		client.MoreHeaders[key] = value
	}
	id := ref.String()
	if ref.IsName() {
		id, err = New(&client).Images.ResolveID(ctx, ref)
		if err != nil {
			if sourceErr := prepared.check(ctx); sourceErr != nil {
				err = errors.Join(err, sourceErr)
			}
			return nil, err
		}
		if !utf8.ValidString(id) {
			return nil, uploadInvalid("resolved image ID must be valid UTF-8")
		}
		if err := createImportID(id); err != nil {
			return nil, err
		}
	}
	if err := prepared.check(ctx); err != nil {
		return nil, err
	}
	prepared.id = id
	return prepared, nil
}

func (prepared *preparedImageMutation) check(ctx context.Context) error {
	if prepared.service.client != prepared.source || prepared.source.ProviderClient != prepared.provider {
		return uploadInvalid("image mutation source or provider changed")
	}
	if _, err := validateImageMutationSource(ctx, prepared.source); err != nil {
		return err
	}
	return rest.ValidateTarget(prepared.source, prepared.base)
}

func (s *Service) executeImageMutation(ctx context.Context, ref resource.Ref, tag *string, action, method string, options []ImageMutationOption) (string, *rest.Response, error) {
	prepared, err := s.prepareImageMutation(ctx, ref, tag, options)
	if err != nil {
		return "", nil, err
	}
	endpoint := prepared.base + "images/" + url.PathEscape(prepared.id)
	if tag != nil {
		endpoint += "/tags/" + url.PathEscape(*tag)
	} else {
		endpoint += "/actions/" + action
	}
	response, err := rest.DoJSON(ctx, prepared.client, method, endpoint, nil, nil, http.StatusNoContent)
	return prepared.id, response, err
}

func validateImageMutationTag(value string) error {
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > 255 || value == "." || value == ".." || strings.ContainsAny(value, "/\\") {
		return uploadInvalid("image tag must be a nonempty valid UTF-8 segment of at most 255 characters")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return uploadInvalid("image tag must not contain control characters")
		}
	}
	return nil
}

func validateImageMutationSource(ctx context.Context, client *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, uploadInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || client.ProviderClient == nil {
		return nil, uploadInvalid("image service client is required")
	}
	if client.Type != "image" {
		return nil, fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if !utf8.ValidString(client.Endpoint) || !utf8.ValidString(base) {
		return nil, uploadInvalid("image service target must be valid UTF-8")
	}
	if err := rest.ValidateTarget(client, base); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return nil, uploadInvalid("image service base must be query-free and end in a slash")
	}
	if !utf8.ValidString(client.Microversion) || !downloadHeaderValue(client.Microversion) {
		return nil, uploadInvalid("invalid image microversion")
	}
	return imageMutationHeaders(client.MoreHeaders, true, client.Microversion)
}

func imageMutationHeaders(values map[string]string, source bool, version string) (map[string]string, error) {
	headers := make(map[string]string, len(values))
	for key, value := range values {
		if key == "" || !utf8.ValidString(value) || !downloadHeaderValue(value) {
			return nil, uploadInvalid("invalid image mutation header")
		}
		for _, char := range []byte(key) {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
				continue
			}
			return nil, uploadInvalid("invalid image mutation header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if previous, exists := headers[name]; exists && previous != value {
			return nil, uploadInvalid("conflicting image mutation header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version", "x-openstack-image-size":
			return nil, uploadInvalid("mutation header %q is owned by the SDK", key)
		case "accept", "content-type":
			if !source {
				return nil, uploadInvalid("mutation header %q is owned by the SDK", key)
			}
		case "openstack-api-version":
			if !source || version == "" || value != "image "+version {
				return nil, uploadInvalid("image version header conflicts with selected microversion")
			}
		}
		headers[name] = value
	}
	return headers, nil
}

func wrapImageMutationError(ctx context.Context, operation string, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap(operation, "image", err)
}
