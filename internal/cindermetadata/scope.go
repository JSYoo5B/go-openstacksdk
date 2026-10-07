// Package cindermetadata implements fixed Cinder Volume/Snapshot metadata scopes.
package cindermetadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage/metadata"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type Scope struct {
	client                   *gophercloud.ServiceClient
	id, collection, endpoint string
}

// New captures the selected metadata URI once. No response can retarget it.
func New(ctx context.Context, client *gophercloud.ServiceClient, collection, id string) (*Scope, error) {
	if err := Validate(ctx, client); err != nil {
		return nil, request.Wrap("MetadataIn", "metadata", err)
	}
	if collection != "volumes" && collection != "snapshots" {
		return nil, request.Wrap("MetadataIn", "metadata", invalid("unsupported metadata collection"))
	}
	if err := identifier(id); err != nil {
		return nil, request.Wrap("MetadataIn", collection+"_metadata", err)
	}
	endpoint := client.ServiceURL(collection, url.PathEscape(id), "metadata")
	if err := rest.ValidateTarget(client, endpoint); err != nil {
		return nil, request.Wrap("MetadataIn", collection+"_metadata", err)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.RawQuery != "" {
		return nil, request.Wrap("MetadataIn", collection+"_metadata", invalid("metadata URL cannot contain a query"))
	}
	return &Scope{client: client, id: id, collection: collection, endpoint: endpoint}, nil
}

func (s *Scope) ID() string {
	if s == nil {
		return ""
	}
	return s.id
}
func (s *Scope) RawClient() *gophercloud.ServiceClient {
	if s == nil {
		return nil
	}
	return s.client
}
func (s *Scope) kind() string {
	if s == nil || s.collection == "" {
		return "metadata"
	}
	return s.collection + "_metadata"
}
func (s *Scope) validate(ctx context.Context) error {
	if err := Validate(ctx, s.RawClient()); err != nil {
		return err
	}
	if s.collection != "volumes" && s.collection != "snapshots" {
		return invalid("unsupported metadata collection")
	}
	if err := identifier(s.id); err != nil {
		return err
	}
	return rest.ValidateTarget(s.client, s.endpoint)
}

func (s *Scope) prepare(ctx context.Context, options []metadata.Option) (map[string]string, error) {
	if err := s.validate(ctx); err != nil {
		return nil, err
	}
	config, err := request.Apply(metadata.Opts{}, append([]metadata.Option(nil), options...)...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(config, false, false, true); err != nil {
		return nil, err
	}
	headers, err := canonicalHeaders(config.Headers, false, "")
	if err != nil {
		return nil, err
	}
	// Copy into a distinct variable: a custom option can retain config itself.
	if err := s.validate(ctx); err != nil {
		return nil, err
	}
	source, err := canonicalHeaders(s.client.MoreHeaders, true, s.client.Microversion)
	if err != nil {
		return nil, err
	}
	if requested, present := headers["If-Match"]; present {
		if configured, present := source["If-Match"]; present && configured != requested {
			return nil, invalid("If-Match conflicts with configured service header")
		}
	}
	return headers, nil
}

func (s *Scope) Get(ctx context.Context, options ...metadata.Option) (*metadata.Result, error) {
	headers, err := s.prepare(ctx, options)
	if err != nil {
		return nil, request.Wrap("Get", s.kind(), err)
	}
	return s.object(ctx, "Get", http.MethodGet, nil, headers)
}

func (s *Scope) Merge(ctx context.Context, values map[string]string, options ...metadata.Option) (*metadata.Result, error) {
	values = copyValues(values)
	if err := validateValues(values); err != nil {
		return nil, request.Wrap("Merge", s.kind(), err)
	}
	headers, err := s.prepare(ctx, options)
	if err != nil {
		return nil, request.Wrap("Merge", s.kind(), err)
	}
	return s.object(ctx, "Merge", http.MethodPost, map[string]any{"metadata": values}, headers)
}

func (s *Scope) Replace(ctx context.Context, values map[string]string, options ...metadata.Option) (*metadata.Result, error) {
	values = copyValues(values)
	if err := validateValues(values); err != nil {
		return nil, request.Wrap("Replace", s.kind(), err)
	}
	headers, err := s.prepare(ctx, options)
	if err != nil {
		return nil, request.Wrap("Replace", s.kind(), err)
	}
	return s.object(ctx, "Replace", http.MethodPut, map[string]any{"metadata": values}, headers)
}

// DeleteKeys clears all keys with one PUT when keys is nil. A nonnil empty
// slice performs no HTTP. Other slices retain order and duplicate keys.
func (s *Scope) DeleteKeys(ctx context.Context, keys []string, options ...metadata.Option) (*metadata.DeleteResult, error) {
	var snapshot []string
	if keys != nil {
		snapshot = append(make([]string, 0, len(keys)), keys...)
	}
	headers, err := s.prepare(ctx, options)
	if err != nil {
		return nil, request.Wrap("DeleteKeys", s.kind(), err)
	}
	for _, key := range snapshot {
		if err := metadataKey(key); err != nil {
			return nil, request.Wrap("DeleteKeys", s.kind(), err)
		}
	}
	result := &metadata.DeleteResult{}
	if keys == nil {
		cleared, err := s.object(ctx, "DeleteKeys", http.MethodPut, map[string]any{"metadata": map[string]string{}}, headers)
		if err != nil {
			return nil, err
		}
		result.Cleared = cleared
		return result, nil
	}
	result.Deleted = make([]metadata.DeletedKey, 0, len(snapshot))
	for _, key := range snapshot {
		response, err := s.perform(ctx, http.MethodDelete, s.endpoint+"/"+url.PathEscape(key), nil, headers)
		if err != nil {
			return result, request.Wrap("DeleteKeys", s.kind(), err)
		}
		result.Deleted = append(result.Deleted, metadata.DeletedKey{Key: key, Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode})
	}
	if err := s.validate(ctx); err != nil {
		return result, request.Wrap("DeleteKeys", s.kind(), err)
	}
	return result, nil
}

func (s *Scope) object(ctx context.Context, operation, method string, body any, headers map[string]string) (*metadata.Result, error) {
	response, err := s.perform(ctx, method, s.endpoint, body, headers)
	if err != nil {
		return nil, request.Wrap(operation, s.kind(), err)
	}
	value, err := decode(response)
	if err != nil {
		return nil, request.Wrap(operation, s.kind(), err)
	}
	return value, nil
}

func (s *Scope) perform(ctx context.Context, method, endpoint string, body any, headers map[string]string) (*rest.Response, error) {
	if err := s.validate(ctx); err != nil {
		return nil, err
	}
	// Canonical source aliases avoid native map iteration weakening a condition.
	canonical, err := canonicalHeaders(s.client.MoreHeaders, true, s.client.Microversion)
	if err != nil {
		return nil, err
	}
	if explicit, present := headers["If-Match"]; present {
		if configured, present := canonical["If-Match"]; present && configured != explicit {
			return nil, invalid("If-Match conflicts with configured service header")
		}
	}
	client := *s.client
	client.MoreHeaders = canonical
	response, err := rest.DoJSON(ctx, &client, method, endpoint, body, maps.Clone(headers), http.StatusOK)
	if err != nil && ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	return response, err
}

// Validate checks context and source without introducing a metadata version gate.
func Validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return invalid("Cinder service client is required")
	}
	switch client.Type {
	case "block-storage", "block-store", "volume", "volumev3":
	default:
		return invalid("Cinder service client type is required")
	}
	base := client.ServiceURL()
	if err := rest.ValidateTarget(client, base); err != nil {
		return err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" {
		return invalid("Cinder service base cannot contain a query")
	}
	_, err = canonicalHeaders(client.MoreHeaders, true, client.Microversion)
	return err
}

func canonicalHeaders(headers map[string]string, source bool, version string) (map[string]string, error) {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		if !headerName(key) || !headerValue(value) {
			return nil, invalid("invalid metadata header %q", key)
		}
		name := http.CanonicalHeaderKey(key)
		if old, present := result[name]; present && old != value {
			return nil, invalid("conflicting metadata header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade":
			return nil, invalid("header %q is owned by the SDK", key)
		case "openstack-api-version":
			if !source || version == "" || value != "volume "+version {
				return nil, invalid("Cinder version header conflicts with selected microversion")
			}
		case "x-openstack-volume-api-version":
			if !source || version == "" || value != version {
				return nil, invalid("Cinder version header conflicts with selected microversion")
			}
		}
		result[name] = value
	}
	return result, nil
}

func headerName(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range []byte(value) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			continue
		}
		return false
	}
	return true
}
func headerValue(value string) bool {
	for _, c := range []byte(value) {
		if (c < 32 && c != '\t') || c == 127 {
			return false
		}
	}
	return true
}
func identifier(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return invalid("ID must be valid UTF-8")
	}
	for _, c := range id {
		if unicode.IsSpace(c) || unicode.IsControl(c) {
			return invalid("ID contains whitespace/control characters")
		}
	}
	return nil
}
func metadataKey(key string) error {
	if key == "" || key == "." || key == ".." || !utf8.ValidString(key) {
		return invalid("metadata key must be a nonempty literal UTF-8 path component")
	}
	for _, c := range key {
		if unicode.IsControl(c) {
			return invalid("metadata key contains control characters")
		}
	}
	return nil
}
func copyValues(values map[string]string) map[string]string {
	if values == nil {
		return make(map[string]string)
	}
	return maps.Clone(values)
}
func validateValues(values map[string]string) error {
	for key, value := range values {
		if !utf8.ValidString(key) || !utf8.ValidString(value) {
			return invalid("metadata keys and values must be valid UTF-8 strings")
		}
	}
	return nil
}
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
