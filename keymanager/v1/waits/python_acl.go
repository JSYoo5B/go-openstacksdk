package waits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const secretACLKind = "secret-acls"

// SecretACL owns the parent secret's literal route. It has no status
// descriptor: select acl_ref, name, id or secret_id for string/null waits.
// WaitForDelete only polls this ACL's GET and observes clean absence.
func SecretACL(secretID string) Target {
	view, err := projectACLQuota(secretACLKind, nil, nil)
	return Target{kind: secretACLKind, id: secretID, seed: &Record{Kind: secretACLKind, RequestID: secretID, Resource: view}, err: err}
}

// ForSecretACL snapshots the library's ACL state and immutable SecretID.
// Original Body supplies actual wire fields; when no response Body exists,
// native nullable Read/ACLRef supply cached attributes without invented Wire
// or Envelope. Later response IDs and acl_ref never change the route.
func ForSecretACL(value *secretacls.SecretACL) Target {
	if value == nil {
		return Target{err: invalid("a SecretACL state is required")}
	}
	var wire *resource.RawResource
	fields := map[string]json.RawMessage{}
	if value.Body != nil {
		wire = (&resource.RawResource{Metadata: value.Metadata}).Clone()
		fields = aclQuotaFields(secretACLKind, wire.Body)
	} else {
		if value.Read != nil {
			fields["read"] = bytes.Clone(value.Read)
		}
		if value.ACLRef != nil {
			fields["acl_ref"] = bytes.Clone(value.ACLRef)
		}
	}
	view, err := projectACLQuota(secretACLKind, fields, wire)
	if err != nil {
		return Target{err: err}
	}
	view.Header, view.StatusCode = value.Header.Clone(), value.StatusCode
	return fromRecord(secretACLKind, value.SecretID, &Record{Resource: view, Wire: wire, Header: value.Header.Clone(), StatusCode: value.StatusCode})
}

func secretACLAttribute(attribute string) bool {
	return attribute == "secret_id" || attribute == "location" || aclQuotaDeclared(secretACLKind, attribute)
}

// waitAttribute includes SecretACL's URI descriptor without adding it to the
// Body/to_dict view. Every other finite target retains its Body behavior.
func waitAttribute(kind, id string, record *Record, attribute string) json.RawMessage {
	if kind == secretACLKind && attribute == "secret_id" {
		encoded, _ := json.Marshal(id)
		return encoded
	}
	if record == nil || record.Resource == nil {
		return nil
	}
	return record.Resource.Body[attribute]
}

func aclQuotaDeclared(kind, field string) bool {
	for _, name := range aclQuotaNames(kind) {
		if name == field {
			return true
		}
	}
	return false
}
func aclQuotaNames(kind string) []string {
	if kind == secretACLKind {
		return []string{"id", "name", "read", "acl_ref"}
	}
	return []string{"id", "name", "secrets", "orders", "containers", "consumers", "cas"}
}
func aclQuotaFields(kind string, body map[string]json.RawMessage) map[string]json.RawMessage {
	fields := map[string]json.RawMessage{}
	for _, name := range aclQuotaNames(kind) {
		if raw, present := body[name]; present {
			fields[name] = bytes.Clone(raw)
		}
	}
	return fields
}
func projectACLQuota(kind string, fields map[string]json.RawMessage, wire *resource.RawResource) (*resource.RawResource, error) {
	view := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}}}
	if wire != nil {
		view.Header, view.StatusCode = wire.Header.Clone(), wire.StatusCode
	}
	for _, name := range aclQuotaNames(kind) {
		raw, present := fields[name]
		if !present {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, invalid("ACL/quota descriptor requires complete UTF-8 JSON")
		}
		// Source fields._convert_type(dict) keeps an object, preserves None,
		// and converts every other JSON value to an empty dict on access.
		if kind == secretACLKind && name == "read" {
			trimmed := bytes.TrimSpace(raw)
			if trimmed[0] != '{' && !bytes.Equal(trimmed, []byte("null")) {
				raw = json.RawMessage("{}")
			}
		}
		view.Body[name] = bytes.Clone(raw)
	}
	view.Body["location"] = json.RawMessage("null")
	return view, nil
}

func fetchSecretACL(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, id string, previous *Record) (*Record, error) {
	return fetchACLQuota(ctx, source, guard, secretACLKind, id, source.Client.ServiceURL("secrets", url.PathEscape(id), "acl"), "", previous)
}

// fetchACLQuota reuses the captured guarded reader. Pinned generic wait's
// fetch(requires_id=True) override is corrected by these owned stock routes;
// no response ID/HREF or developer adapter controls subsequent polling.
func fetchACLQuota(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, kind, id, target, envelope string, previous *Record) (*Record, error) {
	response, err := reader(ctx, &source.Client, guard, target, nil)
	if response == nil {
		receipt := recordFromError(kind, id, err)
		if receipt == nil {
			var auth *gophercloud.ErrUnableToReauthenticate
			if errors.As(err, &auth) {
				receipt = recordFromError(kind, id, auth.ErrOriginal)
			}
		}
		return receipt, cloudread.ContextError(ctx, err)
	}
	result := &Record{Kind: kind, RequestID: id, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err != nil {
		var wire resource.RawResource
		if json.Unmarshal(response.Body, &wire) == nil {
			wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
			result.Wire = wire.Clone()
		}
		return result, cloudread.ContextError(ctx, err)
	}
	if !utf8.Valid(response.Body) {
		return result, response.Fail(invalid("ACL/quota metadata must be valid UTF-8"))
	}
	fields := map[string]json.RawMessage{}
	if previous != nil && previous.Resource != nil {
		fields = aclQuotaFields(kind, previous.Resource.Body)
	}
	if json.Valid(response.Body) {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(response.Body, &object); err != nil || object == nil {
			if err == nil {
				err = invalid("ACL/quota response must be an object")
			}
			return result, response.Fail(err)
		}
		selected := response.Body
		if envelope != "" {
			if inner, present := object[envelope]; present {
				selected = inner
			}
		}
		var wire resource.RawResource
		if err := json.Unmarshal(selected, &wire); err != nil {
			return result, response.Fail(err)
		}
		wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
		result.Wire = wire.Clone()
		for name, raw := range aclQuotaFields(kind, wire.Body) {
			fields[name] = raw
		}
	}
	// Generic Source Resource ignores JSON ValueError/empty 202 or 204 bodies
	// and keeps synchronized descriptors; valid non-object shapes still fail.
	view, cause := projectACLQuota(kind, fields, result.Wire)
	if view != nil {
		view.Header, view.StatusCode = response.Header.Clone(), response.StatusCode
	}
	if cause = errors.Join(cause, guard(ctx)); cause != nil {
		return result, response.Fail(cloudread.ContextError(ctx, cause))
	}
	result.Resource = view
	return result, nil
}
