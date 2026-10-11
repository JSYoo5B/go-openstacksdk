package floatingips

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// AttributeOption configures the dedicated pinned Source mutation profile.
// Body values are JSON snapshots; revision and If-Match stay outside the Body.
type AttributeOption = request.Option[struct{}]

type neutronIfMatchAttribute struct {
	encoded json.RawMessage
	err     error
}

// WithAttribute maps canonical Python Body names to wire names and snapshots
// supplied JSON. Unknown extension keys stay in Body. The inherited if_match
// Header accepts JSON strings/string lists and null (the Source string "None").
// Raw Header validation is deferred because Source revision overrides it first.
func WithAttribute(key string, value any) AttributeOption {
	if strings.EqualFold(key, "if_match") || strings.EqualFold(key, "if-match") {
		encoded, err := json.Marshal(value)
		return func(config *request.Config[struct{}]) error {
			config.Arguments["if_match"] = neutronIfMatchAttribute{encoded: append(json.RawMessage(nil), encoded...), err: err}
			return nil
		}
	}
	if wire, ok := neutronBodyFieldNames[key]; ok {
		key = wire
	}
	return request.WithField[struct{}](key, value)
}

// WithIfMatch snapshots Header values using the Source comma-space list form.
// An explicit empty list sends an empty header rather than omitting it.
func WithIfMatch(values ...string) AttributeOption {
	return WithAttribute("if_match", strings.Join(append([]string(nil), values...), ", "))
}

// WithRevision selects the Source update if_revision argument, including zero.
// It is valid only for UpdateWithAttributes and overrides a raw If-Match option
// regardless of their relative order. Later revision options win.
func WithRevision(value int64) AttributeOption {
	return request.WithArgument[struct{}]("if_revision", value)
}

func neutronMutationContextError(ctx context.Context, err error) error {
	cause := context.Cause(ctx)
	if cause == ctx.Err() {
		cause = nil
	}
	return errors.Join(err, ctx.Err(), cause)
}

func neutronMutationAttributes(ctx context.Context, client *gophercloud.ServiceClient, update bool, options []AttributeOption) (map[string]any, map[string]string, error) {
	if ctx == nil || client == nil || client.ProviderClient == nil {
		return nil, nil, fmt.Errorf("%w: a context and configured client are required", resource.ErrInvalidOption)
	}
	if ctx.Err() != nil {
		return nil, nil, neutronMutationContextError(ctx, nil)
	}
	config, err := request.Apply(struct{}{}, append([]AttributeOption(nil), options...)...)
	allowed := []string{"if_match"}
	if update {
		allowed = append(allowed, "if_revision")
	}
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true, allowed...)
	}
	if err != nil {
		if ctx.Err() != nil {
			err = neutronMutationContextError(ctx, err)
		}
		return nil, nil, err
	}
	if ctx.Err() != nil {
		return nil, nil, neutronMutationContextError(ctx, nil)
	}
	for _, key := range []string{"base_path", "microversion", "connection", "_synchronized", "resource_type", "__conflicting_attrs", "location"} {
		if _, present := config.Fields[key]; present {
			return nil, nil, fmt.Errorf("%w: %q is a Resource control attribute, not a Body extension", resource.ErrInvalidOption, key)
		}
	}
	for key := range config.Fields {
		if strings.EqualFold(key, "if_match") || strings.EqualFold(key, "if-match") {
			return nil, nil, fmt.Errorf("%w: If-Match is a Header and cannot be set as a raw Body field", resource.ErrInvalidOption)
		}
	}
	if update {
		for _, key := range []string{"id", "if_revision"} {
			if _, present := config.Fields[key]; present {
				return nil, nil, fmt.Errorf("%w: %q must be supplied by the update ID or revision option", resource.ErrInvalidOption, key)
			}
		}
	}
	headers := map[string]string{}
	for key, value := range config.Headers {
		if !strings.EqualFold(key, "If-Match") || len(headers) != 0 {
			return nil, nil, fmt.Errorf("%w: only one If-Match Header is supported", resource.ErrInvalidOption)
		}
		headers["If-Match"] = value
	}
	revisionPresent := false
	if update {
		revision, present, err := request.Argument[int64](config, "if_revision")
		if err != nil {
			return nil, nil, err
		}
		revisionPresent = present
		if present {
			headers["If-Match"] = "revision_number=" + strconv.FormatInt(revision, 10)
		}
	}
	// Proxy._update replaces attrs.if_match before Resource construction. A
	// discarded raw Header must not fail parsing or validation in this branch.
	if !revisionPresent {
		raw, present, err := request.Argument[neutronIfMatchAttribute](config, "if_match")
		if err != nil {
			return nil, nil, err
		}
		if present {
			if raw.err != nil {
				return nil, nil, fmt.Errorf("%w: If-Match: %v", resource.ErrInvalidOption, raw.err)
			}
			encoded := append(json.RawMessage(nil), raw.encoded...)
			header := ""
			if string(encoded) == "null" {
				header = "None"
			} else if len(encoded) > 0 && encoded[0] == '"' {
				if err := json.Unmarshal(encoded, &header); err != nil {
					return nil, nil, fmt.Errorf("%w: If-Match: %v", resource.ErrInvalidOption, err)
				}
			} else if len(encoded) > 0 && encoded[0] == '[' {
				var elements []json.RawMessage
				if err := json.Unmarshal(encoded, &elements); err != nil {
					return nil, nil, fmt.Errorf("%w: If-Match requires a string or string list", resource.ErrInvalidOption)
				}
				values := make([]string, len(elements))
				for index, element := range elements {
					if len(element) == 0 || element[0] != '"' {
						return nil, nil, fmt.Errorf("%w: If-Match list elements must be strings", resource.ErrInvalidOption)
					}
					if err := json.Unmarshal(element, &values[index]); err != nil {
						return nil, nil, fmt.Errorf("%w: If-Match: %v", resource.ErrInvalidOption, err)
					}
				}
				header = strings.Join(values, ", ")
			} else {
				return nil, nil, fmt.Errorf("%w: If-Match requires a string or string list", resource.ErrInvalidOption)
			}
			headers["If-Match"] = header
		}
	}
	if strings.ContainsAny(headers["If-Match"], "\r\n") {
		return nil, nil, fmt.Errorf("%w: invalid If-Match Header", resource.ErrInvalidOption)
	}
	body, err := request.MergeFields(map[string]any{}, config.Fields)
	return body, headers, err
}

func neutronMutationClient(client *gophercloud.ServiceClient, headers map[string]string) *gophercloud.ServiceClient {
	captured := *client
	captured.MoreHeaders = maps.Clone(client.MoreHeaders)
	if _, present := headers["If-Match"]; present {
		for key := range captured.MoreHeaders {
			if strings.EqualFold(key, "If-Match") {
				delete(captured.MoreHeaders, key)
			}
		}
	}
	return &captured
}

func neutronMutationCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

// CreateWithAttributes posts only supplied Source Body attributes and headers.
// No attributes sends an empty resource envelope; Neutron validates required
// fields. Native Create and result decoding remain unchanged. In particular a
// native 204 result is an ID-less zero model; no request values are synthesized.
func (a *API) CreateWithAttributes(ctx context.Context, options ...AttributeOption) (*FloatingIP, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	attributes, headers, err := neutronMutationAttributes(ctx, client, false, options)
	if err != nil {
		return nil, request.Wrap("CreateWithAttributes", "floatingips", err)
	}
	client = neutronMutationClient(client, headers)
	var result CreateResult
	response, err := client.Post(ctx, client.ServiceURL("floatingips"), map[string]any{"floatingip": attributes}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes(), MoreHeaders: headers})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	value, err := result.Extract()
	return value, request.Wrap("CreateWithAttributes", "floatingips", err)
}

// UpdateWithAttributes commits Body/header changes to a literal ID. No options
// sends no HTTP and returns an ID-only native model. A revision or If-Match
// header alone is a change and sends an empty resource envelope in a PUT.
func (a *API) UpdateWithAttributes(ctx context.Context, id string, options ...AttributeOption) (*FloatingIP, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	attributes, headers, err := neutronMutationAttributes(ctx, client, true, options)
	if err != nil {
		return nil, request.Wrap("UpdateWithAttributes", "floatingips", err)
	}
	if len(attributes) == 0 && len(headers) == 0 {
		return &FloatingIP{ID: id}, nil
	}
	client = neutronMutationClient(client, headers)
	var result UpdateResult
	response, err := client.Put(ctx, client.ServiceURL("floatingips", id), map[string]any{"floatingip": attributes}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes(), MoreHeaders: headers})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	value, err := result.Extract()
	return value, request.Wrap("UpdateWithAttributes", "floatingips", err)
}
