package neutronfind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// mutationFields owns every byte before policy or transport work. Nil raw
// members represent explicit JSON null, while a nil map is an empty object.
func mutationFields(value map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	owned := findFields(value)
	for key, raw := range owned {
		if !utf8.ValidString(key) {
			return nil, findInvalid("mutation field name must be UTF-8")
		}
		if raw == nil {
			owned[key] = json.RawMessage("null")
			continue
		}
		if !utf8.Valid(raw) {
			return nil, findInvalid("mutation field %q must contain UTF-8 JSON", key)
		}
		if !json.Valid(raw) {
			var decoded any
			cause := json.Unmarshal(raw, &decoded)
			return nil, errors.Join(findInvalid("mutation field %q must contain complete JSON", key), cause)
		}
	}
	return owned, nil
}

// mutationSource applies the same captured service, selected version and fixed
// header policy as a member read. Query belongs to read/list APIs, not mutations.
func (e *Engine) mutationSource(ctx context.Context, policy Options) (*cloudread.Source, rest.RequestHeaderPolicy, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, rest.RequestHeaderPolicy{}, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, rest.RequestHeaderPolicy{}, err
	}
	if len(policy.Query) != 0 {
		return nil, rest.RequestHeaderPolicy{}, findInvalid("mutation and seed projection do not accept query options")
	}
	policy = CloneOptions(policy)
	source := *e.source
	if e.profile.SourceFindPolicy && policy.Microversion != nil {
		stripVersion(&source)
	}
	if err := errors.Join(source.WithPolicy(ctx, policy.Microversion, policy.Headers), e.Check(ctx)); err != nil {
		return nil, rest.RequestHeaderPolicy{}, err
	}
	headers := e.headerPolicy(&source, false, policy.Headers)
	source.Client.MoreHeaders = maps.Clone(headers.Values)
	return &source, headers, nil
}

// mutationReceipt preserves physical evidence without inventing a decoded
// resource. DELETE and failed physical response handling have no Resource/Wire.
func mutationReceipt(response *rest.Response) *Record {
	if response == nil {
		return nil
	}
	return &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

// Mutate performs one fixed collection POST, member PUT or bodyless member
// DELETE using shared REST ownership and rejection handling. Leaves prepare
// their Source dirty fields and seed state; this engine does not send defaults,
// allocate resources, infer parent identities or decide an update is a no-op.
func (e *Engine) Mutate(ctx context.Context, method, identity string, body, seed map[string]json.RawMessage, policy Options) (*Record, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	switch method {
	case http.MethodPost:
		if identity != "" {
			return nil, findInvalid("collection POST requires an empty identity")
		}
	case http.MethodPut, http.MethodDelete:
		if err := findSegment(identity); err != nil {
			return nil, err
		}
	default:
		return nil, findInvalid("mutation method must be POST, PUT or DELETE")
	}
	if method == http.MethodDelete && len(body) != 0 {
		return nil, findInvalid("DELETE must be bodyless")
	}
	if method != http.MethodDelete && e.profile.Singular == "" {
		return nil, findInvalid("mutation singular envelope is required")
	}
	ownedBody, err := mutationFields(body)
	if err != nil {
		return nil, err
	}
	ownedSeed, err := mutationFields(seed)
	if err != nil {
		return nil, err
	}
	source, headers, err := e.mutationSource(ctx, policy)
	if err != nil {
		return nil, err
	}
	target := source.Client.ServiceURL(e.profile.Path)
	if method != http.MethodPost {
		target = source.Client.ServiceURL(e.profile.Path, url.PathEscape(identity))
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.RawQuery != "" || parsed.ForceQuery {
		return nil, errors.Join(findInvalid("mutation route must not contain a query"), err)
	}
	var payload any
	if method != http.MethodDelete {
		// JSONBody causes native Gophercloud to emit this default. Own it across
		// retries, while preserving an explicitly selected Source/caller value.
		if _, supplied := headers.Values["Content-Type"]; !supplied {
			headers.Values["Content-Type"] = "application/json"
		}
		source.Client.MoreHeaders = maps.Clone(headers.Values)
		payload = map[string]any{e.profile.Singular: ownedBody}
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, e.Check, method, target, payload, headers, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true, StopFaultRetryCallbacks: e.profile.StopFaultRetryCallbacks}, findCodes()...)
	if err != nil {
		return mutationReceipt(response), errors.Join(err, e.Check(ctx))
	}
	if method == http.MethodDelete {
		value := mutationReceipt(response)
		if err := e.Check(ctx); err != nil {
			return value, response.Fail(err)
		}
		return value, nil
	}
	value, err := e.findResponse(ctx, response, ownedSeed, source.Client.Microversion)
	if err = errors.Join(err, e.Check(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}

// ProjectSeed validates the selected transport policy, then returns an owned
// Source/default view without HTTP or fabricated response evidence. A leaf
// decides which prepared update is a no-op and validates its identity first.
func (e *Engine) ProjectSeed(ctx context.Context, seed map[string]json.RawMessage, policy Options) (*Record, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	owned, err := mutationFields(seed)
	if err != nil {
		return nil, err
	}
	source, _, err := e.mutationSource(ctx, policy)
	if err != nil {
		return nil, err
	}
	value := &Record{microversion: source.Client.Microversion}
	if err := errors.Join(e.findProject(ctx, value, owned), e.Check(ctx)); err != nil {
		return nil, err
	}
	return value, nil
}
