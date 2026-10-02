package metadata

import (
	"fmt"
	"maps"
	"net/http"
	"strings"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Opts has no body, query, routing or microversion controls.
type Opts struct{}
type Option = request.Option[Opts]

// WithHeader adds an ordinary request header. Later case-insensitive options win.
// Authentication, transport and version headers remain owned by the SDK.
func WithHeader(key, value string) Option {
	return func(config *request.Config[Opts]) error {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid metadata header", resource.ErrInvalidOption)
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		config.Headers[http.CanonicalHeaderKey(key)] = value
		return nil
	}
}

// WithHeaders snapshots and merges headers. Conflicting case aliases inside
// this map are rejected; sequential options can replace an earlier value.
func WithHeaders(headers map[string]string) Option {
	snapshot := maps.Clone(headers)
	return func(config *request.Config[Opts]) error {
		canonical := make(map[string]string, len(snapshot))
		for key, value := range snapshot {
			name := http.CanonicalHeaderKey(key)
			if old, present := canonical[name]; present && old != value {
				return fmt.Errorf("%w: conflicting metadata header aliases %q", resource.ErrInvalidOption, key)
			}
			canonical[name] = value
		}
		for key, value := range canonical {
			if err := WithHeader(key, value)(config); err != nil {
				return err
			}
		}
		return nil
	}
}
