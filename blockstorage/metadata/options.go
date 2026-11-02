package metadata

import (
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Opts has no body, query, routing or microversion controls.
type Opts struct{}

// Option is one option of the Cinder metadata operations (Get, Merge,
// Replace, DeleteKeys and the record metadata exchanges): WithHeaders,
// request.Header (one ordinary header, validated and merged like WithHeaders)
// or request.Custom over *request.Config[Opts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type Option = request.GetCall

func setHeader(config *request.Config[Opts], key, value string) error {
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%w: invalid metadata header", resource.ErrInvalidOption)
	}
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	config.Headers[http.CanonicalHeaderKey(key)] = value
	return nil
}

// WithHeaders snapshots and merges ordinary request headers. Later
// case-insensitive options win. Conflicting case aliases inside this map are
// rejected; sequential options can replace an earlier value. Authentication,
// transport and version headers remain owned by the SDK. request.Header
// applies one header as a one-entry WithHeaders map.
func WithHeaders(headers map[string]string) Option {
	snapshot := maps.Clone(headers)
	return request.Custom(func(config *request.Config[Opts]) error {
		canonical := make(map[string]string, len(snapshot))
		for key, value := range snapshot {
			name := http.CanonicalHeaderKey(key)
			if old, present := canonical[name]; present && old != value {
				return fmt.Errorf("%w: conflicting metadata header aliases %q", resource.ErrInvalidOption, key)
			}
			canonical[name] = value
		}
		for key, value := range canonical {
			if err := setHeader(config, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}
