package volumes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// SummaryObservation is an actual response. Discovery and member observations
// are distinct; neither invents a volume or completed remote state.
type SummaryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// VolumeSummaryRecord retains untyped aggregate fields, literal wire fields,
// each fetch's terminal actual receipt and independently observed discovery.
type VolumeSummaryRecord struct {
	Resource, Wire      *resource.RawResource
	Envelope            json.RawMessage
	Header              http.Header
	StatusCode          int
	Microversion        string
	Discovery, Requests []SummaryObservation
}

func summaryInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func freezeSummaryQuery(cfg *SummaryOpts) error {
	// Validate every control key before invoking any value's marshaler.
	for key := range cfg.Query {
		switch key {
		case "self", "session", "all_tenants", "all_projects", "requires_id", "base_path", "error_message", "skip_cache", "resource_response_key", "microversion":
			return summaryInvalid("summary control %q requires its typed option", key)
		}
		if !utf8.ValidString(key) || strings.ContainsAny(key, "\r\n\x00") {
			return summaryInvalid("invalid summary query key")
		}
	}
	for key, value := range cfg.Query {
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if !utf8.Valid(raw) {
			return summaryInvalid("summary query field %q must be UTF-8", key)
		}
		cfg.Query[key] = json.RawMessage(bytes.Clone(raw))
	}
	return nil
}
func summaryCodes(first, end int) []int {
	out := make([]int, end-first)
	for i := range out {
		out[i] = first + i
	}
	return out
}
func summaryObservation(r *rest.Response) SummaryObservation {
	return SummaryObservation{bytes.Clone(r.Body), r.Header.Clone(), r.StatusCode}
}
func summaryNativeResponse(err error) *rest.Response {
	var authentication *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &authentication) {
		err = authentication.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) {
		return nil
	}
	return &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
}
func summaryView(r *rest.Response, key string, fields map[string]json.RawMessage) (*resource.RawResource, *resource.RawResource, error) {
	if !utf8.Valid(r.Body) {
		return nil, nil, summaryInvalid("summary response must be UTF-8")
	}
	var wire *resource.RawResource
	// Pinned Resource.fetch tolerates accepted non-JSON/empty receipts.
	if json.Valid(r.Body) {
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(r.Body, &envelope); err != nil || envelope == nil {
			return nil, nil, fmt.Errorf("summary response must be a nonnull JSON object")
		}
		raw := r.Body
		if nested, ok := envelope[key]; key != "" && ok {
			raw = nested
		}
		var actual resource.RawResource
		if err := json.Unmarshal(raw, &actual); err != nil {
			return nil, nil, err
		}
		actual.Header, actual.StatusCode = r.Header.Clone(), r.StatusCode
		wire = actual.Clone()
		for _, name := range []string{"id", "name", "total_size", "total_count", "metadata"} {
			if value, ok := actual.Body[name]; ok {
				fields[name] = bytes.Clone(value)
			}
		}
	}
	body := make(map[string]json.RawMessage, 6)
	for _, name := range []string{"id", "name", "total_size", "total_count", "metadata"} {
		body[name] = bytes.Clone(fields[name])
		if body[name] == nil {
			body[name] = json.RawMessage("null")
		}
	}
	body["location"] = json.RawMessage("null")
	return &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: r.Header.Clone(), StatusCode: r.StatusCode}}, wire, nil
}

// SummaryRecord implements Proxy.summary; it is distinct from ListSummary,
// which lists individual /volumes rows. AllProjects defaults to false. The
// pinned Source first fetches without a query, then fetches volume-summary
// using all_tenants=False/True. SingleFetch explicitly removes the first read.
func (a *API) SummaryRecord(ctx context.Context, options ...SummaryOption) (*VolumeSummaryRecord, error) {
	const op = "SummaryRecord"
	wrap := func(err error) error { return request.Wrap(op, "volume-summary", cloudread.ContextError(ctx, err)) }
	if a == nil {
		return nil, wrap(summaryInvalid("configured volume API is required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = summaryInvalid("selected summary client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	cfg := cloneSummary(SummaryOpts{})
	for _, apply := range slices.Clone(options) {
		if err = guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(summaryInvalid("nil summary option"))
		}
		next := cloneSummary(cfg)
		err = apply(&next)
		if err == nil {
			err = freezeSummaryQuery(&next)
		}
		err = errors.Join(err, guard(ctx))
		if err != nil {
			return nil, wrap(err)
		}
		cfg = cloneSummary(next)
	}
	if err = errors.Join(freezeSummaryQuery(&cfg), guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	query := make(url.Values)
	for key, value := range cfg.Query {
		values, cause := cloudfilter.RequestQueryValues(value.(json.RawMessage))
		if len(values) > 0 {
			query[key] = slices.Clone(values)
		}
		if cause = errors.Join(cause, guard(ctx)); cause != nil {
			return nil, wrap(cause)
		}
	}
	if cfg.AllProjects {
		query.Set("all_tenants", "True")
	} else {
		query.Set("all_tenants", "False")
	}
	// The first pinned fetch uses the session version. A fetch-specific override
	// belongs only to the second request, so it cannot rewrite the first receipt.
	for key := range cfg.Headers {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			return nil, wrap(summaryInvalid("summary version headers are SDK owned"))
		}
	}
	if err = errors.Join(source.WithPolicy(ctx, nil, cfg.Headers), guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	if cfg.Microversion != nil && *cfg.Microversion != "" {
		if _, err = microversions.ParseVersion(*cfg.Microversion); err != nil {
			return nil, wrap(summaryInvalid("invalid explicit summary microversion: %v", err))
		}
	}
	result := &VolumeSummaryRecord{}
	negotiated := source.Client.Microversion
	if negotiated != "" {
		if _, err = microversions.ParseVersion(negotiated); err != nil {
			return nil, wrap(summaryInvalid("invalid selected summary microversion: %v", err))
		}
	} else if !cfg.SingleFetch || cfg.Microversion == nil {
		var pages []*rest.Response
		negotiated, pages, err = cinderrequest.Negotiate(ctx, source, "3.36")
		for _, page := range pages {
			result.Discovery = append(result.Discovery, summaryObservation(page))
		}
		if err = errors.Join(err, guard(ctx)); err != nil {
			return result, wrap(err)
		}
	}
	fields := make(map[string]json.RawMessage)
	fetch := func(version string, query url.Values, key string) error {
		headers := rest.RequestHeaderPolicy{Values: maps.Clone(source.Client.MoreHeaders), Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version"}}
		if headers.Values == nil {
			headers.Values = make(map[string]string)
		}
		for k := range headers.Values {
			if strings.EqualFold(k, "OpenStack-API-Version") || strings.EqualFold(k, "X-OpenStack-Volume-API-Version") {
				delete(headers.Values, k)
			}
		}
		if version != "" {
			headers.Values["OpenStack-API-Version"] = "volume " + version
			headers.Values["X-OpenStack-Volume-API-Version"] = version
		} else {
			headers.Absent = append(headers.Absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
		}
		source.Client.Microversion = version
		for k := range source.Client.MoreHeaders {
			if strings.EqualFold(k, "OpenStack-API-Version") || strings.EqualFold(k, "X-OpenStack-Volume-API-Version") {
				delete(source.Client.MoreHeaders, k)
			}
		}
		target := source.Client.ServiceURL("volumes", "summary")
		if len(query) > 0 {
			target += "?" + query.Encode()
		}
		response, cause := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodGet, target, nil, headers, rest.RejectionPolicy{Codes: summaryCodes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, summaryCodes(100, 400)...)
		if response == nil {
			response = summaryNativeResponse(cause)
		}
		if response != nil {
			result.Envelope = bytes.Clone(response.Body)
			result.Header = response.Header.Clone()
			result.StatusCode = response.StatusCode
			result.Requests = append(result.Requests, summaryObservation(response))
		}
		result.Microversion = version
		result.Resource, result.Wire = nil, nil
		if cause = errors.Join(cause, guard(ctx)); cause != nil {
			return cause
		}
		result.Resource, result.Wire, cause = summaryView(response, key, fields)
		if cause = errors.Join(cause, guard(ctx)); cause != nil {
			return response.Fail(cause)
		}
		return nil
	}
	if !cfg.SingleFetch {
		if err = fetch(negotiated, nil, ""); err != nil {
			return result, wrap(err)
		}
	}
	version := negotiated
	if cfg.Microversion != nil {
		version = *cfg.Microversion
	}
	if err = fetch(version, query, "volume-summary"); err != nil {
		return result, wrap(err)
	}
	return result, wrap(guard(ctx))
}
