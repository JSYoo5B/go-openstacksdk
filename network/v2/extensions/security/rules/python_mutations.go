package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// AttributeOption configures the dedicated pinned Source mutation profile.
// Body values are JSON snapshots; inherited If-Match stays outside the Body.
type AttributeOption = request.Option[struct{}]

type neutronIfMatchAttribute struct {
	encoded json.RawMessage
	err     error
}

// WithAttribute maps canonical Python Body names to wire names and snapshots
// supplied JSON. Unknown extension keys stay in Body. The inherited if_match
// Header accepts JSON strings/string lists and null (the Source string "None").
// Raw Header parsing follows option application using an independent snapshot.
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

func neutronMutationContextError(ctx context.Context, err error) error {
	cause := context.Cause(ctx)
	if cause == ctx.Err() {
		cause = nil
	}
	return errors.Join(err, ctx.Err(), cause)
}

func neutronMutationAttributes(ctx context.Context, client *gophercloud.ServiceClient, options []AttributeOption) (map[string]any, map[string]string, error) {
	if ctx == nil || client == nil || client.ProviderClient == nil {
		return nil, nil, fmt.Errorf("%w: a context and configured client are required", resource.ErrInvalidOption)
	}
	if ctx.Err() != nil {
		return nil, nil, neutronMutationContextError(ctx, nil)
	}
	config, err := request.Apply(struct{}{}, append([]AttributeOption(nil), options...)...)
	allowed := []string{"if_match"}
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
	headers := map[string]string{}
	for key, value := range config.Headers {
		if !strings.EqualFold(key, "If-Match") || len(headers) != 0 {
			return nil, nil, fmt.Errorf("%w: only one If-Match Header is supported", resource.ErrInvalidOption)
		}
		headers["If-Match"] = value
	}
	{
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

// neutronSingleRuleBody preserves the pinned SecurityGroupRule override:
// a falsy remote_address_group_id is removed only for singular create. Bulk
// calls _prepare_request without a singular envelope and keeps those values.
func neutronSingleRuleBody(attributes map[string]any) (map[string]any, error) {
	body := maps.Clone(attributes)
	raw, present := body["remote_address_group_id"]
	if !present {
		return body, nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: remote_address_group_id: %v", resource.ErrInvalidOption, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%w: remote_address_group_id: %v", resource.ErrInvalidOption, err)
	}
	falsy := false
	switch value := value.(type) {
	case nil:
		falsy = true
	case bool:
		falsy = !value
	case string:
		falsy = value == ""
	case []any:
		falsy = len(value) == 0
	case map[string]any:
		falsy = len(value) == 0
	case json.Number:
		// Every valid JSON number with an all-zero mantissa is zero. Do not
		// round nonzero high-precision or small-exponent extension values.
		mantissa := strings.SplitN(strings.SplitN(string(value), "e", 2)[0], "E", 2)[0]
		falsy = true
		for _, digit := range mantissa {
			if digit >= '1' && digit <= '9' {
				falsy = false
				break
			}
		}
	}
	if falsy {
		delete(body, "remote_address_group_id")
	}
	return body, nil
}

// CreateWithAttributes sends supplied Source attrs without native required-field
// checks. A falsy remote_address_group_id is omitted for pinned compatibility.
// Native extraction is retained; a bodyless 204 returns nil,nil.
func (a *API) CreateWithAttributes(ctx context.Context, options ...AttributeOption) (*SecGroupRule, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	attributes, headers, err := neutronMutationAttributes(ctx, client, options)
	if err == nil {
		attributes, err = neutronSingleRuleBody(attributes)
	}
	if err != nil {
		return nil, request.Wrap("CreateWithAttributes", "rules", err)
	}
	client = neutronMutationClient(client, headers)
	var result CreateResult
	response, err := client.Post(ctx, client.ServiceURL("security-group-rules"), map[string]any{"security_group_rule": attributes}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes(), MoreHeaders: headers})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	value, err := result.Extract()
	return value, request.Wrap("CreateWithAttributes", "rules", err)
}

// CreateBulkWithAttributes posts a nonempty sequence of raw attribute rows in
// caller order. Rows are snapshotted before callbacks. Source uses the last
// row's Header, and bulk keeps falsy remote_address_group_id values. The eager
// native slice result does not synthesize partial or bodyless Resource models.
func (a *API) CreateBulkWithAttributes(ctx context.Context, data ...[]AttributeOption) ([]SecGroupRule, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	if ctx == nil || client == nil || client.ProviderClient == nil {
		return nil, request.Wrap("CreateBulkWithAttributes", "rules", fmt.Errorf("%w: a context and configured client are required", resource.ErrInvalidOption))
	}
	if ctx.Err() != nil {
		return nil, request.Wrap("CreateBulkWithAttributes", "rules", neutronMutationContextError(ctx, nil))
	}
	if len(data) == 0 {
		return nil, request.Wrap("CreateBulkWithAttributes", "rules", fmt.Errorf("%w: bulk data must contain at least one attribute row", resource.ErrInvalidOption))
	}
	rows := make([][]AttributeOption, len(data))
	for index, row := range data {
		rows[index] = append([]AttributeOption(nil), row...)
	}
	body := make([]map[string]any, len(rows))
	var headers map[string]string
	for index, row := range rows {
		attributes, rowHeaders, err := neutronMutationAttributes(ctx, client, row)
		if err != nil {
			return nil, request.Wrap("CreateBulkWithAttributes", "rules", err)
		}
		body[index], headers = attributes, rowHeaders
	}
	client = neutronMutationClient(client, headers)
	var result CreateBulkResult
	response, err := client.Post(ctx, client.ServiceURL("security-group-rules"), map[string]any{"security_group_rules": body}, &result.Body, &gophercloud.RequestOpts{OkCodes: neutronMutationCodes(), MoreHeaders: headers})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	if ctx.Err() != nil {
		result.Err = neutronMutationContextError(ctx, result.Err)
	}
	values, err := result.Extract()
	return values, request.Wrap("CreateBulkWithAttributes", "rules", err)
}
