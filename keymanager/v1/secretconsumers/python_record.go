package secretconsumers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ConsumerRecord separates the five Source Body descriptors and their defaults
// from actual wire fields. SecretID is fixed URI state, not an association ID.
// Envelope is the complete POST response or list page; it is never a route.
type ConsumerRecord struct {
	Resource, Wire *resource.RawResource
	SecretID       string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// DeletedConsumer keeps the bodyful DELETE receipt without parsing its body,
// because Source delete_secret_consumer returns None and has_body=False.
type DeletedConsumer struct {
	SecretID   string
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// InSecretRecord fixes a literal parent ID without resolving a name or following
// a secret_ref. It reuses the native SecretScope and authenticated client.
func (a *API) InSecretRecord(ctx context.Context, id string) (*SecretScope, error) {
	if _, err := recordSource(ctx, a.RawClient(), id); err != nil {
		return nil, request.Wrap("InSecretRecord", kind, err)
	}
	return &SecretScope{client: a.RawClient(), secretID: id}, nil
}
func recordSource(ctx context.Context, client *gophercloud.ServiceClient, id string) (*cloudread.Source, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if id == "" || strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return nil, recordInvalid("secret ID must be nonempty literal UTF-8 without controls or dot segments")
	}
	return cloudread.Capture(ctx, client, "key-manager")
}
func recordGuard(source *cloudread.Source) func(context.Context) error {
	return func(ctx context.Context) error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
}
func recordSuccessCodes() []int {
	result := make([]int, 200)
	for i := range result {
		result[i] = i + 200
	}
	return result
}

func (a *API) CreateRecord(ctx context.Context, secretID string, options ...CreateRecordOption) (*ConsumerRecord, error) {
	return (&SecretScope{client: a.RawClient(), secretID: secretID}).CreateRecord(ctx, options...)
}
func (a *API) DeleteRecord(ctx context.Context, secretID string, options ...DeleteRecordOption) (*DeletedConsumer, error) {
	return (&SecretScope{client: a.RawClient(), secretID: secretID}).DeleteRecord(ctx, options...)
}

// CreateRecord owns pinned create_secret_consumer, including new() attribute
// discard/defaults and the seeded Resource returned when Barbican sends a full
// Secret root. The native concrete Create surface retains its existing policy.
func (s *SecretScope) CreateRecord(ctx context.Context, options ...CreateRecordOption) (*ConsumerRecord, error) {
	response, attributes, err := s.mutateRecord(ctx, http.MethodPost, options)
	if response == nil {
		return nil, request.Wrap("CreateRecord", kind, err)
	}
	result := &ConsumerRecord{SecretID: s.secretID, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err != nil {
		return result, request.Wrap("CreateRecord", kind, err)
	}
	// Source catches response.json() ValueError, retaining the request seed.
	// Valid scalar/array/null representations are not a Resource object.
	if len(bytes.TrimSpace(response.Body)) != 0 && json.Valid(response.Body) {
		wire, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
		if decodeErr != nil {
			return result, request.Wrap("CreateRecord", kind, decodeErr)
		}
		result.Wire = wire.Clone()
		for key, raw := range wire.Body {
			if recordBodyField(key) {
				attributes[key] = bytes.Clone(raw)
			}
		}
	} else if !utf8.Valid(response.Body) {
		return result, request.Wrap("CreateRecord", kind, response.Fail(recordInvalid("consumer response must be valid UTF-8")))
	}
	result.Resource = recordView(attributes, nil)
	result.Resource.Header, result.Resource.StatusCode = response.Header.Clone(), response.StatusCode
	return result, nil
}

// DeleteRecord defaults ignore_missing=true. Only a clean native HTTP404 may
// be suppressed; rejected Read/Close/source/retry/cancel causes remain errors.
// Successful bodies are passive bytes, including malformed JSON and HTTP204.
func (s *SecretScope) DeleteRecord(ctx context.Context, options ...DeleteRecordOption) (*DeletedConsumer, error) {
	response, _, err := s.mutateRecord(ctx, http.MethodDelete, options)
	if response == nil {
		return nil, request.Wrap("DeleteRecord", kind, err)
	}
	result := &DeletedConsumer{SecretID: s.secretID, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, request.Wrap("DeleteRecord", kind, err)
}

func (s *SecretScope) mutateRecord(ctx context.Context, method string, options []request.Option[RecordMutationOpts]) (*rest.Response, map[string]json.RawMessage, error) {
	id := ""
	if s != nil {
		id = s.secretID
	}
	source, err := recordSource(ctx, s.RawClient(), id)
	if err != nil {
		return nil, nil, err
	}
	guard := recordGuard(source)
	owned := slices.Clone(options)
	prepared := make([]request.Option[RecordMutationOpts], len(owned))
	own := func(config *request.Config[RecordMutationOpts]) error {
		fields, err := recordAttributes(config.Options.Attributes, false)
		if err != nil {
			return err
		}
		attributes, err := decodeRecordAttributes(fields)
		if err != nil {
			return err
		}
		config.Options.Attributes = attributes
		if config.Options.IgnoreMissing != nil {
			copy := *config.Options.IgnoreMissing
			config.Options.IgnoreMissing = &copy
		}
		ownRecordConfig(config)
		return nil
	}
	for i, option := range owned {
		apply := option
		prepared[i] = func(config *request.Config[RecordMutationOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return recordInvalid("nil consumer record option")
			}
			if err := own(config); err != nil {
				return errors.Join(err, guard(ctx))
			}
			applied := apply(config)
			return errors.Join(applied, own(config), guard(ctx))
		}
	}
	config, err := request.Apply(RecordMutationOpts{}, prepared...)
	if err == nil {
		err = own(&config)
	}
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = recordHeaderPolicy(config.Headers)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	var attributes map[string]json.RawMessage
	if err == nil {
		attributes, err = recordAttributes(config.Options.Attributes, false)
	}
	err = errors.Join(err, guard(ctx))
	if err != nil {
		return nil, nil, cloudread.ContextError(ctx, err)
	}
	target := source.Client.ServiceURL("secrets", url.PathEscape(id), "consumers")
	response, err := rest.DoJSONGuardedRejections(ctx, &source.Client, guard, method, target, recordCloneFields(attributes), nil, rest.RejectionPolicy{Codes: []int{400, 401, 403, 404, 409}, PreserveCleanRetry: true}, recordSuccessCodes()...)
	err = cloudread.ContextError(ctx, errors.Join(err, guard(ctx)))
	if method == http.MethodDelete && (config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing) {
		err = resource.IgnoreMissing(err)
	}
	return response, attributes, err
}
func recordView(fields map[string]json.RawMessage, metadata *resource.Metadata) *resource.RawResource {
	value := &resource.RawResource{Metadata: resource.Metadata{Body: make(map[string]json.RawMessage)}}
	if metadata != nil {
		value.Header, value.StatusCode = metadata.Header.Clone(), metadata.StatusCode
	}
	for _, key := range []string{"id", "name", "service", "resource_type", "resource_id"} {
		raw, exists := fields[key]
		if !exists {
			raw = json.RawMessage("null")
		}
		value.Body[key] = bytes.Clone(raw)
	}
	// No cloud/session is fabricated from credentials or a foreign response.
	value.Body["location"] = json.RawMessage("null")
	return value
}
