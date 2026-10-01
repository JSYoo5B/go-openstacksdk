package shareaccessrules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
	"gophercloudsdk/sharedfilesystems/v2/shares"
)

// AccessRuleScope owns access IDs within a share. It uses the modern 2.45+
// endpoints; legacy access_list actions remain available through Shares API.
type AccessRuleScope struct {
	api       *API
	shareID   string
	resources *resource.Collection[AccessRule]
}

func (a *API) InShare(ctx context.Context, parent resource.Ref) (*AccessRuleScope, error) {
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return nil, invalid("a service client is required")
	}
	if err := requireVersion(ctx, a.client, 45); err != nil {
		return nil, err
	}
	id, err := shares.New(a.client).Resources.ResolveID(ctx, parent)
	if err != nil {
		return nil, err
	}
	scope := &AccessRuleScope{api: a, shareID: id}
	scope.resources = resource.NewCollection(resource.Adapter[AccessRule]{
		Kind: "share access rules",
		Get: func(ctx context.Context, id string) (*AccessRule, error) {
			value, err := scope.Get(ctx, id)
			return value, maskParent(err)
		},
		ID:     func(value *AccessRule) string { return value.ID },
		Status: func(value *AccessRule) string { return value.State },
		Failed: func(state string) bool { return strings.EqualFold(state, "error") },
		Delete: func(ctx context.Context, id string) error {
			return maskParent(scope.Deny(ctx, id, WithDenyIgnoreMissing(false)))
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*AccessRule, error] {
			return func(yield func(*AccessRule, error) bool) {
				if len(query) != 0 {
					yield(nil, resource.ErrUnsupported)
					return
				}
				for value, err := range scope.List(ctx) {
					if !yield(value, err) {
						return
					}
				}
			}
		},
	})
	return scope, nil
}

func (s *AccessRuleScope) ShareID() string { return s.shareID }

func invalid(format string, values ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, values...))
}

func requireVersion(ctx context.Context, client *gophercloud.ServiceClient, minimum int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parts := strings.Split(client.Microversion, ".")
	if client.Microversion == "" {
		return fmt.Errorf("%w: share access scope requires selected Manila microversion 2.%d", resource.ErrUnsupported, minimum)
	}
	if len(parts) != 2 {
		return invalid("invalid selected microversion %q", client.Microversion)
	}
	for _, part := range parts {
		if part == "" {
			return invalid("invalid selected microversion %q", client.Microversion)
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return invalid("invalid selected microversion %q", client.Microversion)
			}
		}
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return invalid("selected microversion is out of range")
	}
	if major != 2 || minor < minimum {
		return fmt.Errorf("%w: operation requires Manila 2.%d or newer, selected %s", resource.ErrUnsupported, minimum, client.Microversion)
	}
	return nil
}

func extraHeaders[T any](cfg request.Config[T], accept string) (map[string]string, error) {
	return request.MergeHeadersFor(map[string]string{
		"Accept": accept, "Content-Type": "application/json", "X-Auth-Token": "",
		"X-OpenStack-Manila-API-Version": "", "OpenStack-API-Version": "",
	}, cfg.Headers, cfg.Options)
}

func cleanHeaders(headers map[string]string) map[string]string {
	// Authentication and microversion headers are supplied by the shared client.
	delete(headers, "X-Auth-Token")
	delete(headers, "X-OpenStack-Manila-API-Version")
	delete(headers, "OpenStack-API-Version")
	return headers
}

func (s *AccessRuleScope) checkParent(ctx context.Context) error {
	value, err := shares.New(s.api.client).Get(ctx, s.shareID)
	if err == nil && (value == nil || value.ID != s.shareID) {
		err = fmt.Errorf("parent response has a missing or different share id")
	}
	if err != nil {
		return &ParentError{ShareID: s.shareID, Cause: err}
	}
	return nil
}

func normalizeRuleError(id string, err error) error {
	var parent *ParentError
	if !errors.As(err, &parent) && gophercloud.ResponseCodeIs(err, 404) {
		return &resource.NotFoundError{Resource: "share access rules", Reference: id, Cause: err}
	}
	return err
}

// Get checks the global access endpoint's share_id before returning the rule.
func (s *AccessRuleScope) Get(ctx context.Context, id string, options ...GetOption) (*AccessRule, error) {
	if err := requireVersion(ctx, s.api.client, 45); err != nil {
		return nil, err
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, err
	}
	cfg, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	headers, err := extraHeaders(cfg, "application/json")
	if err != nil {
		return nil, err
	}
	var wire struct {
		Access json.RawMessage `json:"access"`
	}
	response, err := s.api.client.Get(ctx, s.api.client.ServiceURL("share-access-rules", id), &wire,
		&gophercloud.RequestOpts{OkCodes: []int{200}, MoreHeaders: cleanHeaders(headers)})
	if err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			if parentErr := s.checkParent(ctx); parentErr != nil {
				err = parentErr
			}
		}
		return nil, request.Wrap("Get", "share access rules", normalizeRuleError(id, err))
	}
	value, err := decodeRule(wire.Access, response.Header, s.shareID)
	if err == nil && value.ShareID != s.shareID {
		err = &ParentMismatchError{ShareID: s.shareID, AccessID: id, ActualShareID: value.ShareID}
	}
	if err != nil {
		return nil, request.Wrap("Get", "share access rules", err)
	}
	return value, nil
}

// List lazily yields the modern endpoint's slice. No pagination capability is
// inferred from Python's generic limit/marker mapping or fabricated next links.
func (s *AccessRuleScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*AccessRule, error] {
	return func(yield func(*AccessRule, error) bool) {
		if err := requireVersion(ctx, s.api.client, 45); err != nil {
			yield(nil, err)
			return
		}
		cfg, err := request.Apply(ListOpts{}, options...)
		if err == nil {
			err = request.ValidateCapabilities(cfg, false, true, true)
		}
		if err != nil {
			yield(nil, err)
			return
		}
		if _, exists := cfg.Query["share_id"]; exists {
			yield(nil, invalid("share_id is fixed by the scope"))
			return
		}
		headers, err := extraHeaders(cfg, "application/json")
		if err != nil {
			yield(nil, err)
			return
		}
		cfg.Query.Set("share_id", s.shareID)
		var wire struct {
			Rules json.RawMessage `json:"access_list"`
		}
		response, err := s.api.client.Get(ctx, s.api.client.ServiceURL("share-access-rules")+"?"+cfg.Query.Encode(), &wire,
			&gophercloud.RequestOpts{OkCodes: []int{200}, MoreHeaders: cleanHeaders(headers)})
		if err != nil {
			yield(nil, request.Wrap("List", "share access rules", err))
			return
		}
		var entries []json.RawMessage
		if len(wire.Rules) == 0 || string(wire.Rules) == "null" {
			yield(nil, fmt.Errorf("access_list must be a JSON array"))
			return
		}
		if err := json.Unmarshal(wire.Rules, &entries); err != nil {
			yield(nil, err)
			return
		}
		for _, raw := range entries {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			value, err := decodeRule(raw, response.Header, s.shareID)
			if err == nil && value.ShareID != "" && value.ShareID != s.shareID {
				err = &ParentMismatchError{ShareID: s.shareID, AccessID: value.ID, ActualShareID: value.ShareID}
			}
			if err != nil {
				yield(nil, request.Wrap("List", "share access rules", err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func (s *AccessRuleScope) All(ctx context.Context, options ...ListOption) ([]*AccessRule, error) {
	values := make([]*AccessRule, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *AccessRuleScope) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*AccessRule, error) {
	value, err := s.resources.Find(ctx, ref, options...)
	return value, restoreParent(err)
}
func (s *AccessRuleScope) ResolveID(ctx context.Context, ref resource.Ref) (string, error) {
	return s.resources.ResolveID(ctx, ref)
}
func (s *AccessRuleScope) Delete(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return restoreParent(s.resources.Delete(ctx, ref, options...))
}
func (s *AccessRuleScope) Wait(ctx context.Context, ref resource.Ref, state string, options ...resource.WaitOption) (*AccessRule, error) {
	value, err := s.resources.Wait(ctx, ref, state, options...)
	return value, restoreParent(err)
}
func (s *AccessRuleScope) WaitDeleted(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return restoreParent(s.resources.WaitDeleted(ctx, ref, options...))
}

// parentBridge keeps the shared collection's 404 policy from hiding a missing
// share. Public methods restore the complete original error, including its cause.
type parentBridge struct{ cause error }

func (e *parentBridge) Error() string { return e.cause.Error() }
func maskParent(err error) error {
	var parent *ParentError
	if errors.As(err, &parent) {
		return &parentBridge{cause: err}
	}
	return err
}
func restoreParent(err error) error {
	var bridge *parentBridge
	if errors.As(err, &bridge) {
		return bridge.cause
	}
	return err
}

// Allow grants access via the fixed share's action endpoint and returns the
// creation response. Acceptance does not imply state=active; use Wait afterwards.
func (s *AccessRuleScope) Allow(ctx context.Context, opts AllowOpts, options ...AllowOption) (*AccessRule, error) {
	if err := requireVersion(ctx, s.api.client, 45); err != nil {
		return nil, err
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, true)
	}
	if err != nil {
		return nil, err
	}
	if cfg.Options.AccessType == "" || cfg.Options.AccessTo == "" {
		return nil, invalid("access_type and access_to are required")
	}
	if cfg.Options.AccessLevel != "" && cfg.Options.AccessLevel != "ro" && cfg.Options.AccessLevel != "rw" {
		return nil, invalid("access_level must be ro or rw")
	}
	if cfg.Options.LockVisibility != nil || cfg.Options.LockDeletion != nil || cfg.Options.LockReason != nil {
		if err := requireVersion(ctx, s.api.client, 82); err != nil {
			return nil, err
		}
	}
	body, err := gophercloud.BuildRequestBody(cfg.Options, "allow_access")
	if err == nil {
		body, err = request.MergeFieldsFor(body, cfg.Fields, cfg.Options)
	}
	if err != nil {
		return nil, err
	}
	headers, err := extraHeaders(cfg, "application/json")
	if err != nil {
		return nil, err
	}
	var wire struct {
		Access json.RawMessage `json:"access"`
	}
	response, err := s.api.client.Post(ctx, s.api.client.ServiceURL("shares", s.shareID, "action"), body, &wire,
		&gophercloud.RequestOpts{OkCodes: []int{200}, MoreHeaders: cleanHeaders(headers)})
	if err != nil {
		return nil, request.Wrap("Allow", "share access rules", err)
	}
	value, err := decodeRule(wire.Access, response.Header, s.shareID)
	if err == nil && value.ShareID != "" && value.ShareID != s.shareID {
		err = &ParentMismatchError{ShareID: s.shareID, AccessID: value.ID, ActualShareID: value.ShareID}
	}
	return value, request.Wrap("Allow", "share access rules", err)
}

func (s *AccessRuleScope) Create(ctx context.Context, opts AllowOpts, options ...AllowOption) (*AccessRule, error) {
	return s.Allow(ctx, opts, options...)
}

// Deny verifies the rule's parent before revocation. Only a missing rule in an
// existing, readable share is ignored by default; parent and authorization errors
// remain observable. Unrestrict, including explicit false, requires Manila 2.82.
func (s *AccessRuleScope) Deny(ctx context.Context, id string, options ...DenyOption) error {
	if err := requireVersion(ctx, s.api.client, 45); err != nil {
		return err
	}
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	cfg, err := request.Apply(DenyOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, true)
	}
	if err != nil {
		return err
	}
	if cfg.Options.Unrestrict != nil {
		if err := requireVersion(ctx, s.api.client, 82); err != nil {
			return err
		}
	}
	wireOpts := struct {
		AccessID   string `json:"access_id"`
		Unrestrict *bool  `json:"unrestrict,omitempty"`
	}{id, cfg.Options.Unrestrict}
	body, err := gophercloud.BuildRequestBody(wireOpts, "deny_access")
	if err == nil {
		body, err = request.MergeFieldsFor(body, cfg.Fields, wireOpts)
	}
	if err != nil {
		return err
	}
	headers, err := extraHeaders(cfg, "")
	if err != nil {
		return err
	}
	ignore := cfg.Options.IgnoreMissing == nil || *cfg.Options.IgnoreMissing
	_, err = s.Get(ctx, id)
	if err != nil {
		var parent *ParentError
		if ignore && errors.Is(err, resource.ErrNotFound) && !errors.As(err, &parent) {
			return nil
		}
		return err
	}
	_, err = s.api.client.Post(ctx, s.api.client.ServiceURL("shares", s.shareID, "action"), body, nil,
		&gophercloud.RequestOpts{OkCodes: []int{200, 202}, MoreHeaders: cleanHeaders(headers)})
	if gophercloud.ResponseCodeIs(err, 404) {
		if parentErr := s.checkParent(ctx); parentErr != nil {
			return request.Wrap("Deny", "share access rules", parentErr)
		}
		if ignore {
			return nil
		}
	}
	return request.Wrap("Deny", "share access rules", normalizeRuleError(id, err))
}
