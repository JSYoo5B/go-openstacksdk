package neutronfind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ExistingRowPolicy selects the Source constructor used by the leaf. Bulk
// projection is lazy after POST and never merges request seed attributes.
type ExistingRowPolicy struct{ Bulk bool }

// PostCollection owns a single fixed-path POST. The SDK leaf supplies an owned
// plural envelope and chooses its constructor/header policy; response projection
// remains separate so later malformed rows retain earlier yielded records.
func (e *Engine) PostCollection(ctx context.Context, body map[string]json.RawMessage, policy Options) (*rest.Response, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	owned, err := mutationFields(body)
	if err != nil {
		return nil, err
	}
	source, headers, err := e.mutationSource(ctx, policy)
	if err != nil {
		return nil, err
	}
	target := source.Client.ServiceURL(e.profile.Path)
	parsed, err := url.Parse(target)
	if err != nil || parsed.RawQuery != "" || parsed.ForceQuery {
		return nil, errors.Join(findInvalid("collection POST route must not contain query"), err)
	}
	if _, exists := headers.Values["Content-Type"]; !exists {
		headers.Values["Content-Type"] = "application/json"
	}
	source.Client.MoreHeaders = maps.Clone(headers.Values)
	return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, e.Check, http.MethodPost, target, owned, headers, rest.RejectionPolicy{Codes: findRejectedCodes(), PreserveCleanRetry: true, StopFaultRetryCallbacks: e.profile.StopFaultRetryCallbacks}, e.codes()...)
}

// ProjectExistingRow retains the actual whole response and remote row evidence.
// Constructor failures produce no logical Resource; the leaf attaches the same
// physical receipt to its iteration error instead of creating placeholder rows.
func (e *Engine) ProjectExistingRow(ctx context.Context, row json.RawMessage, response *rest.Response, policy ExistingRowPolicy) (*Record, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	if response == nil {
		return nil, findInvalid("existing row requires actual response")
	}
	value := &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, fromList: true}
	if err := value.UnmarshalJSON(bytes.Clone(row)); err != nil {
		return value, response.Fail(err)
	}
	forbidden := []string{"connection", "microversion", "_synchronized"}
	if e.profile.SourceFindPolicy {
		forbidden = append(forbidden, "cls")
	}
	if policy.Bulk {
		forbidden = append(forbidden, "self")
	}
	for _, key := range forbidden {
		if _, exists := value.Wire.Body[key]; exists {
			return value, response.Fail(findInvalid("existing row %q collides with Source constructor", key))
		}
	}
	value.Wire.Header, value.Wire.StatusCode = response.Header.Clone(), response.StatusCode
	if err := errors.Join(e.findProject(ctx, value, nil), e.Check(ctx)); err != nil {
		return value, response.Fail(err)
	}
	return value, nil
}
