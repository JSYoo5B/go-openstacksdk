package credentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordUpdateOpts owns the literal-ID Credential update policy. Fields are
// declared Python Body attributes; unknown attributes and id are ignored.
// Headers are an explicit Go request-policy extension, not dirty Resource attrs.
type RecordUpdateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

type RecordUpdateOption func(*RecordUpdateOpts) error

func copyRecordUpdate(v RecordUpdateOpts) RecordUpdateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneRecordFields(v.Fields)
	return v
}

// WithRecordUpdateOpts snapshots the whole policy; later field options overlay it.
func WithRecordUpdateOpts(v RecordUpdateOpts) RecordUpdateOption {
	owned := copyRecordUpdate(v)
	return func(c *RecordUpdateOpts) error { *c = copyRecordUpdate(owned); return nil }
}

// WithRecordUpdateField snapshots a declared Body value, including null, false,
// zero and empty collections. Credential has no additional Body aliases.
// Unknown attributes and id do not change the fixed literal member target.
func WithRecordUpdateField(key string, value any) RecordUpdateOption {
	if !recordKnown(key) || key == "id" {
		return func(*RecordUpdateOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordUpdateOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}

func WithRecordUpdateHeaders(values map[string]string) RecordUpdateOption {
	owned := maps.Clone(values)
	return func(c *RecordUpdateOpts) error { c.Headers = maps.Clone(owned); return nil }
}

func recordUpdateRejectedCodes() []int {
	codes := make([]int, 0, 700)
	for code := 100; code <= 999; code++ {
		if code < 200 || code >= 400 {
			codes = append(codes, code)
		}
	}
	return codes
}

// UpdateRecord commits declared attributes to PATCH /credentials/{id} without
// fetching first. Empty, unknown-only or id-only changes make no HTTP request.
// Response fields overlay requested values; omitted fields retain their seed.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordUpdateOption) (*CredentialRecord, error) {
	const op = "UpdateRecord"
	p, err := a.captureRecord(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	policy, err := applyRecordOptions(ctx, p.guard, RecordUpdateOpts{}, copyRecordUpdate, slices.Clone(options))
	if err == nil {
		err = recordID(id)
	}
	if err == nil {
		err = p.headers(policy.Headers)
	}
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}

	fields := make(map[string]json.RawMessage)
	for key, raw := range policy.Fields {
		if !recordKnown(key) || key == "id" {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, recordWrap(ctx, op, recordInvalid("attribute %q must be UTF-8 JSON", key))
		}
		fields[key] = bytes.Clone(raw)
	}
	idJSON, _ := json.Marshal(id)
	seed := cloneRecordFields(fields)
	seed["id"] = idJSON
	if len(fields) == 0 {
		value := &CredentialRecord{}
		projectRecord(value, seed)
		return value, recordWrap(ctx, op, p.guard(ctx))
	}

	response, err := rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.guard,
		http.MethodPatch, p.source.Client.ServiceURL("credentials", url.PathEscape(id)),
		map[string]any{"credential": cloneRecordFields(fields)}, nil,
		rest.RejectionPolicy{Codes: recordUpdateRejectedCodes(), PreserveCleanRetry: true}, recordCodes()...)
	if err != nil {
		if response == nil {
			return nil, recordWrap(ctx, op, err)
		}
		return &CredentialRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, recordWrap(ctx, op, err)
	}
	value, err := recordFromResponse(response, seed)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, recordWrap(ctx, op, err)
}
