package nodes

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

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// TrackedNode owns a cached body, a sticky dirty set and an immutable request ID.
// Value returns the merged cache; Response returns the last successful HTTP
// fields. Both preserve independently owned asynchronous submission evidence.
type TrackedNode struct {
	api            *API
	id             string
	collectionPath string
	state          *senlin.TrackedState
	opMu           sync.Mutex
	responseMu     sync.RWMutex
	response       *Node
}

// Track takes ownership of a cached model without HTTP. Body is authoritative
// when present, including its exact lowercase id; typed edits to that input do
// not change the tracked cache. A Body-less model supplies a local JSON seed.
func (a *API) Track(value *Node) (*TrackedNode, error) {
	return a.trackNode(value, "", "nodes")
}

func (a *API) trackNode(value *Node, routeID, path string) (*TrackedNode, error) {
	if value == nil {
		return nil, request.Wrap("Track", "clustering.nodes", fmt.Errorf("%w: node is required", resource.ErrInvalidOption))
	}
	body, err := senlin.TrackedSeed(value, &value.Metadata)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.nodes", err)
	}
	state, err := senlin.NewTrackedState(body)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.nodes", err)
	}
	response, err := decodeTrackedNode(body, &value.Metadata, value.Operation)
	if err != nil {
		return nil, request.Wrap("Track", "clustering.nodes", err)
	}
	if routeID == "" {
		routeID = response.ID
	}
	if err := senlin.Identifier(routeID); err != nil {
		return nil, request.Wrap("Track", "clustering.nodes", err)
	}
	return &TrackedNode{api: a, id: routeID, collectionPath: path, state: state, response: response}, nil
}

// Load performs one strict explicit-reference lookup. An ID reference fixes
// the route to that input even when the returned id changes or is absent/null;
// a Name reference captures the returned canonical lowercase id once.
func (a *API) Load(ctx context.Context, ref resource.Ref) (*TrackedNode, error) {
	return a.loadAt(ctx, ref, "nodes")
}

func (a *API) loadAt(ctx context.Context, ref resource.Ref, path string) (*TrackedNode, error) {
	collection := spec(a.RawClient())
	collection.Path = path
	collection.ValidateItem = func(value *Node) error {
		var err error
		value.ID, err = trackedNodeString(value.Body, "id")
		if err != nil {
			return err
		}
		if ref.IsName() {
			if err := senlin.Identifier(value.ID); err != nil {
				return err
			}
		}
		value.Name, err = trackedNodeString(value.Body, "name")
		return err
	}
	collection.Name = func(value *Node) string {
		if value == nil {
			return ""
		}
		name, _ := trackedNodeString(value.Body, "name")
		return name
	}
	value, err := rest.Collection(collection).Find(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Load", "clustering.nodes", err)
	}
	routeID := ""
	if !ref.IsName() {
		routeID = ref.String()
	}
	return a.trackNode(value, routeID, path)
}

func nodeMetadata(value *Node) *resource.Metadata { return &value.Metadata }

func decodeTrackedNode(body map[string]json.RawMessage, evidence *resource.Metadata, operation *actions.Submission) (*Node, error) {
	value, err := senlin.TrackedDecode(body, evidence, nodeMetadata)
	if err != nil {
		return nil, err
	}
	// Canonical raw identity controls the typed cache and constructor route.
	// Missing/null id must not inherit an encoding/json case-variant shadow.
	value.ID, err = trackedNodeString(body, "id")
	if err != nil {
		return nil, err
	}
	value.Name, err = trackedNodeString(body, "name")
	if err != nil {
		return nil, err
	}
	value.Operation = operation.Snapshot()
	return value, nil
}

func trackedNodeString(body map[string]json.RawMessage, key string) (string, error) {
	var value string
	if raw, exists := body[key]; exists {
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", fmt.Errorf("%w: tracked node %s must be a string or null", resource.ErrInvalidOption, key)
		}
		if key == "id" && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			if err := senlin.Identifier(value); err != nil {
				return "", err
			}
		}
	}
	return value, nil
}

func (tracked *TrackedNode) Value() *Node {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, _ := decodeTrackedNode(tracked.state.Body(), &tracked.response.Metadata, tracked.response.Operation)
	return value
}

func (tracked *TrackedNode) Response() *Node {
	if tracked == nil || tracked.state == nil {
		return nil
	}
	tracked.responseMu.RLock()
	defer tracked.responseMu.RUnlock()
	value, _ := decodeTrackedNode(tracked.response.Body, &tracked.response.Metadata, tracked.response.Operation)
	return value
}

func (tracked *TrackedNode) Dirty() bool {
	return tracked != nil && tracked.state != nil && tracked.state.Dirty()
}

// Edit validates owned mutable inputs locally and applies them atomically.
// Name/null presence and metadata object/null rules match stateless Update.
// Headers belong to Commit; the tainted microversion gate belongs to HTTP.
func (tracked *TrackedNode) Edit(opts UpdateOpts, options ...UpdateOption) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked node is required", resource.ErrInvalidOption)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil {
		err = senlin.ValidateTrackedFields[Node](config.Fields)
	}
	if err == nil {
		for key := range config.Fields {
			for _, protected := range append(append([]string(nil), responseFields...), "cluster_id") {
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
	if err == nil && config.Options.Name.IsSet() && !config.Options.Name.IsNull() {
		name, _ := config.Options.Name.Get()
		err = validateName(name)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, "node", append(append([]string(nil), responseFields...), "cluster_id")...)
	}
	if err != nil {
		return request.Wrap("Edit", "clustering.nodes", err)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return request.Wrap("Edit", "clustering.nodes", err)
	}
	return request.Wrap("Edit", "clustering.nodes", tracked.state.Edit(envelope["node"]))
}

func (tracked *TrackedNode) remove(key string) error {
	if tracked == nil || tracked.state == nil {
		return fmt.Errorf("%w: tracked node is required", resource.ErrInvalidOption)
	}
	tracked.state.Remove(key)
	return nil
}

// RemoveName deletes the cached field and records name:null for Commit.
func (tracked *TrackedNode) RemoveName() error { return tracked.remove("name") }

// RemoveProfileID deletes the cached field and records profile_id:null.
func (tracked *TrackedNode) RemoveProfileID() error { return tracked.remove("profile_id") }

// RemoveRole deletes the cached field and records role:null.
func (tracked *TrackedNode) RemoveRole() error { return tracked.remove("role") }

// RemoveMetadata deletes the cached field and records metadata:null. An Edit
// with WithUpdateMetadata(nil) instead retains a present null in the cache.
func (tracked *TrackedNode) RemoveMetadata() error { return tracked.remove("metadata") }

// RemoveTainted records a tainted:null mutation, which requires 1.13 on Commit.
func (tracked *TrackedNode) RemoveTainted() error { return tracked.remove("tainted") }

func nodeCommitHeaders(options []UpdateOption) (map[string]string, error) {
	config, err := request.Apply(UpdateOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil && (config.Options.Name.IsSet() || config.Options.ProfileID.IsSet() || config.Options.Role.IsSet() || config.Options.Tainted.IsSet() || len(config.Options.Metadata) > 0) {
		err = fmt.Errorf("%w: Commit options only support headers; call Edit for body changes", resource.ErrInvalidOption)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	return maps.Clone(config.Headers), err
}

// Commit sends the dirty snapshot to the fixed route and retains the accepted
// action alongside response fields. A clean call validates input/source/context
// and returns the cache without HTTP. Acceptance clears only submitted revisions;
// errors retain dirty state and never cause an automatic mutation resend.
func (tracked *TrackedNode) Commit(ctx context.Context, options ...UpdateOption) (*Node, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked node is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	headers, err := nodeCommitHeaders(options)
	client := tracked.api.RawClient()
	if err == nil {
		err = senlin.Validate(ctx, client)
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.nodes", err)
	}
	pending := tracked.state.Pending()
	if len(pending.Body) == 0 {
		return tracked.Value(), nil
	}
	body, err := json.Marshal(map[string]any{"node": pending.Body})
	if err == nil {
		err = senlin.Validate(ctx, client)
	}
	if err == nil {
		if _, changesTainted := pending.Body["tainted"]; changesTainted {
			err = senlin.RequireVersion(ctx, client, 13)
		}
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, client.ServiceURL(tracked.collectionPath, url.PathEscape(tracked.id)), json.RawMessage(body), headers, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.nodes", err)
	}
	value, err := decodeMutation(client, response, true)
	if err == nil {
		err = tracked.accept(pending, value)
		if err != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, request.Wrap("Commit", "clustering.nodes", err)
	}
	return tracked.Value(), nil
}

// Refresh GETs the fixed ID, merges valid fields and resets revisions present
// before GET while preserving newer edits. A successful synchronous read clears
// the last accepted Operation; a failure leaves cache and evidence untouched.
func (tracked *TrackedNode) Refresh(ctx context.Context) (*Node, error) {
	if tracked == nil || tracked.state == nil {
		return nil, fmt.Errorf("%w: tracked node is required", resource.ErrInvalidOption)
	}
	tracked.opMu.Lock()
	defer tracked.opMu.Unlock()
	client := tracked.api.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Refresh", "clustering.nodes", err)
	}
	pending := tracked.state.Pending()
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL(tracked.collectionPath, url.PathEscape(tracked.id)), nil, nil, http.StatusOK)
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "clustering.nodes", Reference: tracked.id, Cause: err}
	}
	if err != nil {
		return nil, request.Wrap("Refresh", "clustering.nodes", err)
	}
	value, err := rest.Decode(response, "node", nodeMetadata)
	if err == nil {
		value.Operation = nil
		err = tracked.accept(pending, value)
		if err != nil {
			err = response.Fail(err)
		}
	}
	if err != nil {
		return nil, request.Wrap("Refresh", "clustering.nodes", err)
	}
	return tracked.Value(), nil
}

func (tracked *TrackedNode) accept(pending senlin.TrackedPatch, value *Node) error {
	owned, err := decodeTrackedNode(value.Body, &value.Metadata, value.Operation)
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
