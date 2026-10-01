package clusters

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

// TrackedCluster owns cached fields, a sticky dirty set and a fixed request ID.
// Value is a merged cache view; Response retains the last accepted HTTP fields.
// Operation is an asynchronous submission, never proof of applied changes.
// Each returned model and its HTTP evidence are independent snapshots.
type TrackedCluster struct {
	api            *API
	id             string
	collectionPath string
	state          *senlin.TrackedState
	opMu           sync.Mutex
	responseMu     sync.RWMutex
	response       *Cluster
}

// Track wraps a cached model without HTTP. Body is authoritative when present;
// use Edit instead of modifying the input's exported fields. A manually built
// model has a local seed, with no fabricated HTTP response or action.
func (a *API) Track(value *Cluster) (*TrackedCluster, error) {
	return a.trackCluster(value, "", "clusters")
}

func (a *API) trackCluster(value *Cluster, routeID, path string) (*TrackedCluster, error) {
	if value == nil {
		return nil, request.Wrap("Track", "clustering.clusters", fmt.Errorf("%w: cluster is required", resource.ErrInvalidOption))
	}
	body, err := senlin.TrackedSeed(value, &value.Metadata)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.clusters", err)
	}
	state, err := senlin.NewTrackedState(body)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.clusters", err)
	}
	response, err := senlin.TrackedDecode(body, &value.Metadata, clusterLifecycleMetadata)
	if err == nil {
		err = normalizeTrackedCluster(response, routeID == "")
	}
	if err != nil {
		return nil, request.Wrap("Track", "clustering.clusters", err)
	}
	response.Operation = value.Operation.Snapshot()
	if routeID == "" {
		routeID = response.ID
	}
	if err := senlin.Identifier(routeID); err != nil {
		return nil, request.Wrap("Track", "clustering.clusters", err)
	}
	return &TrackedCluster{api: a, id: routeID, collectionPath: path, state: state, response: response}, nil
}

// Load fetches an explicit ID once or resolves an exact name through one list
// traversal. An explicit ID remains the route even when the response omits it;
// a name reference uses only the selected row's canonical lowercase id field.
func (a *API) Load(ctx context.Context, ref resource.Ref) (*TrackedCluster, error) {
	return a.loadAt(ctx, ref, "clusters")
}

func (a *API) loadAt(ctx context.Context, ref resource.Ref, path string) (*TrackedCluster, error) {
	loadSpec := spec(a.RawClient())
	loadSpec.Path = path
	validateItem := loadSpec.ValidateItem
	loadSpec.ValidateItem = func(value *Cluster) error {
		if validateItem != nil {
			if err := validateItem(value); err != nil {
				return err
			}
		}
		return normalizeTrackedCluster(value, ref.IsName())
	}
	loadSpec.Name = func(value *Cluster) string {
		if value == nil {
			return ""
		}
		var name string
		_ = json.Unmarshal(value.Body["name"], &name)
		return name
	}
	value, err := rest.Collection(loadSpec).Find(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Load", "clustering.clusters", err)
	}
	routeID := ""
	if !ref.IsName() {
		routeID = ref.String()
	}
	tracked, err := a.trackCluster(value, routeID, path)
	return tracked, request.Wrap("Load", "clustering.clusters", err)
}

func clusterLifecycleMetadata(value *Cluster) *resource.Metadata { return &value.Metadata }

// normalizeTrackedCluster keeps case-folded JSON aliases from changing the
// canonical identity or the exact-name match. Missing/null response identities
// are allowed only when a caller already supplied the fixed request route.
func normalizeTrackedCluster(value *Cluster, requireID bool) error {
	if value == nil {
		return fmt.Errorf("%w: cluster is required", resource.ErrInvalidOption)
	}
	var id, name string
	if raw, present := value.Body["id"]; present && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if err := json.Unmarshal(raw, &id); err != nil {
			return fmt.Errorf("%w: cluster id must be a string or null", resource.ErrInvalidOption)
		}
		if err := senlin.Identifier(id); err != nil {
			return err
		}
	} else if requireID {
		return fmt.Errorf("%w: tracked cluster requires a canonical id string", resource.ErrInvalidOption)
	}
	if raw, present := value.Body["name"]; present {
		if err := json.Unmarshal(raw, &name); err != nil {
			return fmt.Errorf("%w: cluster name must be a string or null", resource.ErrInvalidOption)
		}
	}
	value.ID, value.Name = id, name
	return nil
}

func (tracked *TrackedCluster) Value() *Cluster {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, err := senlin.TrackedDecode(tracked.state.Body(), &tracked.response.Metadata, clusterLifecycleMetadata)
	if err != nil || normalizeTrackedCluster(value, false) != nil {
		return nil
	}
	value.Operation = tracked.response.Operation.Snapshot()
	return value
}

func (tracked *TrackedCluster) Response() *Cluster {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, err := senlin.TrackedDecode(tracked.response.Body, &tracked.response.Metadata, clusterLifecycleMetadata)
	if err != nil || normalizeTrackedCluster(value, false) != nil {
		return nil
	}
	value.Operation = tracked.response.Operation.Snapshot()
	return value
}

func (tracked *TrackedCluster) Dirty() bool {
	return tracked != nil && tracked.state != nil && tracked.state.Dirty()
}

// Edit atomically applies concrete UpdateOpts or owned extension fields without
// HTTP. Headers belong to Commit. Only a pending profile_only field, including
// removal, requires microversion 1.6 when Commit sends its PATCH.
func (tracked *TrackedCluster) Edit(opts UpdateOpts, options ...UpdateOption) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked cluster is required", resource.ErrInvalidOption)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil {
		err = senlin.ValidateTrackedFields[Cluster](config.Fields)
	}
	if err == nil {
		if name, set := config.Options.Name.Get(); set {
			err = validateName(name)
		}
	}
	if err == nil {
		if profile, set := config.Options.ProfileID.Get(); set {
			err = senlin.Required(profile)
		}
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Config, "config")
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	forbidden := append(append([]string(nil), responseFields...), "min_size", "max_size", "desired_capacity", "is_profile_only")
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
		body, err = senlin.CommandBody(config, "cluster", forbidden...)
	}
	if err != nil {
		return request.Wrap("Edit", "clustering.clusters", err)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return request.Wrap("Edit", "clustering.clusters", err)
	}
	return request.Wrap("Edit", "clustering.clusters", tracked.state.Edit(envelope["cluster"]))
}

func (tracked *TrackedCluster) remove(key string) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked cluster is required", resource.ErrInvalidOption)
	}
	tracked.state.Remove(key)
	return nil
}

// RemoveName records a name:null PATCH while removing the cached field.
func (tracked *TrackedCluster) RemoveName() error { return tracked.remove("name") }

// RemoveProfileID records a profile_id:null PATCH. The server owns null meaning.
func (tracked *TrackedCluster) RemoveProfileID() error { return tracked.remove("profile_id") }

// RemoveTimeout records a timeout:null PATCH while removing the cached field.
func (tracked *TrackedCluster) RemoveTimeout() error { return tracked.remove("timeout") }

// RemoveConfig records a config:null PATCH. It does not imply server deletion.
func (tracked *TrackedCluster) RemoveConfig() error { return tracked.remove("config") }

// RemoveMetadata records a metadata:null PATCH. Senlin may treat null as omitted;
// Edit with an empty metadata object requests whole-object replacement instead.
func (tracked *TrackedCluster) RemoveMetadata() error { return tracked.remove("metadata") }

// RemoveProfileOnly records profile_only:null and requires 1.6 when sent.
func (tracked *TrackedCluster) RemoveProfileOnly() error { return tracked.remove("profile_only") }

func clusterCommitHeaders(options []UpdateOption) (map[string]string, error) {
	config, err := request.Apply(UpdateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil && (config.Options.Name.IsSet() || config.Options.ProfileID.IsSet() || config.Options.Timeout.IsSet() || len(config.Options.Config) > 0 || len(config.Options.Metadata) > 0 || config.Options.ProfileOnly != nil || config.Options.profileOnlyNull) {
		err = fmt.Errorf("%w: Commit options only support headers; call Edit for body changes", resource.ErrInvalidOption)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	return maps.Clone(config.Headers), err
}

// Commit submits dirty fields once, accepts a valid 202 response and clears only
// its revisions. Clean commits validate context/client and return without HTTP.
// Acceptance is not completion. A successful PATCH followed by a ResponseError
// leaves dirty state intact; callers can Refresh before deciding to resend.
func (tracked *TrackedCluster) Commit(ctx context.Context, options ...UpdateOption) (*Cluster, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked cluster is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	headers, err := clusterCommitHeaders(options)
	client := tracked.api.RawClient()
	if err == nil {
		err = senlin.Validate(ctx, client)
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.clusters", err)
	}
	pending := tracked.state.Pending()
	if len(pending.Body) == 0 {
		return tracked.Value(), nil
	}
	if _, gated := pending.Body["profile_only"]; gated {
		if err := senlin.RequireVersion(ctx, client, 6); err != nil {
			return nil, request.Wrap("Commit", "clustering.clusters", err)
		}
	}
	body, err := json.Marshal(map[string]any{"cluster": pending.Body})
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.clusters", err)
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Commit", "clustering.clusters", err)
	}
	if _, gated := pending.Body["profile_only"]; gated {
		if err := senlin.RequireVersion(ctx, client, 6); err != nil {
			return nil, request.Wrap("Commit", "clustering.clusters", err)
		}
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL(tracked.collectionPath, url.PathEscape(tracked.id)), json.RawMessage(body), headers, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.clusters", err)
	}
	value, err := decodeMutation(client, response, true)
	if err == nil {
		if err = tracked.accept(pending, value); err != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.clusters", err)
	}
	return tracked.Value(), nil
}

// Refresh GETs the fixed route, merges observed fields and clears the dirty
// revisions present before GET. Newer edits survive. A valid GET clears the last
// submission without fetching or waiting for its action; failure changes nothing.
func (tracked *TrackedCluster) Refresh(ctx context.Context) (*Cluster, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked cluster is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	pending := tracked.state.Pending()
	client := tracked.api.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Refresh", "clustering.clusters", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL(tracked.collectionPath, url.PathEscape(tracked.id)), nil, nil, http.StatusOK)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			err = &resource.NotFoundError{Resource: "clustering.clusters", Reference: tracked.id, Cause: err}
		}
		return nil, request.Wrap("Refresh", "clustering.clusters", err)
	}
	value, err := rest.Decode(response, "cluster", clusterLifecycleMetadata)
	if err == nil {
		if err = tracked.accept(pending, value); err != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, request.Wrap("Refresh", "clustering.clusters", err)
	}
	return tracked.Value(), nil
}

func (tracked *TrackedCluster) accept(pending senlin.TrackedPatch, value *Cluster) error {
	owned, err := senlin.TrackedDecode(value.Body, &value.Metadata, clusterLifecycleMetadata)
	if err == nil {
		err = normalizeTrackedCluster(owned, false)
	}
	if err != nil {
		return err
	}
	owned.Operation = value.Operation.Snapshot()
	tracked.responseMu.Lock()
	defer tracked.responseMu.Unlock()
	if err := tracked.state.Accept(pending, owned.Body); err != nil {
		return err
	}
	tracked.response = owned
	return nil
}
