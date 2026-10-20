package objects

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MetadataInfo contains observed headers. Missing fields remain nil; strings
// retain their literal values, including an explicitly empty value.
type MetadataInfo struct {
	Values        map[string]string
	ContentLength *int64
	ETag, ContentType, ContentEncoding, ContentDisposition, Timestamp,
	LastModified, DeleteAt, ObjectManifest, CacheControl, ContentLanguage,
	Expires, RobotsTag *string
}

// GetMetadataResult retains actual HTTP evidence even when projection fails.
type GetMetadataResult struct {
	Metadata   *MetadataInfo
	Body       []byte
	Header     http.Header
	StatusCode int
}

// MetadataResponse acknowledges a POST without claiming refreshed state.
type MetadataResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

// MetadataChangeResult keeps the observed state and write acknowledgement
// separate. A write error may still retain a completed Before response.
type MetadataChangeResult struct {
	Before          *GetMetadataResult
	Acknowledgement *MetadataResponse
}

func metadataStringFields(value *MetadataInfo) map[string]**string {
	return map[string]**string{
		"etag": &value.ETag, "content-type": &value.ContentType,
		"content-encoding": &value.ContentEncoding, "content-disposition": &value.ContentDisposition,
		"x-timestamp": &value.Timestamp, "last-modified": &value.LastModified,
		"x-delete-at": &value.DeleteAt, "x-object-manifest": &value.ObjectManifest,
		"cache-control": &value.CacheControl, "content-language": &value.ContentLanguage,
		"expires": &value.Expires, "x-robots-tag": &value.RobotsTag,
	}
}

func projectMetadata(headers http.Header) (*MetadataInfo, error) {
	result := &MetadataInfo{Values: make(map[string]string)}
	fields := metadataStringFields(result)
	seen := make(map[string]bool)
	for key, values := range headers {
		name := strings.ToLower(key)
		custom := strings.HasPrefix(name, "x-object-meta-")
		field, known := fields[name]
		if !custom && !known && name != "content-length" {
			continue
		}
		if !metadataToken(key) {
			return nil, fmt.Errorf("invalid object header name %q", key)
		}
		if seen[name] || len(values) != 1 {
			return nil, fmt.Errorf("object header %q must have exactly one value", key)
		}
		seen[name] = true
		value := values[0]
		if name == "content-length" {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("object content length: %w", err)
			}
			result.ContentLength = &parsed
		} else {
			if !metadataFieldValue(value) {
				return nil, fmt.Errorf("invalid object header value %q", key)
			}
			if custom {
				suffix := key[len("x-object-meta-"):]
				if !metadataToken(suffix) {
					return nil, fmt.Errorf("invalid object metadata suffix %q", key)
				}
				result.Values[strings.ToLower(suffix)] = value
			} else {
				*field = &value
			}
		}
	}
	return result, nil
}

func mutableMetadataHeaders(value *MetadataInfo) map[string]string {
	result := make(map[string]string)
	for name, field := range map[string]*string{
		"Content-Type": value.ContentType, "Content-Encoding": value.ContentEncoding,
		"Content-Disposition": value.ContentDisposition, "X-Delete-At": value.DeleteAt,
		"X-Object-Manifest": value.ObjectManifest, "Cache-Control": value.CacheControl,
		"Content-Language": value.ContentLanguage, "Expires": value.Expires,
		"X-Robots-Tag": value.RobotsTag,
	} {
		if field != nil {
			result[name] = *field
		}
	}
	return result
}

func rejectMetadataSymlink(headers http.Header) error {
	seen := make(map[string]bool)
	for key, values := range headers {
		name := strings.ToLower(key)
		if name != "x-symlink-target" && name != "x-symlink-target-account" && name != "x-symlink-target-etag" && name != "x-symlink-target-bytes" {
			continue
		}
		if !metadataToken(key) || seen[name] || len(values) != 1 || !metadataFieldValue(values[0]) {
			return fmt.Errorf("invalid object symlink header %q", key)
		}
		seen[name] = true
	}
	if seen["x-symlink-target"] {
		return fmt.Errorf("%w: metadata mutation of an observed symlink", resource.ErrUnsupported)
	}
	return nil
}
