package quotasets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaRevertOpts carries Python revert_quota_set **query as DELETE params.
// Every Query key is a literal URI query parameter, including spellings such
// as headers, base_path and microversion. Only the typed Go fields configure
// transport policy. Options copy raw JSON before callbacks and HTTP.
type QuotaRevertOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type QuotaRevertOption func(*QuotaRevertOpts) error

func copyQuotaRevert(v QuotaRevertOpts) QuotaRevertOpts {
	v.Query = cinderRevertFields(v.Query)
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = map[string]string{}
	}
	v.order = slices.Clone(v.order)
	missing := make([]string, 0)
	for key := range v.Query {
		if !slices.Contains(v.order, key) {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)
	v.order = append(v.order, missing...)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func cinderRevertFields(v map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(v))
	for key, value := range v {
		out[key] = bytes.Clone(value)
	}
	return out
}
func WithQuotaRevertOpts(v QuotaRevertOpts) QuotaRevertOption {
	owned := copyQuotaRevert(v)
	return func(o *QuotaRevertOpts) error { *o = copyQuotaRevert(owned); return nil }
}
func WithQuotaRevertQuery(key string, value any) QuotaRevertOption {
	raw, err := json.Marshal(value)
	return func(o *QuotaRevertOpts) error {
		if err != nil {
			return err
		}
		if o.Query == nil {
			o.Query = map[string]json.RawMessage{}
		}
		if !slices.Contains(o.order, key) {
			o.order = append(o.order, key)
		}
		o.Query[key] = bytes.Clone(raw)
		return nil
	}
}
func WithQuotaRevertHeaders(v map[string]string) QuotaRevertOption {
	owned := maps.Clone(v)
	return func(o *QuotaRevertOpts) error { o.Headers = maps.Clone(owned); return nil }
}
func WithQuotaRevertMicroversion(v string) QuotaRevertOption {
	return func(o *QuotaRevertOpts) error { x := v; o.Microversion = &x; return nil }
}

// QuotaRevertRecord is actual DELETE evidence, never a decoded quota Resource.
// Python returns None; Go exposes immutable project/version metadata and raw
// response evidence without a follow-up GET or fabricated deletion success.
type QuotaRevertRecord struct {
	ProjectID, Microversion string
	Envelope                json.RawMessage
	Header                  http.Header
	StatusCode              int
}

// RevertRecord implements the scalar-project revert_quota_set lane. Source's
// Resource._prepare_request uses urllib.parse.urlencode(params), without
// doseq: None/list/dict values become one Python string each. It does not use
// Requests' ordinary repeated-list/null-omitting parameter expansion.
func (s *ProjectQuotaScope) RevertRecord(ctx context.Context, options ...QuotaRevertOption) (*QuotaRevertRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("RevertRecord", "block storage quota", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if s == nil || s.api == nil {
		return nil, wrap(resource.ErrInvalidOption)
	}
	api, projectID := s.api, s.projectID
	selected := api.client
	if err := resource.ID(projectID).Validate(); err != nil {
		return nil, wrap(err)
	}
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if s.api != api || s.projectID != projectID || api.client != selected {
			changed = resource.ErrInvalidOption
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	policy := copyQuotaRevert(QuotaRevertOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(resource.ErrInvalidOption)
		}
		next := copyQuotaRevert(policy)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyQuotaRevert(next)
	}
	keys := slices.Clone(policy.order)
	remaining := make([]string, 0)
	for key := range policy.Query {
		if !slices.Contains(keys, key) {
			remaining = append(remaining, key)
		}
	}
	slices.Sort(remaining)
	keys = append(keys, remaining...)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		raw, present := policy.Query[key]
		if !present {
			continue
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, wrap(resource.ErrInvalidOption)
		}
		value, err := cloudfilter.PythonString(raw)
		if err != nil {
			return nil, wrap(err)
		}
		pairs = append(pairs, url.QueryEscape(key)+"="+url.QueryEscape(value))
	}
	if policy.Microversion != nil {
		source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
	}
	if err := source.WithPolicy(ctx, policy.Microversion, policy.Headers); err != nil {
		return nil, wrap(err)
	}
	target := source.Client.ServiceURL("os-quota-sets", url.PathEscape(projectID))
	if len(pairs) != 0 {
		target += "?" + strings.Join(pairs, "&")
	}
	values := maps.Clone(source.Client.MoreHeaders)
	if values == nil {
		values = map[string]string{}
	}
	headerPolicy := rest.RequestHeaderPolicy{Values: values, Absent: []string{"Authorization", "Cookie", "X-Service-Token", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version"}}
	if source.Client.Microversion != "" {
		values["OpenStack-API-Version"] = "volume " + source.Client.Microversion
		values["X-OpenStack-Volume-API-Version"] = source.Client.Microversion
	} else {
		headerPolicy.Absent = append(headerPolicy.Absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
	}
	accepted := make([]int, 300)
	for i := range accepted {
		accepted[i] = 100 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodDelete, target, nil, headerPolicy, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	var record *QuotaRevertRecord
	if response != nil {
		record = &QuotaRevertRecord{ProjectID: projectID, Microversion: source.Client.Microversion, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	if native, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && native.Actual == 404 {
		err = &resource.NotFoundError{Resource: "block storage quota", Reference: projectID, Cause: err}
	}
	return record, wrap(errors.Join(err, guard(ctx)))
}
