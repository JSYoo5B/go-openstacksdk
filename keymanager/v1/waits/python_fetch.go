package waits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/keymanagerread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func reader(ctx context.Context, client *gophercloud.ServiceClient, guard func(context.Context) error, target string, accept *string) (*rest.Response, error) {
	values := maps.Clone(client.MoreHeaders)
	if values == nil {
		values = map[string]string{}
	}
	absent := []string{"Authorization", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version"}
	if client.Microversion == "" {
		absent = append(absent, "OpenStack-API-Version")
	} else {
		values["OpenStack-API-Version"] = "key-manager " + client.Microversion
	}
	copy := *client
	copy.MoreHeaders = maps.Clone(client.MoreHeaders)
	if accept != nil {
		for key := range values {
			if strings.EqualFold(key, "Accept") {
				delete(values, key)
			}
		}
		for key := range copy.MoreHeaders {
			if strings.EqualFold(key, "Accept") {
				delete(copy.MoreHeaders, key)
			}
		}
		values["Accept"] = *accept
	}
	accepted := make([]int, 300)
	for i := range accepted {
		accepted[i] = 100 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &copy, guard, http.MethodGet, target, nil, rest.RequestHeaderPolicy{Values: values, Absent: absent}, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
}
func fetch(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, kind, id string, previous *Record) (*Record, error) {
	if kind == secretACLKind {
		return fetchSecretACL(ctx, source, guard, id, previous)
	}
	if kind == projectQuotaKind {
		return fetchProjectQuota(ctx, source, guard, id, previous)
	}
	if kind == quotaKind {
		return fetchQuota(ctx, source, guard, id, previous)
	}
	if kind == "secret-stores" {
		return fetchSecretStore(ctx, source, guard, id, previous)
	}
	target := source.Client.ServiceURL(kind, url.PathEscape(id))
	response, err := reader(ctx, &source.Client, guard, target, nil)
	if response == nil {
		return recordFromError(kind, id, err), cloudread.ContextError(ctx, err)
	}
	result := &Record{Kind: kind, RequestID: id, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	var wire resource.RawResource
	if err != nil {
		if json.Unmarshal(response.Body, &wire) == nil {
			wire.Header = response.Header.Clone()
			wire.StatusCode = response.StatusCode
			result.Wire = wire.Clone()
		}
		return result, err
	}
	if !utf8.Valid(response.Body) {
		return result, response.Fail(invalid("metadata must be valid UTF-8"))
	}
	if !json.Valid(response.Body) {
		if kind == "secrets" {
			var decoded any
			decodeErr := json.Unmarshal(response.Body, &decoded)
			return result, response.Fail(decodeErr)
		}
		// Resource._translate_response ignores JSON ValueError for generic resources,
		// retaining the previous synchronized fields and observing the new headers.
	} else {
		if decodeErr := json.Unmarshal(response.Body, &wire); decodeErr != nil {
			return result, response.Fail(decodeErr)
		}
		wire.Header = response.Header.Clone()
		wire.StatusCode = response.StatusCode
		result.Wire = wire.Clone()
	}
	fields := canonicalFields(kind, previous.Resource.Body)
	if result.Wire != nil {
		maps.Copy(fields, canonicalFields(kind, result.Wire.Body))
	}
	if kind == "secrets" {
		contentType, err := payloadSelector(previous, result.Wire)
		if err != nil {
			return result, response.Fail(err)
		}
		if contentType != nil {
			if !validAccept(*contentType) {
				return result, response.Fail(invalid("invalid selected payload Accept"))
			}
			payload, err := reader(ctx, &source.Client, guard, source.Client.ServiceURL(kind, url.PathEscape(id), "payload"), contentType)
			if payload != nil {
				result.Payload = &secrets.FetchedPayload{Body: bytes.Clone(payload.Body), Header: payload.Header.Clone(), StatusCode: payload.StatusCode, Accept: *contentType}
			}
			if err != nil {
				if payload == nil {
					if proof := recordFromError(kind, id, err); proof != nil {
						result.Payload = &secrets.FetchedPayload{Body: bytes.Clone(proof.Envelope), Header: proof.Header.Clone(), StatusCode: proof.StatusCode, Accept: *contentType}
					}
				}
				return result, &payloadFailure{cause: cloudread.ContextError(ctx, err)}
			}
			if *contentType == "text/plain" {
				if !utf8.Valid(payload.Body) {
					return result, &payloadFailure{cause: payload.Fail(invalid("text payload must be valid UTF-8"))}
				}
				v := string(payload.Body)
				result.Payload.Text = &v
				fields["payload"], _ = json.Marshal(v)
			} else {
				fields["payload"], _ = json.Marshal([]byte(payload.Body))
			}
		} else {
			result.Payload = previous.Clone().Payload
		}
	}
	// Project the synchronized accumulated descriptors, but expose only literal
	// current metadata in Wire and the actual current envelope in Envelope.
	effective := &resource.RawResource{Metadata: resource.Metadata{Body: fields, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	view, err := keymanagerread.ProjectRecord(effective, kind, &id)
	if err := errors.Join(err, guard(ctx)); err != nil {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	result.Resource = view
	return result, nil
}
func payloadSelector(previous *Record, wire *resource.RawResource) (*string, error) {
	if previous != nil && previous.Resource != nil {
		selected, err := text(previous.Resource.Body["payload_content_type"])
		if err != nil {
			return nil, err
		}
		if selected != nil {
			return selected, nil
		}
	}
	if wire == nil {
		return nil, invalid("secret metadata body is required")
	}
	raw, present := wire.Body["content_types"]
	if !present {
		return nil, nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, invalid("content_types must be an object")
	}
	value, present := values["default"]
	if !present {
		return nil, invalid("content_types.default is missing")
	}
	return text(value)
}
func validAccept(v string) bool {
	if !utf8.ValidString(v) {
		return false
	}
	for _, c := range v {
		if unicode.IsControl(c) && c != '\t' {
			return false
		}
	}
	return true
}

// Payload absence is not absence of the Secret. The common WaitDeleted error
// classifier uses this marker while retaining the original physical cause.
type payloadFailure struct{ cause error }

func (e *payloadFailure) Error() string            { return "secret payload fetch: " + e.cause.Error() }
func (e *payloadFailure) Unwrap() error            { return e.cause }
func (e *payloadFailure) TerminalSDKFailure() bool { return true }
