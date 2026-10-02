package secrets

import (
	"fmt"
	"strings"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// FetchOpts controls composition. Nil Payload selects the automatic payload
// behavior; nil ContentType selects the response's content_types.default.
// An explicit empty ContentType remains an explicit (empty) Accept value.
type FetchOpts struct {
	Payload     *bool
	ContentType *string
}

type FetchOption = request.Option[FetchOpts]

func copyFetchOptions(value FetchOpts) FetchOpts {
	if value.Payload != nil {
		owned := *value.Payload
		value.Payload = &owned
	}
	if value.ContentType != nil {
		owned := *value.ContentType
		value.ContentType = &owned
	}
	return value
}

func WithFetchOptions(value FetchOpts) FetchOption {
	owned := copyFetchOptions(value)
	return func(config *request.Config[FetchOpts]) error {
		config.Options = copyFetchOptions(owned)
		return nil
	}
}
func WithFetchPayload(value bool) FetchOption {
	return func(config *request.Config[FetchOpts]) error {
		owned := value
		config.Options.Payload = &owned
		return nil
	}
}
func WithFetchContentType(value string) FetchOption {
	return func(config *request.Config[FetchOpts]) error {
		owned := value
		config.Options.ContentType = &owned
		return nil
	}
}
func WithFetchHeader(key, value string) FetchOption {
	return request.WithHeader[FetchOpts](key, value)
}

func validateFetchHeaders(headers map[string]string) error {
	seen := make(map[string]string, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid fetch header", resource.ErrInvalidOption)
		}
		canonical := strings.ToLower(key)
		if previous, exists := seen[canonical]; exists && previous != value {
			return fmt.Errorf("%w: conflicting case variants of header %q", resource.ErrInvalidOption, key)
		}
		seen[canonical] = value
		switch canonical {
		case "accept", "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "openstack-api-version":
			return fmt.Errorf("%w: header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}
