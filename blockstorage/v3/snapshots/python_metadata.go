package snapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/metadata"
	"github.com/JSYoo5B/go-openstacksdk/internal/cindermetadata"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// RecordMetadataOpts supplies owned headers and an optional declared resource
// seed. The literal method ID remains routing authority; seed fields cannot
// select a service, collection or metadata target.
type RecordMetadataOpts struct {
	Headers map[string]string
	Seed    *SnapshotRecord
}
type RecordMetadataOption func(*RecordMetadataOpts) error

func WithRecordMetadataHeader(key, value string) RecordMetadataOption {
	return WithRecordMetadataHeaders(map[string]string{key: value})
}
func WithRecordMetadataHeaders(headers map[string]string) RecordMetadataOption {
	owned := maps.Clone(headers)
	return func(cfg *RecordMetadataOpts) error {
		native, err := request.Apply(metadata.Opts{}, metadata.WithHeaders(owned))
		if err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		maps.Copy(cfg.Headers, native.Headers)
		return nil
	}
}
func WithRecordMetadataSeed(seed *SnapshotRecord) RecordMetadataOption {
	owned := cloneMetadataRecordSeed(seed)
	return func(cfg *RecordMetadataOpts) error { cfg.Seed = cloneMetadataRecordSeed(owned); return nil }
}

func cloneMetadataRecordSeed(seed *SnapshotRecord) *SnapshotRecord {
	if seed == nil {
		return nil
	}
	// Only the declared view is a seed; wire/envelope never become resource
	// state merely because an earlier operation observed their bytes.
	return &SnapshotRecord{Resource: seed.Resource.Clone()}
}
func copyRecordMetadataOpts(cfg RecordMetadataOpts) RecordMetadataOpts {
	cfg.Headers = maps.Clone(cfg.Headers)
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	cfg.Seed = cloneMetadataRecordSeed(cfg.Seed)

	return cfg
}

// MetadataRecordReceipt is one actual accepted metadata response. Key is nil
// for a collection request; an empty successful response stays empty evidence.
type MetadataRecordReceipt struct {
	Key        *string
	Method     string
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// MetadataRecordDeletion preserves the owned cached view and every accepted
// physical receipt, including a partial last attempt on failure. This is an
// ordered workflow, not a server transaction or proof of lasting absence.
type MetadataRecordDeletion struct {
	Record      *SnapshotRecord
	Responses   []MetadataRecordReceipt
	DeletedKeys []string
	Cleared     bool
}

type recordMetadataState struct {
	source  *cloudread.Source
	scope   *cindermetadata.Scope
	guard   func(context.Context) error
	headers map[string]string
	fields  map[string]json.RawMessage
}

func (a *API) prepareRecordMetadata(ctx context.Context, id string, options []RecordMetadataOption) (*recordMetadataState, error) {
	var selected *gophercloud.ServiceClient
	if a != nil {
		selected = a.client
	}
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a == nil || a.client != selected {
			changed = findInvalid("selected metadata client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err := errors.Join(findSegment(id), guard(ctx)); err != nil {
		return nil, err
	}
	cfg := RecordMetadataOpts{Headers: make(map[string]string)}
	for _, option := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, err
		}
		if option == nil {
			return nil, findInvalid("nil metadata record option")
		}
		next := copyRecordMetadataOpts(cfg)
		if err := errors.Join(option(&next), guard(ctx)); err != nil {
			return nil, err
		}
		cfg = copyRecordMetadataOpts(next)
	}
	if err := applyRecordMetadataHeaders(ctx, source, cfg.Headers); err != nil {
		return nil, errors.Join(err, guard(ctx))
	}
	scope, err := cindermetadata.NewRecord(ctx, &source.Client, "snapshots", id)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return nil, err
	}
	rawID, _ := json.Marshal(id)
	fields := map[string]json.RawMessage{"id": rawID}
	if cfg.Seed != nil {
		if cfg.Seed.Resource == nil {
			return nil, findInvalid("metadata seed needs a declared Resource view")
		}
		for key, raw := range cfg.Seed.Resource.Body {
			if findBodyKnown(key) && key != "id" {
				if err := metadataRecordSeedField(key, raw); err != nil {
					return nil, errors.Join(err, guard(ctx))
				}
				fields[key] = bytes.Clone(raw)
			}
		}
	}
	state := &recordMetadataState{source: source, scope: scope, guard: guard, headers: maps.Clone(cfg.Headers), fields: fields}

	// Seed descriptor validation cannot create a physical response receipt.
	if _, err := state.view(nil); err != nil {
		return nil, errors.Join(err, guard(ctx))
	}
	return state, guard(ctx)
}
func applyRecordMetadataHeaders(ctx context.Context, source *cloudread.Source, headers map[string]string) error {
	// Reuse metadata's concrete header validation even when an empty key list
	// means no HTTP. Authentication, content and version controls remain owned.
	cfg, err := request.Apply(metadata.Opts{}, metadata.WithHeaders(headers))
	if err != nil {
		return err
	}
	for key, value := range cfg.Headers {
		switch strings.ToLower(key) {
		case "content-type", "openstack-api-version", "x-openstack-volume-api-version", "x-openstack-nova-api-version":
			return findInvalid("metadata header %q is SDK owned", key)
		case "if-match":
			for configured, old := range source.Client.MoreHeaders {
				if strings.EqualFold(configured, key) && old != value {
					return findInvalid("If-Match conflicts with configured metadata condition")
				}
			}
		}
	}
	if err := source.WithPolicy(ctx, nil, cfg.Headers); err != nil {
		return err
	}
	return cindermetadata.Validate(ctx, &source.Client)
}

func metadataRecordReceipt(method string, key *string, response *rest.Response) MetadataRecordReceipt {
	value := MetadataRecordReceipt{Method: method, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if key != nil {
		owned := *key
		value.Key = &owned
	}
	return value
}
func (s *recordMetadataState) exchange(ctx context.Context, method string, key *string, body any) (*rest.Response, error) {
	// MetadataMixin uses the adapter's selected default and never invokes
	// Resource microversion discovery. Keep this captured default unchanged.
	response, err := s.scope.ExchangeRecord(ctx, method, key, body, s.guard, metadata.WithHeaders(s.headers))
	return response, errors.Join(err, s.guard(ctx))
}
func (s *recordMetadataState) receipt(response *rest.Response) *SnapshotRecord {
	value := &SnapshotRecord{}
	if response != nil {
		value.Envelope, value.Header, value.StatusCode = bytes.Clone(response.Body), response.Header.Clone(), response.StatusCode
		// Mutations do not parse the response. A well-formed object can be
		// retained as Wire without promoting any field into the cached view.
		var wire resource.RawResource
		if json.Unmarshal(response.Body, &wire) == nil {
			wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
			value.Wire = wire.Clone()
		}
	}
	return value
}
func (s *recordMetadataState) view(response *rest.Response) (*SnapshotRecord, error) {
	value := s.receipt(response)
	value.fields = metadataRecordFields(s.fields)
	err := findProject(value, s.fields)
	if err != nil {
		err = fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	return value, err
}
func metadataRecordSeedField(key string, raw json.RawMessage) error {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return findInvalid("metadata record seed field %q needs valid UTF-8 JSON", key)
	}
	return nil
}
func metadataRecordFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(fields))
	for key, raw := range fields {
		owned[key] = bytes.Clone(raw)
	}
	return owned
}

// FetchMetadataRecord returns the selected resource with its metadata updated,
// plus the actual metadata endpoint receipt. Missing metadata preserves a seed.
func (a *API) FetchMetadataRecord(ctx context.Context, id string, options ...RecordMetadataOption) (*SnapshotRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("FetchMetadataRecord", "snapshots.metadata", cloudread.ContextError(ctx, err))
	}
	state, err := a.prepareRecordMetadata(ctx, id, options)
	if err != nil {
		return nil, wrap(err)
	}
	response, err := state.exchange(ctx, http.MethodGet, nil, nil)
	if response == nil {
		return nil, wrap(err)
	}
	if err != nil {
		return state.receipt(response), wrap(err)
	}
	if !utf8.Valid(response.Body) {
		return state.receipt(response), wrap(response.Fail(findInvalid("metadata response must be UTF-8 JSON")))
	}
	// Source fetch parses JSON but changes exactly the metadata member.
	var object resource.RawResource
	if err := json.Unmarshal(response.Body, &object); err != nil {
		return state.receipt(response), wrap(response.Fail(err))
	}
	if raw, present := object.Body["metadata"]; present {
		state.fields["metadata"] = bytes.Clone(raw)
	}
	value, err := state.view(response)
	if err = errors.Join(err, state.guard(ctx)); err != nil {
		return value, wrap(response.Fail(err))
	}
	return value, nil
}

// GetMetadataRecord is the pinned deprecated get_*_metadata alias. Go does
// not emit Python warnings; FetchMetadataRecord is the preferred callable.
func (a *API) GetMetadataRecord(ctx context.Context, id string, options ...RecordMetadataOption) (*SnapshotRecord, error) {
	return a.FetchMetadataRecord(ctx, id, options...)
}

// SetMetadataRecord sends a POST merge and caches the request delta exactly.
// It never treats the server's merged metadata response as Source cached state.
func (a *API) SetMetadataRecord(ctx context.Context, id string, values map[string]any, options ...RecordMetadataOption) (*SnapshotRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("SetMetadataRecord", "snapshots.metadata", cloudread.ContextError(ctx, err))
	}
	state, err := a.prepareRecordMetadata(ctx, id, nil)
	if err != nil {
		return nil, wrap(err)
	}
	// Capture the source before custom MarshalJSON, and snapshot values before
	// option callbacks can mutate the caller's nested map.
	if values == nil {
		values = map[string]any{}
	}
	raw, marshalErr := json.Marshal(values)
	if marshalErr == nil && !utf8.Valid(raw) {
		marshalErr = findInvalid("metadata input must be UTF-8 JSON")
	}
	if err = errors.Join(marshalErr, state.guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	// Applying options through the same captured state must not recapture an
	// original route changed by a custom input callback.
	cfg := RecordMetadataOpts{Headers: make(map[string]string)}
	for _, option := range slices.Clone(options) {
		if err := state.guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if option == nil {
			return nil, wrap(findInvalid("nil metadata record option"))
		}
		next := copyRecordMetadataOpts(cfg)
		if err := errors.Join(option(&next), state.guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = copyRecordMetadataOpts(next)
	}
	if cfg.Seed != nil {
		if cfg.Seed.Resource == nil {
			return nil, wrap(findInvalid("metadata seed needs a declared Resource view"))
		}
		for key, value := range cfg.Seed.Resource.Body {
			if findBodyKnown(key) && key != "id" {
				if err := metadataRecordSeedField(key, value); err != nil {
					return nil, wrap(errors.Join(err, state.guard(ctx)))
				}
				state.fields[key] = bytes.Clone(value)
			}
		}
	}
	if err := applyRecordMetadataHeaders(ctx, state.source, cfg.Headers); err != nil {
		return nil, wrap(errors.Join(err, state.guard(ctx)))
	}
	state.headers = maps.Clone(cfg.Headers)

	if _, err := state.view(nil); err != nil {
		return nil, wrap(errors.Join(err, state.guard(ctx)))
	}
	response, err := state.exchange(ctx, http.MethodPost, nil, map[string]any{"metadata": json.RawMessage(bytes.Clone(raw))})
	if response == nil {
		return nil, wrap(err)
	}
	if err != nil {
		return state.receipt(response), wrap(err)
	}
	state.fields["metadata"] = bytes.Clone(raw)
	value, err := state.view(response)
	if err = errors.Join(err, state.guard(ctx)); err != nil {
		return value, wrap(response.Fail(err))
	}
	return value, nil
}

// DeleteMetadataRecord follows keys=None versus a nonnil iterable: nil clears
// with PUT metadata:{}, empty performs no HTTP, and other keys are ordered.
func (a *API) DeleteMetadataRecord(ctx context.Context, id string, keys []string, options ...RecordMetadataOption) (*MetadataRecordDeletion, error) {
	wrap := func(err error) error {
		return request.Wrap("DeleteMetadataRecord", "snapshots.metadata", cloudread.ContextError(ctx, err))
	}
	snapshot := slices.Clone(keys)
	state, err := a.prepareRecordMetadata(ctx, id, options)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := state.view(nil)
	if err != nil {
		return nil, wrap(err)
	}
	result := &MetadataRecordDeletion{Record: value}
	if keys == nil {
		response, err := state.exchange(ctx, http.MethodPut, nil, map[string]any{"metadata": map[string]any{}})
		if response != nil && response.StatusCode < 400 {
			result.Responses = append(result.Responses, metadataRecordReceipt(http.MethodPut, nil, response))
		}
		if err != nil {
			return result, wrap(err)
		}
		state.fields["metadata"] = json.RawMessage("{}")
		result.Record, err = state.view(response)
		if err = errors.Join(err, state.guard(ctx)); err != nil {
			return result, wrap(response.Fail(err))
		}
		result.Cleared = true
		return result, nil
	}
	for _, key := range snapshot {
		owned := key
		response, err := state.exchange(ctx, http.MethodDelete, &owned, nil)
		if response != nil && response.StatusCode < 400 {
			result.Responses = append(result.Responses, metadataRecordReceipt(http.MethodDelete, &owned, response))
		}
		if err != nil {
			return result, wrap(err)
		}
		// Source pops from its truthy descriptor dictionary after HTTP. An
		// absent cached key raises KeyError even if the server accepted DELETE.
		projected, err := state.view(response)
		if err != nil {
			return result, wrap(response.Fail(err))
		}
		cached := projected.Resource.Body["metadata"]
		truthy, err := cloudfilter.PythonTruthy(cached)
		if err != nil {
			return result, wrap(response.Fail(err))
		}
		values := make(map[string]json.RawMessage)
		if truthy {
			if err := json.Unmarshal(cached, &values); err != nil {
				return result, wrap(response.Fail(err))
			}
			if _, present := values[key]; !present {
				return result, wrap(response.Fail(&MetadataRecordCacheKeyError{Key: key}))
			}
			delete(values, key)
		}
		state.fields["metadata"], err = json.Marshal(values)
		if err != nil {
			return result, wrap(response.Fail(err))
		}
		result.DeletedKeys = append(result.DeletedKeys, key)
		result.Record, err = state.view(response)
		if err = errors.Join(err, state.guard(ctx)); err != nil {
			return result, wrap(response.Fail(err))
		}
	}
	return result, wrap(state.guard(ctx))
}

// MetadataRecordCacheKeyError describes a Source cached-dictionary pop failure
// after an actual DELETE ACK; it never means the server rejected that deletion.
type MetadataRecordCacheKeyError struct{ Key string }

func (e *MetadataRecordCacheKeyError) Error() string {
	return fmt.Sprintf("metadata cached key %q is absent after DELETE acknowledgement", e.Key)
}
