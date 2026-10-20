package accounts

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// MetadataInfo contains only observed account headers. Missing counters remain nil.
type MetadataInfo struct {
	Values                                 map[string]string
	BytesUsed, ContainerCount, ObjectCount *int64
	Timestamp                              *string
}

// GetMetadataResult retains HTTP evidence even when atomic header projection fails.
type GetMetadataResult struct {
	Metadata   *MetadataInfo
	Body       []byte
	Header     http.Header
	StatusCode int
}

// MetadataResponse acknowledges a metadata POST without claiming refreshed state.
type MetadataResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

func projectMetadata(headers http.Header) (*MetadataInfo, error) {
	result := &MetadataInfo{Values: make(map[string]string)}
	seen := make(map[string]bool)
	for key, values := range headers {
		name := strings.ToLower(key)
		custom := strings.HasPrefix(name, "x-account-meta-")
		counter := name == "x-account-bytes-used" || name == "x-account-container-count" || name == "x-account-object-count"
		if !custom && !counter && name != "x-timestamp" {
			continue
		}
		if !metadataToken(key) {
			return nil, fmt.Errorf("invalid account header name %q", key)
		}
		if seen[name] || len(values) != 1 {
			return nil, fmt.Errorf("account header %q must have exactly one value", key)
		}
		seen[name] = true
		value := values[0]
		if custom {
			suffix := key[len("x-account-meta-"):]
			if !metadataToken(suffix) || !metadataFieldValue(value) {
				return nil, fmt.Errorf("invalid account metadata header %q", key)
			}
			result.Values[strings.ToLower(suffix)] = value
		} else if counter {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("account counter %q: %w", key, err)
			}
			switch name {
			case "x-account-bytes-used":
				result.BytesUsed = &parsed
			case "x-account-container-count":
				result.ContainerCount = &parsed
			case "x-account-object-count":
				result.ObjectCount = &parsed
			}
		} else {
			if !metadataFieldValue(value) {
				return nil, fmt.Errorf("invalid account timestamp header")
			}
			result.Timestamp = &value
		}
	}
	return result, nil
}
