package imagedata

import (
	"fmt"
	"maps"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// StageOpts owns staging headers. Size is the expected byte count sent as
// X-OpenStack-Image-Size; it does not set HTTP Content-Length or inspect data.
type StageOpts struct {
	Size    *int64
	Headers map[string]string
}

// StageOption configures a staging workflow. Later options replace earlier values.
type StageOption func(*StageOpts) error

// WithStageOpts snapshots and replaces the complete options value.
func WithStageOpts(value StageOpts) StageOption {
	snapshot := copyStageOpts(value)
	return func(config *StageOpts) error { *config = copyStageOpts(snapshot); return nil }
}

// WithStageSize sets the nonnegative expected byte count, including zero.
func WithStageSize(value int64) StageOption {
	return func(config *StageOpts) error { owned := value; config.Size = &owned; return nil }
}

// WithStageHeader adds an ordinary header. Case-insensitive later options win.
func WithStageHeader(key, value string) StageOption {
	return func(config *StageOpts) error {
		headers, err := stageHeaders(map[string]string{key: value}, false, "")
		if err != nil {
			return err
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		for existing := range config.Headers {
			if strings.EqualFold(existing, key) {
				delete(config.Headers, existing)
			}
		}
		for name, value := range headers {
			config.Headers[name] = value
		}
		return nil
	}
}

// WithStageHeaders snapshots and merges headers, rejecting conflicting case aliases.
func WithStageHeaders(values map[string]string) StageOption {
	snapshot := maps.Clone(values)
	return func(config *StageOpts) error {
		headers, err := stageHeaders(snapshot, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			if err := WithStageHeader(key, value)(config); err != nil {
				return err
			}
		}
		return nil
	}
}

func copyStageOpts(value StageOpts) StageOpts {
	if value.Size != nil {
		owned := *value.Size
		value.Size = &owned
	}
	value.Headers = maps.Clone(value.Headers)
	return value
}

func parseStageOpts(options []StageOption) (StageOpts, error) {
	var config StageOpts
	for _, apply := range options {
		if apply == nil {
			return config, stageInvalid("nil stage option")
		}
		if err := apply(&config); err != nil {
			return config, err
		}
	}
	// Keep the parsed variable separate: custom callbacks may retain its pointer.
	snapshot := copyStageOpts(config)
	if snapshot.Size != nil && *snapshot.Size < 0 {
		return snapshot, stageInvalid("stage size must be nonnegative")
	}
	var err error
	snapshot.Headers, err = stageHeaders(snapshot.Headers, false, "")
	return snapshot, err
}

func stageHeaders(values map[string]string, source bool, version string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for key, value := range values {
		if key == "" || !utf8.ValidString(value) {
			return nil, stageInvalid("invalid stage header")
		}
		for _, c := range []byte(key) {
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
				continue
			}
			return nil, stageInvalid("invalid stage header %q", key)
		}
		for _, c := range []byte(value) {
			if c < 32 && c != '\t' || c == 127 {
				return nil, stageInvalid("invalid stage header value")
			}
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := result[name]; exists && old != value {
			return nil, stageInvalid("conflicting stage header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version", "x-openstack-image-size":
			return nil, stageInvalid("header %q is owned by the SDK", key)
		case "accept", "content-type":
			if !source {
				return nil, stageInvalid("header %q is owned by the SDK", key)
			}
		case "openstack-api-version":
			if !source || version == "" || value != "image "+version {
				return nil, stageInvalid("image version header conflicts with selected microversion")
			}
		}
		result[name] = value
	}
	return result, nil
}

func stageInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
