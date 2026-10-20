package waits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const secretStoreKind = "secret-stores"

// SecretStore fixes a literal member identity. Returned HREFs and the native
// SecretStore convenience ID never change this route. This target only polls
// GET; Source SecretStore.allow_delete is false.
func SecretStore(id string) Target {
	view, err := projectSecretStore(map[string]json.RawMessage{}, nil, &id)
	return Target{kind: secretStoreKind, id: id, seed: &Record{Kind: secretStoreKind, RequestID: id, Resource: view}, err: err}
}

// ForSecretStore snapshots a native fetched backend. The caller supplies its
// immutable member/selector identity because the native result has no request
// identity. Metadata.Body, rather than typed fields or convenience ID, supplies
// nullable/untyped Source attributes. No original byte envelope is synthesized.
func ForSecretStore(id string, value *secretstores.SecretStore) Target {
	if value == nil || value.Body == nil {
		return Target{err: invalid("a fetched SecretStore with original Body is required")}
	}
	wire := (&resource.RawResource{Metadata: value.Metadata}).Clone()
	keys := make([]string, 0, len(wire.Body))
	for key := range wire.Body {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	members := make([]cloudfilter.JSONMember, 0, len(keys))
	for _, key := range keys {
		members = append(members, cloudfilter.JSONMember{Key: key, Value: wire.Body[key]})
	}
	// A bulk Go map has no original object ordering. The sorted, deterministic
	// constructor policy applies keys in sorted order; later aliases win.
	fields := map[string]json.RawMessage{}
	consumeSecretStore(fields, members)
	view, err := projectSecretStore(fields, wire, &id)
	if err != nil {
		return Target{err: err}
	}
	return fromRecord(secretStoreKind, id, &Record{Resource: view, Wire: wire, Header: wire.Header.Clone(), StatusCode: wire.StatusCode})
}

func secretStoreFields() map[string]string {
	return map[string]string{"id": "id", "name": "name", "status": "status", "created_at": "created", "updated_at": "updated", "secret_store_ref": "secret_store_ref", "global_default": "global_default", "crypto_plugin": "crypto_plugin", "secret_store_plugin": "secret_store_plugin"}
}
func secretStoreAttribute(attribute string) bool {
	_, ok := secretStoreFields()[attribute]
	return ok || attribute == "secret_store_id" || attribute == "location"
}
func secretStoreCanonical(key string) (string, bool) {
	if key == "secret_store_id" {
		return "secret_store_ref", true
	}
	for name, wire := range secretStoreFields() {
		if key == name || key == wire {
			return wire, true
		}
	}
	return "", false
}
func consumeSecretStore(fields map[string]json.RawMessage, members []cloudfilter.JSONMember) {
	for _, member := range members {
		if key, known := secretStoreCanonical(member.Key); known {
			fields[key] = bytes.Clone(member.Value)
		}
	}
}
func secretStorePrevious(previous *Record) map[string]json.RawMessage {
	fields := map[string]json.RawMessage{}
	if previous == nil || previous.Resource == nil {
		return fields
	}
	for name, wire := range secretStoreFields() {
		if raw, present := previous.Resource.Body[name]; present {
			fields[wire] = bytes.Clone(raw)
		}
	}
	return fields
}
func secretStoreTruthy(raw json.RawMessage) (bool, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return false, invalid("SecretStore boolean requires complete UTF-8 JSON")
	}
	token := bytes.TrimSpace(raw)
	if (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		value, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return value != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}
func projectSecretStore(fields map[string]json.RawMessage, wire *resource.RawResource, requestID *string) (*resource.RawResource, error) {
	view := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}}}
	if wire != nil {
		view.Header, view.StatusCode = wire.Header.Clone(), wire.StatusCode
	}
	for name, key := range secretStoreFields() {
		raw, present := fields[key]
		if !present {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, invalid("SecretStore descriptor requires complete UTF-8 JSON")
		}
		view.Body[name] = bytes.Clone(raw)
	}
	if _, present := fields["id"]; !present {
		if requestID != nil {
			view.Body["id"], _ = json.Marshal(*requestID)
		} else {
			view.Body["id"] = bytes.Clone(view.Body["secret_store_ref"])
		}
	}
	raw := bytes.TrimSpace(view.Body["global_default"])
	if !bytes.Equal(raw, []byte("null")) {
		truth, err := secretStoreTruthy(raw)
		if err != nil {
			return nil, err
		}
		view.Body["global_default"], _ = json.Marshal(truth)
	}
	formatted, err := jsonfilter.ReferenceLastComponent(view.Body["secret_store_ref"])
	if err != nil {
		return nil, fmt.Errorf("%w: secret_store_ref formatter: %w", resource.ErrInvalidOption, err)
	}
	view.Body["secret_store_id"] = formatted
	view.Body["location"] = json.RawMessage("null")
	return view, nil
}

// fetchSecretStore adds a finite Source profile, not a new transport. It reuses
// reader's captured source/header/version and physical rejection policy. Raw
// object member ordering is retained before synchronized attribute projection.
func fetchSecretStore(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, id string, previous *Record) (*Record, error) {
	response, err := reader(ctx, &source.Client, guard, source.Client.ServiceURL(secretStoreKind, url.PathEscape(id)), nil)
	if response == nil {
		receipt := recordFromError(secretStoreKind, id, err)
		if receipt == nil {
			var auth *gophercloud.ErrUnableToReauthenticate
			if errors.As(err, &auth) {
				receipt = recordFromError(secretStoreKind, id, auth.ErrOriginal)
			}
		}
		return receipt, cloudread.ContextError(ctx, err)
	}
	result := &Record{Kind: secretStoreKind, RequestID: id, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	var wire resource.RawResource
	if err != nil {
		if json.Unmarshal(response.Body, &wire) == nil {
			wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
			result.Wire = wire.Clone()
		}
		return result, cloudread.ContextError(ctx, err)
	}
	if !utf8.Valid(response.Body) {
		return result, response.Fail(invalid("SecretStore metadata must be valid UTF-8"))
	}
	fields := secretStorePrevious(previous)
	if json.Valid(response.Body) {
		members, cause := cloudfilter.ObjectMembers(response.Body)
		if cause != nil {
			return result, response.Fail(cause)
		}
		if cause = json.Unmarshal(response.Body, &wire); cause != nil {
			return result, response.Fail(cause)
		}
		wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
		result.Wire = wire.Clone()
		consumeSecretStore(fields, members)
	}
	// Source generic Resource.fetch ignores malformed/empty JSON ValueError,
	// preserving its synchronized previous state and the new HTTP observation.
	view, cause := projectSecretStore(fields, result.Wire, &id)
	if result.Wire == nil && view != nil {
		view.Header, view.StatusCode = response.Header.Clone(), response.StatusCode
	}
	if cause = errors.Join(cause, guard(ctx)); cause != nil {
		return result, response.Fail(cloudread.ContextError(ctx, cause))
	}
	result.Resource = view
	return result, nil
}
