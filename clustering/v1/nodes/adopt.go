package nodes

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Adopt synchronously adopts a physical resource at microversion 1.7 or newer.
// Location, if supplied, is retained as HTTP evidence and is not an action.
func (a *API) Adopt(ctx context.Context, value AdoptOpts, options ...AdoptOption) (*Node, error) {
	client := a.RawClient()
	if err := senlin.RequireVersion(ctx, client, 7); err != nil {
		return nil, request.Wrap("Adopt", "clustering.nodes", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = senlin.Required(config.Options.Identity, config.Options.Type)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Overrides, "overrides")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.FlatBody(config)
	}
	if err != nil {
		return nil, request.Wrap("Adopt", "clustering.nodes", err)
	}
	headers := maps.Clone(config.Headers)
	if err := senlin.RequireVersion(ctx, client, 7); err != nil {
		return nil, request.Wrap("Adopt", "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPost, client.ServiceURL("nodes", "adopt"), body, headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Adopt", "clustering.nodes", err)
	}
	result, err := rest.Decode(response, "node", func(value *Node) *resource.Metadata { return &value.Metadata })
	return result, request.Wrap("Adopt", "clustering.nodes", err)
}

// AdoptPreviewSpec preserves the proposed profile and its additional fields.
// Version is raw JSON because published examples include a numeric version.
type AdoptPreviewSpec struct {
	resource.Metadata
	Type       string                     `json:"type"`
	Version    json.RawMessage            `json:"version"`
	Properties map[string]json.RawMessage `json:"properties"`
}

func (value *AdoptPreviewSpec) UnmarshalJSON(data []byte) error {
	type plain AdoptPreviewSpec
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = AdoptPreviewSpec(decoded)
	return nil
}

// AdoptPreviewResult retains the entire response envelope, proposed profile,
// and HTTP evidence. It does not create a Node or imply an asynchronous action.
type AdoptPreviewResult struct {
	resource.Metadata
	Spec    AdoptPreviewSpec `json:"node_preview"`
	RawBody json.RawMessage  `json:"-"`
}

func (value *AdoptPreviewResult) UnmarshalJSON(data []byte) error {
	type plain AdoptPreviewResult
	var decoded plain
	if err := resource.DecodeObject(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*value = AdoptPreviewResult(decoded)
	return nil
}

// AdoptPreview sends exactly identity, type, overrides and snapshot. Additional
// body/query/argument extensions are rejected rather than silently discarded.
func (a *API) AdoptPreview(ctx context.Context, value AdoptPreviewOpts, options ...AdoptPreviewOption) (*AdoptPreviewResult, error) {
	client := a.RawClient()
	if err := senlin.RequireVersion(ctx, client, 7); err != nil {
		return nil, request.Wrap("AdoptPreview", "clustering.nodes", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = senlin.Required(config.Options.Identity, config.Options.Type)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Overrides, "overrides")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.FlatBody(config)
	}
	if err != nil {
		return nil, request.Wrap("AdoptPreview", "clustering.nodes", err)
	}
	headers := maps.Clone(config.Headers)
	if err := senlin.RequireVersion(ctx, client, 7); err != nil {
		return nil, request.Wrap("AdoptPreview", "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPost, client.ServiceURL("nodes", "adopt-preview"), body, headers, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("AdoptPreview", "clustering.nodes", err)
	}
	if _, err := response.Object("node_preview"); err != nil {
		return nil, request.Wrap("AdoptPreview", "clustering.nodes", err)
	}
	result, err := rest.Decode(response, "", func(value *AdoptPreviewResult) *resource.Metadata { return &value.Metadata })
	if err == nil {
		result.RawBody = append(json.RawMessage(nil), response.Body...)
	}
	return result, request.Wrap("AdoptPreview", "clustering.nodes", err)
}
