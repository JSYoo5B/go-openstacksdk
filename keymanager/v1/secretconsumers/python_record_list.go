package secretconsumers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (a *API) ListRecords(ctx context.Context, secretID string, options ...ListRecordOption) iter.Seq2[*ConsumerRecord, error] {
	return (&SecretScope{client: a.RawClient(), secretID: secretID}).ListRecords(ctx, options...)
}
func (a *API) AllRecords(ctx context.Context, secretID string, options ...ListRecordOption) ([]*ConsumerRecord, error) {
	return (&SecretScope{client: a.RawClient(), secretID: secretID}).AllRecords(ctx, options...)
}

// ListRecords is lazy and reusable. Local filters run on declared descriptor
// values before yield; MaxItems counts raw rows, including nonmatching rows.
// The SDK owns headers, transport, fixed-parent continuation and cycle checks.
func (s *SecretScope) ListRecords(ctx context.Context, options ...ListRecordOption) iter.Seq2[*ConsumerRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ConsumerRecord, error) bool) {
		fail := func(err error) { yield(nil, request.Wrap("ListRecords", kind, err)) }
		id := ""
		if s != nil {
			id = s.secretID
		}
		source, err := recordSource(ctx, s.RawClient(), id)
		if err != nil {
			fail(err)
			return
		}
		guard := recordGuard(source)
		own := func(config *request.Config[ListRecordOpts]) error {
			fields, err := recordAttributes(config.Options.Query, true)
			if err != nil {
				return err
			}
			attributes, err := decodeRecordAttributes(fields)
			if err != nil {
				return err
			}
			config.Options.Query = attributes
			if config.Options.Paginated != nil {
				copy := *config.Options.Paginated
				config.Options.Paginated = &copy
			}
			ownRecordConfig(config)
			return nil
		}
		prepared := make([]ListRecordOption, len(owned))
		for i, option := range owned {
			apply := option
			prepared[i] = func(config *request.Config[ListRecordOpts]) error {
				if err := guard(ctx); err != nil {
					return err
				}
				if apply == nil {
					return recordInvalid("nil consumer list record option")
				}
				if err := own(config); err != nil {
					return errors.Join(err, guard(ctx))
				}
				applied := apply(config)
				return errors.Join(applied, own(config), guard(ctx))
			}
		}
		config, err := request.Apply(ListRecordOpts{}, prepared...)
		if err == nil {
			err = own(&config)
		}
		if err == nil {
			err = request.ValidateCapabilities(config, false, false, true)
		}
		if err == nil {
			err = recordHeaderPolicy(config.Headers)
		}
		if err == nil && config.Options.MaxItems < 0 {
			err = recordInvalid("consumer max items must be nonnegative")
		}
		if err == nil {
			err = source.WithPolicy(ctx, nil, config.Headers)
		}
		fields := map[string]json.RawMessage{}
		if err == nil {
			fields, err = recordAttributes(config.Options.Query, true)
		}
		err = errors.Join(err, guard(ctx))
		if err != nil {
			fail(err)
			return
		}
		filters := make(map[string]json.RawMessage)
		query := make(url.Values)
		for key, raw := range fields {
			if recordBodyField(key) {
				filters[key] = bytes.Clone(raw)
				continue
			}
			values, prior := cloudfilter.RequestQueryValues(raw)
			if prior != nil {
				fail(prior)
				return
			}
			if len(values) != 0 {
				query[key] = values
			}
		}
		limitRaw := fields["limit"]
		limitTruthy, err := cloudfilter.PythonTruthy(limitRaw)
		if err != nil {
			fail(err)
			return
		}
		if config.Options.MaxItems > 0 && !limitTruthy {
			query.Set("limit", strconv.Itoa(config.Options.MaxItems))
			limitTruthy = true
		}
		// Source Resource.list explicitly supplies application/json. MoreHeaders
		// otherwise wins in Gophercloud, so change only the private captured copy.
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "Accept") {
				delete(source.Client.MoreHeaders, key)
			}
		}
		target := source.Client.ServiceURL("secrets", url.PathEscape(id), "consumers")
		base, err := url.Parse(target)
		if err != nil {
			fail(err)
			return
		}
		current := *base
		current.RawQuery = query.Encode()
		visited := map[string]bool{current.String(): true}
		consumed := 0
		for page := 0; ; page++ {
			response, prior := rest.DoJSONGuardedRejectionsHeaders(ctx, &source.Client, guard, http.MethodGet, current.String(), nil, map[string]string{"Accept": "application/json"}, rest.RejectionPolicy{Codes: recordRejectionCodes(), PreserveCleanRetry: true}, recordSuccessCodes()...)
			if prior != nil {
				fail(prior)
				return
			}
			envelope, rows, prior := consumerRecordRows(response.Body)
			if prior != nil {
				fail(response.Fail(prior))
				return
			}
			var marker json.RawMessage
			for _, raw := range rows {
				if config.Options.MaxItems > 0 && consumed >= config.Options.MaxItems {
					return
				}
				if err := guard(ctx); err != nil {
					fail(response.Fail(err))
					return
				}
				var wire resource.RawResource
				if err := resource.DecodeObject(raw, &wire, &wire.Metadata); err != nil {
					fail(response.Fail(err))
					return
				}
				wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
				view := recordView(wire.Body, &wire.Metadata)
				marker = bytes.Clone(view.Body["id"])
				matches, err := consumerRecordMatches(view.Body, filters)
				if err != nil {
					fail(response.Fail(err))
					return
				}
				consumed++
				if matches {
					value := &ConsumerRecord{Resource: view, Wire: wire.Clone(), SecretID: id, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
					if !yield(value, nil) {
						return
					}
				}
				if err := guard(ctx); err != nil {
					fail(response.Fail(err))
					return
				}
				if config.Options.MaxItems > 0 && consumed >= config.Options.MaxItems {
					return
				}
			}
			if err := guard(ctx); err != nil {
				fail(response.Fail(err))
				return
			}
			if len(rows) == 0 || config.Options.Paginated != nil && !*config.Options.Paginated {
				return
			}
			next, err := consumerRecordNext(base, &current, envelope, response.Header, marker, query["limit"], limitTruthy, page == 0)
			if err != nil {
				fail(response.Fail(err))
				return
			}
			if next == nil {
				return
			}
			if visited[next.String()] {
				fail(response.Fail(&resource.PaginationCycleError{URL: next.String()}))
				return
			}
			visited[next.String()] = true
			current = *next
		}
	}
}
func (s *SecretScope) AllRecords(ctx context.Context, options ...ListRecordOption) ([]*ConsumerRecord, error) {
	var result []*ConsumerRecord
	for value, err := range s.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func consumerRecordRows(body json.RawMessage) (map[string]json.RawMessage, []json.RawMessage, error) {
	if !utf8.Valid(body) {
		return nil, nil, recordInvalid("consumer page must be valid UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, nil, err
	}
	if envelope == nil {
		return nil, nil, recordInvalid("consumer page must be an object")
	}
	raw := bytes.TrimSpace(envelope["consumers"])
	if len(raw) > 0 && raw[0] == '{' {
		return envelope, []json.RawMessage{bytes.Clone(raw)}, nil
	}
	if len(raw) == 0 || raw[0] != '[' {
		return nil, nil, recordInvalid("consumer page requires consumers array or object")
	}
	var rows []json.RawMessage
	err := json.Unmarshal(raw, &rows)
	return envelope, rows, err
}
func consumerRecordMatches(body, filters map[string]json.RawMessage) (bool, error) {
	for key, expected := range filters {
		actual := body[key]
		if actual == nil {
			actual = json.RawMessage("null")
		}
		matches, err := consumerRecordSubset(actual, expected)
		if err != nil || !matches {
			return matches, err
		}
	}
	return true, nil
}
func consumerRecordSubset(actual, expected json.RawMessage) (bool, error) {
	raw := bytes.TrimSpace(expected)
	if len(raw) > 0 && raw[0] == '{' {
		truthy, err := cloudfilter.PythonTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		var left, right map[string]json.RawMessage
		if err := json.Unmarshal(actual, &left); err != nil {
			return false, recordInvalid("object Body filter requires a dictionary descriptor value")
		}
		if err := json.Unmarshal(expected, &right); err != nil {
			return false, err
		}
		for key, filter := range right {
			value := left[key]
			if value == nil {
				value = json.RawMessage("null")
			}
			match, err := consumerRecordSubset(value, filter)
			if err != nil || !match {
				return match, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, expected)
}

// consumerRecordNext follows Source link priority and marker fallback while
// retaining the service origin, fixed parent, initial filters and forward offset.
// No returned resource/ref/URI state grants routing authority.
func consumerRecordNext(base, current *url.URL, envelope map[string]json.RawMessage, headers http.Header, marker json.RawMessage, limit []string, limitTruthy, first bool) (*url.URL, error) {
	link := ""
	key := "consumers_links"
	if _, present := envelope["links"]; present {
		key = "links"
	}
	if raw, present := envelope[key]; present && len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '[' {
		var links []resource.Link
		if err := json.Unmarshal(raw, &links); err != nil {
			return nil, err
		}
		for _, value := range links {
			if value.Rel == "next" && value.Href != "" {
				link = value.Href
				break
			}
		}
	}
	if link == "" {
		raw := envelope["next"]
		truthy, err := cloudfilter.PythonTruthy(raw)
		if err != nil {
			return nil, err
		}
		if truthy {
			if err := json.Unmarshal(raw, &link); err != nil {
				return nil, err
			}
		}
	}
	if link == "" {
		for _, header := range headers.Values("Link") {
			links, err := rest.HeaderNextLinks(header)
			if err != nil {
				return nil, err
			}
			if len(links) != 0 {
				link = links[0]
				break
			}
		}
	}
	if link == "" {
		if !limitTruthy {
			return nil, nil
		}
		values, err := cloudfilter.RequestQueryValues(marker)
		if err != nil {
			return nil, err
		}
		if len(values) != 1 || values[0] == "" {
			return nil, &resource.PaginationCycleError{URL: current.String()}
		}
		next := *current
		query := next.Query()
		query.Set("marker", values[0])
		query["limit"] = slices.Clone(limit)
		next.RawQuery = query.Encode()
		return &next, nil
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	// Resource strips an initial /v... prefix. Admit only the exact alias of
	// this same fixed collection when a deployment adds a service prefix.
	alias := "/v1/secrets/" + strings.SplitN(base.EscapedPath(), "/secrets/", 2)[1]
	if parsed.Scheme == "" && parsed.Host == "" && parsed.EscapedPath() == alias {
		parsed.Path, parsed.RawPath = base.Path, base.RawPath
	}
	next := current.ResolveReference(parsed)
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, base.Scheme) || !strings.EqualFold(next.Host, base.Host) || next.EscapedPath() != base.EscapedPath() {
		return nil, recordInvalid("consumer pagination changes service origin or fixed parent")
	}
	values, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return nil, err
	}
	previous := current.Query()
	for key, newValues := range values {
		switch key {
		case "marker":
			if len(newValues) != 1 || newValues[0] == "" {
				return nil, recordInvalid("consumer pagination requires one nonempty marker")
			}
		case "offset":
			newOffset, err := consumerRecordOffset(newValues)
			if err != nil {
				return nil, err
			}
			oldOffset := uint64(0)
			if previous.Has("offset") {
				oldOffset, err = consumerRecordOffset(previous["offset"])
				if err != nil {
					return nil, err
				}
			}
			if newOffset <= oldOffset {
				return nil, &resource.PaginationCycleError{URL: next.String()}
			}
		case "limit":
			if !slices.Equal(newValues, previous[key]) {
				if !first {
					return nil, recordInvalid("consumer pagination changes established limit")
				}
				nextLimit, err := consumerRecordOffset(newValues)
				if err != nil || nextLimit == 0 {
					return nil, recordInvalid("consumer pagination limit must be positive")
				}
				if previous.Has(key) {
					oldLimit, err := consumerRecordOffset(previous[key])
					if err != nil || nextLimit > oldLimit {
						return nil, recordInvalid("consumer pagination increases requested limit")
					}
				}
			}
		default:
			return nil, recordInvalid("consumer pagination introduces unknown query filters")
		}
	}
	for key, oldValues := range previous {
		if !values.Has(key) {
			values[key] = slices.Clone(oldValues)
		}
	}
	next.RawQuery = values.Encode()
	return next, nil
}
func consumerRecordOffset(values []string) (uint64, error) {
	if len(values) != 1 || values[0] == "" {
		return 0, recordInvalid("consumer pagination requires one decimal cursor")
	}
	for _, value := range values[0] {
		if value < '0' || value > '9' {
			return 0, recordInvalid("consumer pagination requires a nonnegative decimal cursor")
		}
	}
	parsed, err := strconv.ParseUint(values[0], 10, 64)
	if err != nil {
		return 0, recordInvalid("consumer pagination cursor out of range")
	}
	return parsed, nil
}
