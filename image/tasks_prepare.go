package image

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
)

// preparedTaskSource fixes source identity and routing while retaining live
// provider authentication and its configured native request policy.
type preparedTaskSource struct {
	service                       *Service
	source                        *gophercloud.ServiceClient
	client                        *gophercloud.ServiceClient
	provider                      *gophercloud.ProviderClient
	endpoint, base, kind, version string
}

func (s *Service) captureTaskSource(ctx context.Context) (*preparedTaskSource, error) {
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateImageMutationSource(ctx, source)
	if err != nil {
		return nil, err
	}
	client := *source
	client.MoreHeaders = headers
	return &preparedTaskSource{service: s, source: source, client: &client, provider: source.ProviderClient,
		endpoint: source.Endpoint, base: source.ServiceURL(), kind: source.Type, version: source.Microversion}, nil
}

func (p *preparedTaskSource) check(ctx context.Context) error {
	if p.service.client != p.source || p.source.ProviderClient != p.provider ||
		p.source.Endpoint != p.endpoint || p.source.ServiceURL() != p.base || p.source.Type != p.kind || p.source.Microversion != p.version {
		return uploadInvalid("task source, provider or service target changed")
	}
	_, err := validateImageMutationSource(ctx, p.source)
	return err
}

func (p *preparedTaskSource) finish(ctx context.Context, headers map[string]string, err error) error {
	if sourceErr := p.check(ctx); sourceErr != nil {
		if err == nil {
			return sourceErr
		}
		return errors.Join(err, sourceErr)
	}
	if err != nil {
		return err
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	return nil
}

// Following pages refresh ordinary source headers; fixed option headers still
// take precedence. Target identity and provider remain the captured originals.
func (p *preparedTaskSource) refresh(ctx context.Context, headers map[string]string) error {
	if err := p.check(ctx); err != nil {
		return err
	}
	sourceHeaders, err := validateImageMutationSource(ctx, p.source)
	if err != nil {
		return err
	}
	for key, value := range headers {
		sourceHeaders[key] = value
	}
	p.client.MoreHeaders = sourceHeaders
	return nil
}

func validateTaskID(value string) error {
	if value == "" || !utf8.ValidString(value) || value == "." || value == ".." || strings.ContainsAny(value, "/\\") {
		return uploadInvalid("task ID must be a nonempty valid UTF-8 URI segment")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return uploadInvalid("task ID must not contain control characters")
		}
	}
	return nil
}

func taskQueryText(value string) error {
	if !utf8.ValidString(value) {
		return uploadInvalid("task query must be valid UTF-8")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return uploadInvalid("task query must not contain control characters")
		}
	}
	return nil
}
