package policies

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TrackedPolicy owns cached fields, a sticky dirty set and a fixed request ID.
// Value is a merged cache view; Response retains the last successful HTTP fields.
// Changes to either returned snapshot never change the handle.
type TrackedPolicy struct {
	api        *API
	id         string
	state      *senlin.TrackedState
	opMu       sync.Mutex
	responseMu sync.RWMutex
	response   *Policy
}

// Track wraps a cached model without HTTP. Body is authoritative when present;
// use Edit for changes rather than mutating the input's exported typed fields.
// A manually constructed model has a local seed and no invented HTTP evidence.
func (a *API) Track(value *Policy) (*TrackedPolicy, error) {
	return a.trackPolicy(value, "")
}

func (a *API) trackPolicy(value *Policy, routeID string) (*TrackedPolicy, error) {
	if value == nil {
		return nil, request.Wrap("Track", "clustering.policies", fmt.Errorf("%w: policy is required", resource.ErrInvalidOption))
	}
	body, err := senlin.TrackedSeed(value, &value.Metadata)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.policies", err)
	}
	state, err := senlin.NewTrackedState(body)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.policies", err)
	}
	response, err := senlin.TrackedDecode(body, &value.Metadata, policyMetadata)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.policies", err)
	}
	if routeID == "" {
		routeID = response.ID
	}
	if err := senlin.Identifier(routeID); err != nil {
		return nil, request.Wrap("Track", "clustering.policies", err)
	}
	return &TrackedPolicy{api: a, id: routeID, state: state, response: response}, nil
}

// Load resolves an explicit reference once and tracks that fetched/listed model.
// An explicit ID remains the request identity even if the response ID differs.
func (a *API) Load(ctx context.Context, ref resource.Ref) (*TrackedPolicy, error) {
	value, err := rest.Collection(spec(a.RawClient())).Find(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Load", "clustering.policies", err)
	}
	routeID := ""
	if !ref.IsName() {
		routeID = ref.String()
	}
	return a.trackPolicy(value, routeID)
}

func policyMetadata(value *Policy) *resource.Metadata { return &value.Metadata }

func (tracked *TrackedPolicy) Value() *Policy {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, _ := senlin.TrackedDecode(tracked.state.Body(), &tracked.response.Metadata, policyMetadata)
	return value
}

func (tracked *TrackedPolicy) Response() *Policy {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, _ := senlin.TrackedDecode(tracked.response.Body, &tracked.response.Metadata, policyMetadata)
	return value
}

func (tracked *TrackedPolicy) Dirty() bool {
	return tracked != nil && tracked.state != nil && tracked.state.Dirty()
}

// Edit applies the existing owned UpdateOpts/With options atomically. Headers
// belong to Commit, and readonly/core fields retain the stateless Update policy.
func (tracked *TrackedPolicy) Edit(opts UpdateOpts, options ...UpdateOption) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked policy is required", resource.ErrInvalidOption)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil {
		err = senlin.ValidateTrackedFields[Policy](config.Fields)
	}
	if err == nil && config.Options.Name != nil {
		err = validateName(*config.Options.Name)
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, "policy", append(append([]string(nil), responseFields...), "spec")...)
	}
	if err != nil {
		return request.Wrap("Edit", "clustering.policies", err)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return request.Wrap("Edit", "clustering.policies", err)
	}
	return request.Wrap("Edit", "clustering.policies", tracked.state.Edit(envelope["policy"]))
}

// RemoveName records deletion as an explicit name:null PATCH. The server owns
// null acceptance; ordinary Edit retains the existing nonempty name validation.
func (tracked *TrackedPolicy) RemoveName() error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked policy is required", resource.ErrInvalidOption)
	}
	tracked.state.Remove("name")
	return nil
}

func policyCommitHeaders(options []UpdateOption) (map[string]string, error) {
	config, err := request.Apply(UpdateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil && config.Options.Name != nil {
		err = fmt.Errorf("%w: Commit options only support headers; call Edit for body changes", resource.ErrInvalidOption)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	return maps.Clone(config.Headers), err
}

// Commit sends dirty fields once, merges a valid response and clears accepted
// revisions. Clean commits return the cache without HTTP. A ResponseError may
// follow a successful mutation: dirty state remains and callers should Refresh
// before deciding to resend; the SDK never automatically retries it.
func (tracked *TrackedPolicy) Commit(ctx context.Context, options ...UpdateOption) (*Policy, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked policy is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	headers, err := policyCommitHeaders(options)
	if err == nil {
		err = senlin.Validate(ctx, tracked.api.RawClient())
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.policies", err)
	}
	pending := tracked.state.Pending()
	if len(pending.Body) == 0 {
		return tracked.Value(), nil
	}
	body, err := json.Marshal(map[string]any{"policy": pending.Body})
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.policies", err)
	}
	client := tracked.api.RawClient()
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL("policies", url.PathEscape(tracked.id)), json.RawMessage(body), headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.policies", err)
	}
	value, err := rest.Decode(response, "policy", policyMetadata)
	if err == nil {
		err = tracked.accept(pending, value)
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.policies", err)
	}
	return tracked.Value(), nil
}

// Refresh GETs the fixed ID, merges returned fields and resets the dirty
// revisions present before GET. A failure leaves cache and changes untouched.
func (tracked *TrackedPolicy) Refresh(ctx context.Context) (*Policy, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked policy is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	pending := tracked.state.Pending()
	value, err := tracked.api.Get(ctx, tracked.id)
	if err == nil {
		err = tracked.accept(pending, value)
	}
	if err != nil {
		return nil, request.Wrap("Refresh", "clustering.policies", err)
	}
	return tracked.Value(), nil
}

func (tracked *TrackedPolicy) accept(pending senlin.TrackedPatch, value *Policy) error {
	owned, err := senlin.TrackedDecode(value.Body, &value.Metadata, policyMetadata)
	if err != nil {
		return err
	}
	tracked.responseMu.Lock()
	defer tracked.responseMu.Unlock()
	if err := tracked.state.Accept(pending, owned.Body); err != nil {
		return err
	}
	tracked.response = owned
	return nil
}
