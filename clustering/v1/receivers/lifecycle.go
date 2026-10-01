package receivers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// TrackedReceiver owns cached fields, a sticky dirty set and a fixed request ID.
// Value is a merged cache view; Response retains the last successful HTTP fields.
// Returned models and their HTTP evidence are independent snapshots.
type TrackedReceiver struct {
	api            *API
	id             string
	collectionPath string
	state          *senlin.TrackedState
	opMu           sync.Mutex
	responseMu     sync.RWMutex
	response       *Receiver
}

// Track wraps a cached model without HTTP. Body is authoritative when present;
// use Edit instead of modifying the input's exported fields. A manually built
// model has a local seed, with no fabricated HTTP response.
func (a *API) Track(value *Receiver) (*TrackedReceiver, error) {
	return a.trackReceiver(value, "", "receivers")
}

func (a *API) trackReceiver(value *Receiver, routeID, path string) (*TrackedReceiver, error) {
	if value == nil {
		return nil, request.Wrap("Track", "clustering.receivers", fmt.Errorf("%w: receiver is required", resource.ErrInvalidOption))
	}
	body, err := senlin.TrackedSeed(value, &value.Metadata)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.receivers", err)
	}
	state, err := senlin.NewTrackedState(body)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.receivers", err)
	}
	response, err := senlin.TrackedDecode(body, &value.Metadata, receiverLifecycleMetadata)
	if err == nil {
		err = normalizeTrackedReceiver(response, routeID == "")
	}
	if err != nil {
		return nil, request.Wrap("Track", "clustering.receivers", err)
	}
	if routeID == "" {
		routeID = response.ID
	}
	if err := senlin.Identifier(routeID); err != nil {
		return nil, request.Wrap("Track", "clustering.receivers", err)
	}
	return &TrackedReceiver{api: a, id: routeID, collectionPath: path, state: state, response: response}, nil
}

// Load fetches an explicit ID once or resolves an exact name through one list
// traversal. An explicit ID remains the route when the response omits it; a
// name reference uses only the selected row's canonical lowercase id field.
func (a *API) Load(ctx context.Context, ref resource.Ref) (*TrackedReceiver, error) {
	return a.loadAt(ctx, ref, "receivers")
}

func (a *API) loadAt(ctx context.Context, ref resource.Ref, path string) (*TrackedReceiver, error) {
	loadSpec := spec(a.RawClient())
	loadSpec.Path = path
	validateItem := loadSpec.ValidateItem
	loadSpec.ValidateItem = func(value *Receiver) error {
		if validateItem != nil {
			if err := validateItem(value); err != nil {
				return err
			}
		}
		return normalizeTrackedReceiver(value, ref.IsName())
	}
	loadSpec.Name = func(value *Receiver) string {
		if value == nil {
			return ""
		}
		var name string
		_ = json.Unmarshal(value.Body["name"], &name)
		return name
	}
	value, err := rest.Collection(loadSpec).Find(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Load", "clustering.receivers", err)
	}
	routeID := ""
	if !ref.IsName() {
		routeID = ref.String()
	}
	tracked, err := a.trackReceiver(value, routeID, path)
	return tracked, request.Wrap("Load", "clustering.receivers", err)
}

func receiverLifecycleMetadata(value *Receiver) *resource.Metadata { return &value.Metadata }

// normalizeTrackedReceiver prevents case-folded JSON aliases from changing the
// canonical identity or exact-name match. Missing/null response identities are
// allowed only when a caller already supplied the fixed request route.
func normalizeTrackedReceiver(value *Receiver, requireID bool) error {
	if value == nil {
		return fmt.Errorf("%w: receiver is required", resource.ErrInvalidOption)
	}
	var id, name string
	if raw, present := value.Body["id"]; present && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if err := json.Unmarshal(raw, &id); err != nil {
			return fmt.Errorf("%w: receiver id must be a string or null", resource.ErrInvalidOption)
		}
		if err := senlin.Identifier(id); err != nil {
			return err
		}
	} else if requireID {
		return fmt.Errorf("%w: tracked receiver requires a canonical id string", resource.ErrInvalidOption)
	}
	if raw, present := value.Body["name"]; present {
		if err := json.Unmarshal(raw, &name); err != nil {
			return fmt.Errorf("%w: receiver name must be a string or null", resource.ErrInvalidOption)
		}
	}
	value.ID, value.Name = id, name
	return nil
}

func (tracked *TrackedReceiver) Value() *Receiver {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, err := senlin.TrackedDecode(tracked.state.Body(), &tracked.response.Metadata, receiverLifecycleMetadata)
	if err != nil || normalizeTrackedReceiver(value, false) != nil {
		return nil
	}
	return value
}

func (tracked *TrackedReceiver) Response() *Receiver {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, err := senlin.TrackedDecode(tracked.response.Body, &tracked.response.Metadata, receiverLifecycleMetadata)
	if err != nil || normalizeTrackedReceiver(value, false) != nil {
		return nil
	}
	return value
}

func (tracked *TrackedReceiver) Dirty() bool {
	return tracked != nil && tracked.state != nil && tracked.state.Dirty()
}

// Edit atomically applies concrete UpdateOpts or owned extension fields without
// HTTP. An empty edit is valid; headers belong to Commit. Params is replaced as
// a whole JSON object or null, rather than merged by key.
func (tracked *TrackedReceiver) Edit(opts UpdateOpts, options ...UpdateOption) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked receiver is required", resource.ErrInvalidOption)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil {
		err = senlin.ValidateTrackedFields[Receiver](config.Fields)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Params, "params")
	}
	forbidden := append(append([]string(nil), responseFields...), "type", "cluster_id", "actor")
	if err == nil {
		for key := range config.Fields {
			for _, protected := range forbidden {
				if strings.EqualFold(key, protected) {
					err = fmt.Errorf("%w: tracked extension %q is owned by the SDK", resource.ErrInvalidOption, key)
					break
				}
			}
			if err != nil {
				break
			}
		}
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, "receiver", forbidden...)
	}
	if err != nil {
		return request.Wrap("Edit", "clustering.receivers", err)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return request.Wrap("Edit", "clustering.receivers", err)
	}
	return request.Wrap("Edit", "clustering.receivers", tracked.state.Edit(envelope["receiver"]))
}

func (tracked *TrackedReceiver) remove(key string) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked receiver is required", resource.ErrInvalidOption)
	}
	tracked.state.Remove(key)
	return nil
}

// RemoveName removes the cached field and records a name:null PATCH. The server
// owns null acceptance; removing an absent field is a no-op.
func (tracked *TrackedReceiver) RemoveName() error { return tracked.remove("name") }

// RemoveAction records action:null while removing the cached command name.
func (tracked *TrackedReceiver) RemoveAction() error { return tracked.remove("action") }

// RemoveParams records params:null while removing the cached field. Edit with
// WithUpdateParams(nil) instead keeps an explicit null in the cached body.
func (tracked *TrackedReceiver) RemoveParams() error { return tracked.remove("params") }

func receiverCommitHeaders(options []UpdateOption) (map[string]string, error) {
	config, err := request.Apply(UpdateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil && (config.Options.Name.IsSet() || config.Options.Action.IsSet() || len(config.Options.Params) > 0) {
		err = fmt.Errorf("%w: Commit options only support headers; call Edit for body changes", resource.ErrInvalidOption)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	return maps.Clone(config.Headers), err
}

// Commit sends dirty fields once, merges a valid 200 response and clears only
// its revisions. Clean commits validate context/client and return without HTTP.
// A successful PATCH followed by a ResponseError leaves dirty state intact;
// callers can Refresh before deciding to resend. No action or channel is fetched.
func (tracked *TrackedReceiver) Commit(ctx context.Context, options ...UpdateOption) (*Receiver, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked receiver is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	headers, err := receiverCommitHeaders(options)
	client := tracked.api.RawClient()
	if err == nil {
		err = senlin.Validate(ctx, client)
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.receivers", err)
	}
	pending := tracked.state.Pending()
	if len(pending.Body) == 0 {
		return tracked.Value(), nil
	}
	body, err := json.Marshal(map[string]any{"receiver": pending.Body})
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.receivers", err)
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Commit", "clustering.receivers", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL(tracked.collectionPath, url.PathEscape(tracked.id)), json.RawMessage(body), headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.receivers", err)
	}
	value, err := rest.Decode(response, "receiver", receiverLifecycleMetadata)
	if err == nil {
		if err = tracked.accept(pending, value); err != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.receivers", err)
	}
	return tracked.Value(), nil
}

// Refresh GETs the fixed route, merges observed fields and clears the dirty
// revisions present before GET. Newer edits survive. Failure changes nothing.
func (tracked *TrackedReceiver) Refresh(ctx context.Context) (*Receiver, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked receiver is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	pending := tracked.state.Pending()
	client := tracked.api.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Refresh", "clustering.receivers", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL(tracked.collectionPath, url.PathEscape(tracked.id)), nil, nil, http.StatusOK)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			err = &resource.NotFoundError{Resource: "clustering.receivers", Reference: tracked.id, Cause: err}
		}
		return nil, request.Wrap("Refresh", "clustering.receivers", err)
	}
	value, err := rest.Decode(response, "receiver", receiverLifecycleMetadata)
	if err == nil {
		if err = tracked.accept(pending, value); err != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, request.Wrap("Refresh", "clustering.receivers", err)
	}
	return tracked.Value(), nil
}

func (tracked *TrackedReceiver) accept(pending senlin.TrackedPatch, value *Receiver) error {
	owned, err := senlin.TrackedDecode(value.Body, &value.Metadata, receiverLifecycleMetadata)
	if err == nil {
		err = normalizeTrackedReceiver(owned, false)
	}
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
