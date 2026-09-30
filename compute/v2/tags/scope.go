package tags

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/tags"
	"gophercloudsdk/compute/v2/servers"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// ServerTagScope owns a fixed server's tag set. Obtain it from API.InServer;
// it does not mutate a Server model or cache the server's tags.
type ServerTagScope struct {
	api      *API
	serverID string
}

// InServer resolves a name exactly once using the SDK's shared server lookup.
// An explicit ID requires no lookup request. The client's selected Compute
// microversion must be at least 2.26; this method does not change it.
func (a *API) InServer(ctx context.Context, server resource.Ref) (*ServerTagScope, error) {
	if err := a.validateScope(ctx); err != nil {
		return nil, request.Wrap("InServer", "server tags", err)
	}
	id, err := servers.New(a.client).Resources.ResolveID(ctx, server)
	if err != nil {
		return nil, request.Wrap("InServer", "server tags", err)
	}
	return &ServerTagScope{api: a, serverID: id}, nil
}

// ServerID returns the resolved parent ID without a request.
func (s *ServerTagScope) ServerID() string { return s.serverID }

func (a *API) validateScope(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return fmt.Errorf("%w: server tags require a service client", resource.ErrInvalidOption)
	}
	version := a.client.Microversion
	if version == "" {
		return fmt.Errorf("%w: server tags require Compute microversion 2.26 or newer; none was selected", resource.ErrUnsupported)
	}
	parts := strings.Split(version, ".")
	if len(parts) != 2 || !digits(parts[0]) || !digits(parts[1]) {
		return fmt.Errorf("%w: invalid selected Compute microversion %q", resource.ErrInvalidOption, version)
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil {
		return fmt.Errorf("%w: selected Compute microversion %q is out of range", resource.ErrInvalidOption, version)
	}
	if major != 2 || minor < 26 {
		return fmt.Errorf("%w: server tags require Compute microversion 2.26 or newer, got %s", resource.ErrUnsupported, version)
	}
	return nil
}

func digits(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func validateTag(tag string) error {
	if !utf8.ValidString(tag) || utf8.RuneCountInString(tag) == 0 || utf8.RuneCountInString(tag) > 60 || strings.ContainsAny(tag, ",/") {
		return fmt.Errorf("%w: a server tag must contain 1 to 60 Unicode characters and cannot contain comma or slash", resource.ErrInvalidOption)
	}
	return nil
}

func escapedTag(tag string) string {
	// PathEscape deliberately leaves dots intact. Encode whole dot segments so
	// proxies and HTTP routers cannot interpret an otherwise valid tag as a path.
	if tag == "." || tag == ".." {
		return strings.Repeat("%2E", len(tag))
	}
	return url.PathEscape(tag)
}

// Add ensures that the tag is present. Nova's 201 (added) and 204 (already
// present) both succeed; the request has no JSON body.
func (s *ServerTagScope) Add(ctx context.Context, tag string) error {
	if err := s.api.validateScope(ctx); err != nil {
		return request.Wrap("Add", "server tags", err)
	}
	if err := validateTag(tag); err != nil {
		return request.Wrap("Add", "server tags", err)
	}
	err := upstream.Add(ctx, s.api.client, s.serverID, escapedTag(tag)).ExtractErr()
	return s.resultError("Add", tag, err, false)
}

type missingPolicy struct{ ignore bool }

// MissingOption configures 404 handling for Check, Remove and RemoveAll.
// Later options win. Other HTTP failures are always returned.
type MissingOption func(*missingPolicy) error

// WithIgnoreMissing controls whether a 404 is treated as absence. Both true
// and false are explicit values; false preserves a not-found error.
func WithIgnoreMissing(ignore bool) MissingOption {
	return func(p *missingPolicy) error { p.ignore = ignore; return nil }
}

// WithMissingError gives Check and deletion the strict Python TagMixin policy.
func WithMissingError() MissingOption { return WithIgnoreMissing(false) }

func parseMissing(options []MissingOption) (missingPolicy, error) {
	policy := missingPolicy{ignore: true}
	for _, apply := range options {
		if apply == nil {
			return policy, fmt.Errorf("%w: nil tag missing option", resource.ErrInvalidOption)
		}
		if err := apply(&policy); err != nil {
			return policy, err
		}
	}
	return policy, nil
}

// Check returns true for Nova's 204, and false for a 404 by default. Nova uses
// 404 for both a missing server and a missing tag; the endpoint cannot distinguish
// them. WithMissingError preserves that HTTP error as resource.ErrNotFound.
func (s *ServerTagScope) Check(ctx context.Context, tag string, options ...MissingOption) (bool, error) {
	if err := s.api.validateScope(ctx); err != nil {
		return false, request.Wrap("Check", "server tags", err)
	}
	if err := validateTag(tag); err != nil {
		return false, request.Wrap("Check", "server tags", err)
	}
	policy, err := parseMissing(options)
	if err != nil {
		return false, request.Wrap("Check", "server tags", err)
	}
	// Native CheckResult.Extract clears 404. Read the original error so the
	// caller can select the strict policy without losing the HTTP cause.
	err = upstream.Check(ctx, s.api.client, s.serverID, escapedTag(tag)).Err
	return err == nil, s.resultError("Check", tag, err, policy.ignore)
}

// List fetches the server's tag strings. Tag collection requests take no
// query filters; server-list tag filters belong to Servers.All/List instead.
func (s *ServerTagScope) List(ctx context.Context) ([]string, error) {
	if err := s.api.validateScope(ctx); err != nil {
		return nil, request.Wrap("List", "server tags", err)
	}
	values, err := upstream.List(ctx, s.api.client, s.serverID).Extract()
	if err != nil {
		return nil, s.resultError("List", "", err, false)
	}
	if values == nil {
		values = []string{}
	}
	return values, nil
}

// Replace replaces the entire set. A nil or empty slice sends {"tags":[]},
// clearing all tags. Options use concrete ReplaceAllOpts and library-owned
// builders; body extensions are passed to the server, which validates them.
// Core "tags" cannot be overwritten with an extension. Query/header extensions
// are unsupported for this operation.
func (s *ServerTagScope) Replace(ctx context.Context, values []string, options ...ReplaceAllOption) ([]string, error) {
	if err := s.api.validateScope(ctx); err != nil {
		return nil, request.Wrap("Replace", "server tags", err)
	}
	cfg, err := request.Apply(ReplaceAllOpts{Tags: values}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	if err != nil {
		return nil, request.Wrap("Replace", "server tags", err)
	}
	if len(cfg.Options.Tags) > 50 {
		return nil, request.Wrap("Replace", "server tags", fmt.Errorf("%w: a server supports at most 50 tags", resource.ErrInvalidOption))
	}
	for _, tag := range cfg.Options.Tags {
		if err := validateTag(tag); err != nil {
			return nil, request.Wrap("Replace", "server tags", err)
		}
	}
	// Copy the selected concrete input and preserve [] for an empty JSON array.
	cfg.Options.Tags = append([]string{}, cfg.Options.Tags...)
	builder := replaceAllOptsBuilder{base: cfg.Options, config: cfg}
	result, err := upstream.ReplaceAll(ctx, s.api.client, s.serverID, builder).Extract()
	if err != nil {
		return nil, s.resultError("Replace", "", err, false)
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

// Remove ensures that a single tag is absent. A 404 succeeds by default;
// WithMissingError preserves the failure, including a missing parent server.
func (s *ServerTagScope) Remove(ctx context.Context, tag string, options ...MissingOption) error {
	if err := s.api.validateScope(ctx); err != nil {
		return request.Wrap("Remove", "server tags", err)
	}
	if err := validateTag(tag); err != nil {
		return request.Wrap("Remove", "server tags", err)
	}
	policy, err := parseMissing(options)
	if err != nil {
		return request.Wrap("Remove", "server tags", err)
	}
	err = upstream.Delete(ctx, s.api.client, s.serverID, escapedTag(tag)).ExtractErr()
	return s.resultError("Remove", tag, err, policy.ignore)
}

// RemoveAll ensures that all tags are absent. A 404 succeeds by default;
// WithMissingError preserves a missing parent server's error.
func (s *ServerTagScope) RemoveAll(ctx context.Context, options ...MissingOption) error {
	if err := s.api.validateScope(ctx); err != nil {
		return request.Wrap("RemoveAll", "server tags", err)
	}
	policy, err := parseMissing(options)
	if err != nil {
		return request.Wrap("RemoveAll", "server tags", err)
	}
	err = upstream.DeleteAll(ctx, s.api.client, s.serverID).ExtractErr()
	return s.resultError("RemoveAll", "", err, policy.ignore)
}

func (s *ServerTagScope) resultError(operation, tag string, err error, ignoreMissing bool) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		if ignoreMissing {
			return nil
		}
		reference := s.serverID
		if tag != "" {
			reference += "/" + tag
		}
		err = &resource.NotFoundError{Resource: "server tags", Reference: reference, Cause: err}
	}
	return request.Wrap(operation, "server tags", err)
}
