package containers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// MetadataInfo contains only observed container headers. Missing counters remain nil.
type MetadataInfo struct {
	Values                  map[string]string
	BytesUsed, ObjectCount  *int64
	Timestamp, LastModified *string
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
		custom := strings.HasPrefix(name, "x-container-meta-")
		counter := name == "x-container-bytes-used" || name == "x-container-object-count"
		if !custom && !counter && name != "x-timestamp" && name != "last-modified" {
			continue
		}
		if !metadataToken(key) {
			return nil, fmt.Errorf("invalid container header name %q", key)
		}
		if seen[name] || len(values) != 1 {
			return nil, fmt.Errorf("container header %q must have exactly one value", key)
		}
		seen[name] = true
		value := values[0]
		if custom {
			suffix := key[len("x-container-meta-"):]
			if !metadataToken(suffix) || !metadataFieldValue(value) {
				return nil, fmt.Errorf("invalid container metadata header %q", key)
			}
			result.Values[strings.ToLower(suffix)] = value
		} else if counter {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("container counter %q: %w", key, err)
			}
			switch name {
			case "x-container-bytes-used":
				result.BytesUsed = &parsed
			case "x-container-object-count":
				result.ObjectCount = &parsed
			}
		} else {
			if !metadataFieldValue(value) {
				return nil, fmt.Errorf("invalid container timestamp header")
			}
			if name == "x-timestamp" {
				result.Timestamp = &value
			} else {
				result.LastModified = &value
			}
		}
	}
	return result, nil
}
