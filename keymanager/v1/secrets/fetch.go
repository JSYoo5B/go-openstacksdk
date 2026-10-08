package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Fetch always reads the metadata representation, then conditionally retrieves
// its payload from the original fixed secret URI. A failure after metadata
// decoding returns that owned metadata together with the original error.
// Explicit Name lookup is an additional Go convenience; ID performs no lookup.
func (a *API) Fetch(ctx context.Context, ref resource.Ref, options ...FetchOption) (*FetchedSecret, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	wrap := func(err error) error { return request.Wrap("Fetch", "secrets", err) }
	if err := validateFetch(ctx, client); err != nil {
		return nil, wrap(err)
	}
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	config, err := request.Apply(FetchOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = validateFetchHeaders(config.Headers)
	}
	if err != nil {
		return nil, wrap(err)
	}
	owned := copyFetchOptions(config.Options)
	headers := maps.Clone(config.Headers)
	if owned.ContentType != nil && strings.ContainsAny(*owned.ContentType, "\r\n") {
		return nil, wrap(fmt.Errorf("%w: invalid content type header", resource.ErrInvalidOption))
	}
	if err := validateFetch(ctx, client); err != nil {
		return nil, wrap(err)
	}
	id := ref.String()
	if ref.IsName() {
		// Resolve through a fresh existing binding so nil/custom API.Resources
		// cannot redefine this convenience lookup or its native HTTP contract.
		id, err = New(client).Resources.ResolveID(ctx, ref)
	}
	if err == nil {
		err = validateFetch(ctx, client)
	}
	if err == nil {
		err = validateFetchID(id)
	}
	if err != nil {
		return nil, wrap(err)
	}
	// Capture both targets before metadata HTTP. Neither returned secret_ref/id
	// nor a later ResourceBase change can redirect the second step.
	metadataURL := client.ServiceURL("secrets", url.PathEscape(id))
	payloadURL := client.ServiceURL("secrets", url.PathEscape(id), "payload")
	if err := rest.ValidateTarget(client, payloadURL); err != nil {
		return nil, wrap(err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, metadataURL, nil, fetchHeaders(client, headers, "application/json"), http.StatusOK)
	if err != nil {
		return nil, wrap(fetchContextError(ctx, err))
	}
	value, err := rest.Decode(response, "", func(value *FetchedSecret) *resource.Metadata { return &value.Metadata })
	if err != nil {
		return nil, wrap(err)
	}
	value.SecretID = id
	if err := validateFetch(ctx, client); err != nil {
		return value, wrap(err)
	}
	if owned.Payload != nil && !*owned.Payload {
		return value, nil
	}
	contentType, err := fetchContentType(value.Body, owned.ContentType)
	if err != nil {
		return value, wrap(response.Fail(err))
	}
	if contentType == nil {
		return value, nil
	}
	if strings.ContainsAny(*contentType, "\r\n") {
		return value, wrap(response.Fail(fmt.Errorf("invalid content_types.default header")))
	}
	if err := validateFetch(ctx, client); err != nil {
		return value, wrap(err)
	}
	payload, err := rest.DoJSON(ctx, client, http.MethodGet, payloadURL, nil, fetchHeaders(client, headers, *contentType), http.StatusOK)
	if payload != nil {
		value.Payload = &FetchedPayload{Body: append([]byte(nil), payload.Body...), Accept: *contentType, Header: payload.Header.Clone(), StatusCode: payload.StatusCode}
	}
	if err != nil {
		return value, wrap(fetchContextError(ctx, err))
	}
	if *contentType == "text/plain" {
		if !utf8.Valid(value.Payload.Body) {
			return value, wrap(payload.Fail(fmt.Errorf("payload is not valid UTF-8")))
		}
		text := string(value.Payload.Body)
		value.Payload.Text = &text
	}
	return value, nil
}

func validateFetch(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: key-manager service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "key-manager" {
		return fmt.Errorf("%w: expected key-manager service client", resource.ErrUnsupported)
	}
	return validateFetchHeaders(client.MoreHeaders)
}

func validateFetchID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return fmt.Errorf("%w: secret ID must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, r := range id {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("%w: invalid secret ID", resource.ErrInvalidOption)
		}
	}
	return nil
}

func fetchContentType(body map[string]json.RawMessage, explicit *string) (*string, error) {
	if explicit != nil {
		value := *explicit
		return &value, nil
	}
	raw, exists := body["content_types"]
	if !exists {
		return nil, nil
	}
	var types map[string]json.RawMessage
	if err := json.Unmarshal(raw, &types); err != nil {
		return nil, fmt.Errorf("content_types must contain a default: %w", err)
	}
	if types == nil {
		return nil, fmt.Errorf("content_types must be an object containing default")
	}
	selected, exists := types["default"]
	if !exists {
		return nil, fmt.Errorf("content_types lacks default")
	}
	if bytes.Equal(bytes.TrimSpace(selected), []byte("null")) {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(selected, &value); err != nil {
		return nil, fmt.Errorf("content_types.default must be a string or null: %w", err)
	}
	return &value, nil
}

func fetchContextError(ctx context.Context, err error) error {
	if err != nil && ctx.Err() != nil {
		return errors.Join(err, ctx.Err())
	}
	return err
}

func fetchHeaders(client *gophercloud.ServiceClient, extras map[string]string, accept string) map[string]string {
	headers := make(map[string]string, len(extras)+1)
	for key, value := range extras {
		// ServiceClient applies its headers last. Use its spelling when a
		// case-equivalent extra exists, retaining that precedence deterministically.
		for sourceKey := range client.MoreHeaders {
			if strings.EqualFold(sourceKey, key) {
				key = sourceKey
				break
			}
		}
		headers[key] = value
	}
	headers["Accept"] = accept
	return headers
}
