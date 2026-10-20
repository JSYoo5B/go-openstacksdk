package senlin

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const listMicroversionArgument = "senlin.list.microversion"

// WithListHeader selects an additional per-call header. Request preparation
// validates SDK-owned authentication, version and transport header protection.
func WithListHeader[T any](key, value string) request.Option[T] {
	return request.WithHeader[T](key, value)
}

// WithListMicroversion selects a per-call numeric 1.N microversion. Explicit
// empty selects the server's default 1.0, rather than inheriting the client.
func WithListMicroversion[T any](value string) request.Option[T] {
	return request.WithArgument[T](listMicroversionArgument, value)
}

// ValidateListCapabilities keeps data options separate from per-call headers
// and the one SDK-owned microversion argument. Caller namespaces remain exact.
func ValidateListCapabilities[T any](config request.Config[T], arguments ...string) error {
	allowed := append(append([]string(nil), arguments...), listMicroversionArgument)
	if err := request.ValidateCapabilities(config, false, true, true, allowed...); err != nil {
		return err
	}
	return Headers(config.Headers)
}

// ListQueryConfig removes already-prepared request controls from an independent
// configuration copy, preserving exact Body filter namespace validation.
func ListQueryConfig[T any](config request.Config[T]) request.Config[T] {
	config.Headers = nil
	config.Arguments = maps.Clone(config.Arguments)
	delete(config.Arguments, listMicroversionArgument)
	return config
}

// PrepareListClient snapshots per-call header/version controls without changing
// the shared service client or provider. No controls retains the source client,
// including its existing continuation revalidation behavior.
func PrepareListClient[T any](ctx context.Context, source *gophercloud.ServiceClient, config request.Config[T]) (*gophercloud.ServiceClient, error) {
	if err := Validate(ctx, source); err != nil {
		return nil, err
	}
	if err := Headers(config.Headers); err != nil {
		return nil, err
	}
	version, selected, err := request.Argument[string](config, listMicroversionArgument)
	if err != nil {
		return nil, err
	}
	if len(config.Headers) == 0 && !selected {
		return source, nil
	}
	effective := *source
	effective.MoreHeaders, err = canonicalListHeaders(source.MoreHeaders)
	if err != nil {
		return nil, err
	}
	headers, err := canonicalListHeaders(config.Headers)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		effective.MoreHeaders[key] = value
	}
	if selected {
		effective.Microversion = version
	}
	// An explicit source version header is retained. A stale header that would
	// overwrite the selected per-call version must fail, rather than being
	// silently discarded or rewriting caller-configured service headers.
	if err := Validate(ctx, &effective); err != nil {
		return nil, err
	}
	return &effective, nil
}

func canonicalListHeaders(input map[string]string) (map[string]string, error) {
	owned := make(map[string]string, len(input))
	for key, value := range input {
		canonical := http.CanonicalHeaderKey(key)
		if previous, exists := owned[canonical]; exists && previous != value {
			return nil, fmt.Errorf("%w: conflicting case-insensitive list header %q", resource.ErrInvalidOption, key)
		}
		owned[canonical] = value
	}
	return owned, nil
}

// RejectListControlQuery prevents inherited request controls from becoming
// vendor query parameters. Base-path and JMESPath overrides remain unsupported.
func RejectListControlQuery(query url.Values) error {
	for key := range query {
		switch key {
		case "headers", "microversion", "base_path", "jmespath_filters", "allow_unknown_params", "max_items", "paginated":
			return fmt.Errorf("%w: query %q is a local list request control", resource.ErrInvalidOption, key)
		}
	}
	return nil
}

// ListSpec rebuilds client-capturing service gates for the effective client.
// The original source is still revalidated before each page, while operation
// minimum-version gates apply to the effective per-call version.
func ListSpec[T any](source, effective *gophercloud.ServiceClient, factory func(*gophercloud.ServiceClient) rest.CollectionSpec[T]) rest.CollectionSpec[T] {
	spec := factory(effective)
	validate := spec.Validate
	spec.Validate = func(ctx context.Context) error {
		if err := Validate(ctx, source); err != nil {
			return err
		}
		if validate != nil {
			return validate(ctx)
		}
		return Validate(ctx, effective)
	}
	return spec
}
