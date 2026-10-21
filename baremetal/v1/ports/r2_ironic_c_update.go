package ports

import (
	"context"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/ironicwire"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// PatchOp is one JSON Patch operation; Value is omitted only when nil.
type PatchOp = ironicwire.PatchOp

func portNames() ironicwire.Names {
	names := ironicwire.Names{Attributes: map[string]string{}, ID: "uuid"}
	for _, f := range listRecordProfile().Body {
		names.Attributes[f.Name] = f.Wire
	}
	return names
}

func portInvalid(op, format string, args ...any) error {
	return request.Wrap(op, "ports", fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...)))
}

// PatchResult holds the port returned by PatchPort. Port is
// nil and Sent false when the patch was empty and nothing was sent.
type PatchResult struct {
	Port *Port
	Sent bool
}

// PatchPort sends PATCH /ports/{id} (the client microversion is sent unchanged). Path first segments are
// converted from attribute names to wire names, an operation without op or with
// an empty path fails before any request, and an empty patch sends nothing. Only
// 200 succeeds and a conflict is never retried.
func (a *API) PatchPort(ctx context.Context, id string, patch []PatchOp) (*PatchResult, error) {
	const op = "PatchPort"
	if id == "" {
		return nil, portInvalid(op, "port ID must not be empty")
	}
	ops, err := ironicwire.ConvertPatch(portNames(), patch)
	if err != nil {
		return nil, request.Wrap(op, "ports", err)
	}
	if len(ops) == 0 {
		return &PatchResult{}, nil
	}
	return a.send(ctx, op, id, ops)
}

// UpdateAttributes derives the patch of the pinned update_port(id, **attrs):
// every attribute becomes an "add" operation sorted by path, because only the
// identity is known as the original body. Unknown names and the identity field
// fail before any request. An empty attribute map sends nothing.
func (a *API) UpdateAttributes(ctx context.Context, id string, attrs map[string]any) (*PatchResult, error) {
	const op = "UpdateAttributes"
	if id == "" {
		return nil, portInvalid(op, "port ID must not be empty")
	}
	resolved, err := ironicwire.ResolveAttributes(portNames(), attrs)
	if err != nil {
		return nil, request.Wrap(op, "ports", err)
	}
	if len(resolved) == 0 {
		return &PatchResult{}, nil
	}
	return a.send(ctx, op, id, ironicwire.AttributePatch(resolved))
}

func (a *API) send(ctx context.Context, op, id string, ops []PatchOp) (*PatchResult, error) {
	var body struct {
		Port
	}
	if _, err := a.client.Patch(ctx, a.client.ServiceURL("ports", url.PathEscape(id)), ops, &body.Port, &gophercloud.RequestOpts{OkCodes: []int{200}}); err != nil {
		return nil, request.Wrap(op, "ports", err)
	}
	value := body.Port
	return &PatchResult{Port: &value, Sent: true}, nil
}
