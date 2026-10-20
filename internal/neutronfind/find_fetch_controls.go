package neutronfind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// fetchControlPath implements the bounded Source URI mapping used by fetch.
// Other URI fields/conversions are deliberately not guessed from query values.
func fetchControlPath(path, identity string, fetch bool) (string, error) {
	var out strings.Builder
	for i := 0; i < len(path); {
		if path[i] != '%' {
			out.WriteByte(path[i])
			i++
			continue
		}
		switch {
		case strings.HasPrefix(path[i:], "%%"):
			out.WriteByte('%')
			i += 2
		case strings.HasPrefix(path[i:], "%(id)s"):
			if !fetch {
				return "", findInvalid("fallback base_path requires an unavailable id URI parameter")
			}
			out.WriteString(url.PathEscape(identity))
			i += len("%(id)s")
		default:
			return "", findInvalid("base_path supports only literal routes, %%(id)s and %%%% URI formatting")
		}
	}
	return out.String(), nil
}

func fetchControlHeaders(raw json.RawMessage, base map[string]string) (map[string]string, error) {
	headers := maps.Clone(base)
	if headers == nil {
		headers = make(map[string]string)
	}
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, errors.Join(findInvalid("headers must contain complete UTF-8 JSON"), err)
	}
	if !truthy {
		return headers, nil
	}
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, errors.Join(findInvalid("truthy fallback headers must be a string-valued JSON object"), err)
	}
	for _, member := range members {
		var text string
		if trim := bytes.TrimSpace(member.Value); len(trim) == 0 || trim[0] != '"' {
			return nil, findInvalid("fallback header %q must be a JSON string", member.Key)
		}
		if err := json.Unmarshal(member.Value, &text); err != nil {
			return nil, err
		}
		for key := range headers {
			if strings.EqualFold(key, member.Key) {
				delete(headers, key)
			}
		}
		headers[member.Key] = text
	}
	return headers, nil
}

// findFetchResponse selects a response override only when truthy and present.
// The selected projector still owns the physical, unmodified HTTP envelope.
func (e *Engine) findFetchResponse(ctx context.Context, response *rest.Response, seed map[string]json.RawMessage, version string, key json.RawMessage) (*Record, error) {
	selected := *e
	if len(key) != 0 {
		truthy, err := cloudfilter.PythonJSONTruthy(key)
		if err != nil {
			return nil, response.Fail(err)
		}
		if truthy && json.Valid(response.Body) {
			trim := bytes.TrimSpace(key)
			if len(trim) != 0 && (trim[0] == '[' || trim[0] == '{') {
				return nil, response.Fail(findInvalid("truthy resource_response_key must be a hashable JSON scalar"))
			}
			var name string
			if len(trim) != 0 && trim[0] == '"' {
				if err := json.Unmarshal(key, &name); err != nil {
					return nil, response.Fail(err)
				}
				var envelope map[string]json.RawMessage
				if err := json.Unmarshal(response.Body, &envelope); err == nil {
					if _, present := envelope[name]; present {
						selected.profile.Singular = name
					}
				}
			}
		}
	}
	value, err := selected.findResponse(ctx, response, seed, version)
	if err != nil {
		err = response.Fail(err)
	}
	return value, err
}

// FindWithFetchControls is an opt-in Source Resource.find profile. It consumes
// fetch's raw kwargs without changing Find, Get, List or other leaf bindings.
// skip_cache is accepted but Go's physical reader has no response cache. URI
// templates support %(id)s/%% only; routes remain inside the captured service.
func (e *Engine) FindWithFetchControls(ctx context.Context, identity string, policy Options) (*Record, error) {
	if err := errors.Join(cloudread.Context(ctx), e.Check(ctx)); err != nil {
		return nil, err
	}
	policy = CloneOptions(policy)
	original := CloneOptions(policy)
	controls := make(map[string]json.RawMessage)
	for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key", "headers"} {
		if raw, exists := policy.Query[key]; exists {
			if !utf8.Valid(raw) || !json.Valid(raw) {
				return nil, findInvalid("%s must contain complete UTF-8 JSON", key)
			}
			controls[key] = bytes.Clone(raw)
			removeProxyListKey(&policy, key)
		}
	}
	requiresID := true
	if raw, exists := controls["requires_id"]; exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		var err error
		requiresID, err = cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return nil, err
		}
	}
	path := e.profile.Path
	pathOverridden := policy.BasePath != nil
	if policy.BasePath != nil {
		path = *policy.BasePath
	}
	if raw, exists := controls["base_path"]; exists {
		value, err := proxyListNullableString(raw, "base_path")
		if err != nil {
			return nil, err
		}
		path = e.profile.Path
		pathOverridden = value != nil
		if value != nil {
			path = *value
		}
	}
	getPath := path
	var err error
	if pathOverridden {
		getPath, err = fetchControlPath(path, identity, true)
		if err != nil {
			return nil, err
		}
	}
	getEngine := e
	if pathOverridden {
		getEngine, err = e.withListBasePath(ctx, getPath)
		if err != nil {
			return nil, err
		}
	}
	// An explicit empty path is meaningful to Resource._prepare_request. The
	// older list-route helper intentionally uses empty as "keep default".
	if pathOverridden && getPath == "" {
		getEngine.profile.Path = ""
	}
	getQuery, listQuery, filters, seed, control, err := e.prepareQuery(policy, identity, true)
	if err != nil {
		return nil, err
	}
	// These kwargs are also passed to Resource.existing before fetch consumes
	// them. Preserve unknown constructor attributes only for opted-in profiles.
	for _, key := range original.Order {
		if raw, exists := controls[key]; exists {
			e.consumeFindBody(seed, []cloudfilter.JSONMember{{Key: key, Value: raw}})
		}
	}
	if raw, present := controls["headers"]; present {
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, err
		}
		if len(values) != 0 {
			getQuery["headers"] = values
		}
	}
	source := *e.source
	if e.profile.SourceFindPolicy && policy.Microversion != nil {
		stripVersion(&source)
	}
	if err := errors.Join(source.WithPolicy(ctx, policy.Microversion, policy.Headers), e.Check(ctx)); err != nil {
		return nil, err
	}
	headerPolicy := e.headerPolicy(&source, false, policy.Headers)
	source.Client.MoreHeaders = maps.Clone(headerPolicy.Values)
	get := func(callCtx context.Context, id string) (*Record, error) {
		target := source.Client.ServiceURL(getEngine.profile.Path)
		if requiresID {
			target = source.Client.ServiceURL(getEngine.profile.Path, url.PathEscape(id))
		}
		if len(getQuery) != 0 {
			target += "?" + getQuery.Encode()
		}
		response, cause := rest.DoJSONGuardedRejectionsHeaderPolicy(callCtx, &source.Client, e.Check, http.MethodGet, target, nil, headerPolicy, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true, StopFaultRetryCallbacks: e.profile.StopFaultRetryCallbacks}, e.codes()...)
		if cause != nil {
			// Source error_message decorates HTTP exceptions, not transport or
			// accepted-response decoder faults. A single unwrap retains clean fallback.
			if raw, exists := controls["error_message"]; exists && fetchControlHTTPCause(cause) {
				truthy, _ := cloudfilter.PythonJSONTruthy(raw)
				if truthy {
					text, err := cloudfilter.PythonString(raw)
					if err != nil {
						return nil, errors.Join(cause, err)
					}
					cause = fmt.Errorf("%s: %w", text, cause)
				}
			}
			return nil, cause
		}
		owned := findFields(seed)
		owned[e.profile.IDKey], _ = json.Marshal(id)
		value, cause := e.findFetchResponse(callCtx, response, owned, source.Client.Microversion, controls["resource_response_key"])
		if cause = errors.Join(cause, e.Check(callCtx)); cause != nil {
			cause = response.Fail(cause)
		}
		return value, cause
	}
	list := func(callCtx context.Context, _ url.Values) iter.Seq2[*Record, error] {
		fail := func(cause error) iter.Seq2[*Record, error] {
			return func(yield func(*Record, error) bool) { yield(nil, cause) }
		}
		listPath := path
		listPathOverridden := pathOverridden
		if policy.ListBasePath != nil && *policy.ListBasePath != "" {
			listPath = *policy.ListBasePath
			listPathOverridden = true
		}
		var cause error
		if listPathOverridden {
			listPath, cause = fetchControlPath(listPath, identity, false)
			if cause != nil {
				return fail(cause)
			}
		}
		selected := e
		if listPathOverridden {
			selected, cause = e.withListBasePath(callCtx, listPath)
			if cause != nil {
				return fail(cause)
			}
		}
		if listPathOverridden && listPath == "" {
			selected.profile.Path = ""
			actual, pathErr := url.Parse(e.source.Client.ServiceURL(""))
			if pathErr != nil {
				return fail(pathErr)
			}
			selected.profile.VersionedPath = actual.EscapedPath()
		}
		listPolicy := CloneOptions(policy)
		if raw, present := controls["headers"]; present {
			listPolicy.Headers, cause = fetchControlHeaders(raw, listPolicy.Headers)
			if cause != nil {
				return fail(cause)
			}
		}
		listSource := *e.source
		version := policy.Microversion
		if e.profile.SourceFindPolicy {
			version = nil
		}
		if cause := errors.Join(listSource.WithPolicy(callCtx, version, listPolicy.Headers), e.Check(callCtx)); cause != nil {
			return fail(cause)
		}
		return selected.listRecords(callCtx, &listSource, listPolicy, listQuery, filters, control)
	}
	collection := resource.NewCollection(resource.Adapter[Record]{Kind: e.profile.Kind, IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment,
		IdentityResponseID: func(value *Record) (string, error) { return findText(value, e.profile.IDKey), nil }, Name: func(value *Record) string { return findText(value, e.profile.NameKey) }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, errors.Join(err, e.Check(ctx))
}

func fetchControlHTTPCause(err error) bool {
	var native gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &native) && native.Actual >= 400
}
