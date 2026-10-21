package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicwire"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
)

// PatchOp is one JSON Patch operation. Value is raw JSON and omitted from the
// wire only when nil; an explicit null, false, zero or empty value is sent
// unchanged. From is sent when non-empty.
type PatchOp = ironicwire.PatchOp

func nodeNames() ironicwire.Names {
	names := ironicwire.Names{Attributes: map[string]string{}, ID: "uuid"}
	for _, field := range listRecordProfile().Body {
		names.Attributes[field.Name] = field.Wire
	}
	return names
}

// NodeAttributeWireName returns the wire name of a declared Node attribute
// (for example "instance_id" is "instance_uuid" and "is_maintenance" is
// "maintenance"); a name that is already a wire name resolves to itself.
func NodeAttributeWireName(name string) (string, bool) { return nodeNames().Server(name) }

// PatchResult is the result of PatchNode. Node is nil and Sent is false when
// the patch was empty and no request was needed.
type PatchResult struct {
	Node *Node
	Sent bool
}

// PatchNodeOption configures PatchNode.
type PatchNodeOption func(*patchNodeConfig)

type patchNodeConfig struct{ reset *bool }

// WithResetInterfaces adds the reset_interfaces query (microversion 1.45). The
// value is sent as True or False, also when false.
func WithResetInterfaces(reset bool) PatchNodeOption {
	return func(c *patchNodeConfig) { c.reset = &reset }
}

// PatchNode applies a JSON Patch with the pinned Resource.patch rules. The first
// path segment of each operation may be an attribute name or a wire name and is
// converted to the wire name; operations without op or with an empty path fail
// before any request. An empty patch sends nothing and returns an empty
// PatchResult, like the pinned source for a bare reference. With
// WithResetInterfaces the call asserts microversion 1.45 and appends the query.
// PATCH /nodes/{id} is sent once and only 200 succeeds; a 409 conflict is
// returned and never retried automatically.
func (a *API) PatchNode(ctx context.Context, id string, patch []PatchOp, options ...PatchNodeOption) (*PatchResult, error) {
	const op = "PatchNode"
	if id == "" {
		return nil, nodeInvalid(op, "node ID or name must not be empty")
	}
	var cfg patchNodeConfig
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	ops, err := ironicwire.ConvertPatch(nodeNames(), patch)
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	if len(ops) == 0 {
		return &PatchResult{}, nil
	}
	return a.sendPatch(ctx, op, id, ops, cfg.reset)
}

func (a *API) sendPatch(ctx context.Context, op, id string, ops []PatchOp, reset *bool) (*PatchResult, error) {
	client := a.client
	target := a.nodeURL(client, id)
	if reset != nil {
		version, err := ironicwire.Assert(ctx, a.client, "1.45", ironicwire.NodeCeiling, "")
		if err != nil {
			return nil, request.Wrap(op, "nodes", err)
		}
		client = ironicwire.At(a.client, version)
		target = a.nodeURL(client, id)
		value := "False"
		if *reset {
			value = "True"
		}
		target += "?" + url.Values{"reset_interfaces": {value}}.Encode()
	}
	var result upstream.UpdateResult
	resp, err := client.Patch(ctx, target, ops, &result.Body, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, result.Header, result.Err = gophercloud.ParseResponse(resp, err)
	node, err := result.Extract()
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	return &PatchResult{Node: node, Sent: true}, nil
}

// UpdateAttributes updates a node from attribute values with the pinned
// Proxy.update_node(node_id, **attrs) rules.
//
// attrs maps attribute names or wire names to JSON-encodable values; unknown
// names, the identity field and two names for one field fail before any request
// (the pinned constructor silently drops unknown names). Every attribute becomes
// an "add" operation sorted by path, because only the identity is known as the
// original body, and the patch is sent as PATCH /nodes/{id}, accepted only with
// 200. The patch is not merged with remote state.
//
// maintenance (is_maintenance) and maintenance_reason are never patched. When
// either is present the call sends PUT /nodes/{id}/maintenance with
// {"reason": value} (null when absent) if maintenance is truthy, otherwise
// DELETE /nodes/{id}/maintenance. A truthy reason without a truthy maintenance
// value is an error unless maintenance is also given, in which case the reason
// is dropped. Only 202 is accepted. When no other attribute remains the node is
// read afterwards and returned; otherwise the PATCH result is returned.
//
// A failure after the maintenance request leaves the maintenance change in
// effect remotely. Conflicts (409) are returned and never retried; the pinned
// source retries them by default.
func (a *API) UpdateAttributes(ctx context.Context, id string, attrs map[string]any) (*Node, error) {
	const op = "UpdateAttributes"
	if id == "" {
		return nil, nodeInvalid(op, "node ID or name must not be empty")
	}
	resolved, err := ironicwire.ResolveAttributes(nodeNames(), attrs)
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	return a.commitAttributes(ctx, op, id, resolved, ironicwire.AttributePatch)
}

// commitAttributes applies resolved wire-name attributes that differ from the
// original wire body. It is the shared Node.commit
// flow of UpdateAttributes and the machine update workflow.
func (a *API) commitAttributes(ctx context.Context, op, id string, dirty map[string]json.RawMessage, build func(map[string]json.RawMessage) []PatchOp) (*Node, error) {
	rawMaintenance, hasMaintenance := dirty["maintenance"]
	rawReason, hasReason := dirty["maintenance_reason"]
	rest := map[string]json.RawMessage{}
	for key, value := range dirty {
		if key != "maintenance" && key != "maintenance_reason" {
			rest[key] = value
		}
	}
	if hasMaintenance || hasReason {
		on := hasMaintenance && ironicwire.Truthy(rawMaintenance)
		if !on && hasReason && ironicwire.Truthy(rawReason) {
			if !hasMaintenance {
				return nil, nodeInvalid(op, "maintenance reason cannot be set when maintenance is False")
			}
			rawReason, hasReason = nil, false
		}
		if on {
			reason := json.RawMessage("null")
			if hasReason {
				reason = rawReason
			}
			if _, err := a.client.Put(ctx, a.nodeURL(a.client, id, "maintenance"), map[string]json.RawMessage{"reason": reason}, nil, &gophercloud.RequestOpts{OkCodes: []int{202}}); err != nil {
				return nil, request.Wrap(op, "nodes", err)
			}
		} else if _, err := a.client.Delete(ctx, a.nodeURL(a.client, id, "maintenance"), &gophercloud.RequestOpts{OkCodes: []int{202}}); err != nil {
			return nil, request.Wrap(op, "nodes", err)
		}
		if len(rest) == 0 {
			return a.getOrNil(ctx, op, id)
		}
	}
	if len(rest) == 0 {
		// Nothing to commit: the pinned commit returns the resource unchanged.
		return a.getOrNil(ctx, op, id)
	}
	result, err := a.sendPatch(ctx, op, id, build(rest), nil)
	if err != nil {
		return nil, err
	}
	return result.Node, nil
}

// CreateProvisionState is the initial provision state of CreateWithProvisionState.
type CreateProvisionState string

const (
	CreateEnroll     CreateProvisionState = "enroll"
	CreateManageable CreateProvisionState = "manageable"
	CreateAvailable  CreateProvisionState = "available"
)

var createStateVersions = map[CreateProvisionState]string{CreateEnroll: "1.11", CreateManageable: "1.4", CreateAvailable: "1.1"}

// CreateWithProvisionState creates a node with the pinned Node.create rules.
// Ironic cannot set provision_state in the body, so it is never sent: the state
// only selects the microversion rule. An empty state sends POST /nodes with the
// configured client microversion, exactly like Create. Otherwise the state must
// be enroll, manageable or available. The call asserts microversion 1.11 (enroll),
// 1.4 (manageable) or 1.1 (available, whose version is lowered to at most 1.10);
// a configured version that is too low, or an endpoint that cannot reach it,
// fails with resource.ErrUnsupported before the POST. For manageable the node is
// moved with the "manage" action and a wait when the created node is not yet
// manageable. The POST is never replayed.
func (a *API) CreateWithProvisionState(ctx context.Context, state CreateProvisionState, opts CreateOpts, options ...CreateOption) (*Node, error) {
	const op = "CreateWithProvisionState"
	if state == "" {
		return a.Create(ctx, opts, options...)
	}
	required, ok := createStateVersions[state]
	if !ok {
		return nil, nodeInvalid(op, "node's provision_state must be one of enroll, manageable, available for creation, got %q", state)
	}
	maximum := ""
	if state == CreateAvailable {
		maximum = "1.10"
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	version, err := ironicwire.Assert(ctx, a.client, required, ironicwire.NodeCeiling, maximum)
	if err != nil {
		return nil, request.Wrap(op, "nodes", fmt.Errorf("cannot create a node with initial provision state %s: %w", state, err))
	}
	client := ironicwire.At(a.client, version)
	node, err := upstream.Create(ctx, client, createOptsBuilder{base: cfg.Options, config: cfg}).Extract()
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	if state == CreateManageable && node.ProvisionState != "manageable" {
		id := node.UUID
		managed, err := a.SetProvisionState(ctx, id, ProvisionStateRequest{Target: "manage"}, WithProvisionStateWait())
		if err != nil {
			return node, request.Wrap(op, "nodes", err)
		}
		return managed, nil
	}
	return node, nil
}

// UpdateFromOriginal updates a node that was read earlier, like
// Proxy.update_node(existing_node, **attrs). original holds the wire fields of
// the earlier read (only declared fields are compared) and attrs the new
// values by attribute or wire name. Attributes whose value equals the original
// are not sent. The patch is the sorted structural difference between the
// original and the changed fields (nested objects and arrays are compared, no
// move operations are produced); maintenance fields follow the
// UpdateAttributes routing. An empty change set reads and returns the node
// without writing.
func (a *API) UpdateFromOriginal(ctx context.Context, id string, original map[string]json.RawMessage, attrs map[string]any) (*Node, error) {
	const op = "UpdateFromOriginal"
	if id == "" {
		return nil, nodeInvalid(op, "node ID or name must not be empty")
	}
	resolved, err := ironicwire.ResolveAttributes(nodeNames(), attrs)
	if err != nil {
		return nil, request.Wrap(op, "nodes", err)
	}
	dirty := map[string]json.RawMessage{}
	for wire, value := range resolved {
		if old, ok := original[wire]; ok && jsonEqual(old, value) {
			continue
		}
		dirty[wire] = value
	}
	if len(dirty) == 0 {
		return a.getOrNil(ctx, op, id)
	}
	build := func(rest map[string]json.RawMessage) []PatchOp {
		src := map[string]json.RawMessage{}
		dst := map[string]json.RawMessage{}
		for wire := range rest {
			if old, ok := original[wire]; ok {
				src[wire] = old
			}
			dst[wire] = rest[wire]
		}
		return ironicwire.Diff(src, dst)
	}
	return a.commitAttributes(ctx, op, id, dirty, build)
}

func jsonEqual(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// NodeAttributeName returns the declared attribute name (for example
// "instance_id") for an attribute or wire name.
func NodeAttributeName(name string) (string, bool) {
	for _, f := range listRecordProfile().Body {
		if f.Name == name {
			return f.Name, true
		}
	}
	for _, f := range listRecordProfile().Body {
		if f.Wire == name {
			return f.Name, true
		}
	}
	return "", false
}
