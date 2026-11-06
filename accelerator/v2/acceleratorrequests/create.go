package acceleratorrequests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/common"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cyborg"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type CreateOpts struct {
	DeviceProfileName string `json:"device_profile_name"`
}

// CreateOption is one option of Create: WithCreateOptions, request.Field (an
// extension body member that cannot replace a typed one), request.Header or
// request.Custom over *request.Config[CreateOpts]. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = cyborg.FieldHeaderCalls[CreateOpts]()

func WithCreateOptions(opts CreateOpts) CreateOption {
	return request.Custom(request.WithOptions(opts))
}

// CreateResponse retains every request created for the profile, including an
// empty result. Body stores the envelope; each request stores its own raw fields.
type CreateResponse struct {
	common.Metadata
	Requests []*AcceleratorRequest
	RawBody  json.RawMessage
}

// Create may produce multiple requests. An accepted response with malformed
// resource data returns the response metadata and any decoded requests together
// with an error. It adds no retry or cleanup of accepted resources; the native
// provider's configured authentication and HTTP retry hooks still apply.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*CreateResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, err
	}
	c, err := calls.ApplyConfig(opts, createCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(c, true, false, true)
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.Options.DeviceProfileName) == "" {
		return nil, fmt.Errorf("%w: device profile name is required", resource.ErrInvalidOption)
	}
	body, err := request.MergeFieldsFor(map[string]any{"device_profile_name": c.Options.DeviceProfileName}, c.Fields, c.Options)
	if err != nil {
		return nil, err
	}
	headers, err := cyborg.Headers(c.Headers)
	if err != nil {
		return nil, err
	}
	snapshot, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	raw, meta, err := cyborg.JSONResponse(ctx, a.client, "POST", a.client.ServiceURL("accelerator_requests"), json.RawMessage(snapshot), headers, 201)
	if meta == nil || meta.StatusCode != 201 {
		return nil, request.Wrap("create", "arqs", err)
	}
	result := &CreateResponse{Metadata: *meta, Requests: make([]*AcceleratorRequest, 0), RawBody: append(json.RawMessage(nil), raw...)}
	if err != nil {
		return result, request.Wrap("create", "arqs", err)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return result, request.Wrap("create", "arqs", fmt.Errorf("Cyborg ARQ creation response must be an object"))
	}
	if err := json.Unmarshal(raw, &result.Body); err != nil {
		return result, request.Wrap("create", "arqs", err)
	}
	batch := bytes.TrimSpace(result.Body["arqs"])
	if len(batch) == 0 || batch[0] != '[' {
		return result, request.Wrap("create", "arqs", fmt.Errorf("Cyborg ARQ creation response must contain arqs array"))
	}
	var items []json.RawMessage
	if err := json.Unmarshal(batch, &items); err != nil {
		return result, request.Wrap("create", "arqs", err)
	}
	var decodeErrors []error
	for i, item := range items {
		var v AcceleratorRequest
		if err := json.Unmarshal(item, &v); err != nil {
			decodeErrors = append(decodeErrors, fmt.Errorf("request %d: %w", i, err))
			continue
		}
		v.Header, v.StatusCode = result.Header.Clone(), result.StatusCode
		result.Requests = append(result.Requests, &v)
	}
	return result, request.Wrap("create", "arqs", errors.Join(decodeErrors...))
}
