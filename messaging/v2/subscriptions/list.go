package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (s *QueueScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Subscription, error] {
	owned := append([]ListOption(nil), options...)
	return func(yield func(*Subscription, error) bool) {
		fail := func(err error) { yield(nil, request.Wrap("List", kind, err)) }
		if err := s.validate(ctx); err != nil {
			fail(err)
			return
		}
		config, err := request.Apply(ListOpts{}, owned...)
		if err == nil {
			err = request.ValidateCapabilities(config, false, true, true)
		}
		value := copyList(config.Options)
		if err == nil && (value.Limit < 0 || value.MaxItems < 0) {
			err = fmt.Errorf("%w: list limits must be nonnegative", resource.ErrInvalidOption)
		}
		query := make(url.Values)
		if value.Limit > 0 {
			query.Set("limit", strconv.Itoa(value.Limit))
		}
		if value.Marker != "" {
			query.Set("marker", value.Marker)
		}
		if err == nil {
			for key, values := range config.Query {
				switch strings.ToLower(key) {
				case "queue_name", "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters", "client_id", "project_id":
					err = fmt.Errorf("%w: query %q is a scope or SDK control", resource.ErrInvalidOption, key)
				}
				if strings.TrimSpace(key) == "" {
					err = fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
				}
				query[key] = append([]string(nil), values...)
			}
		}
		limit := 0
		if err == nil && query.Has("limit") {
			values := query["limit"]
			if len(values) != 1 {
				err = fmt.Errorf("%w: one positive limit is required", resource.ErrInvalidOption)
			} else {
				limit, err = strconv.Atoi(values[0])
				if err != nil || limit < 1 {
					err = fmt.Errorf("%w: positive limit is required", resource.ErrInvalidOption)
				}
			}
		}
		if err == nil && query.Has("marker") {
			if values := query["marker"]; len(values) != 1 || strings.TrimSpace(values[0]) == "" {
				err = fmt.Errorf("%w: one nonempty marker is required", resource.ErrInvalidOption)
			}
		}
		var headers map[string]string
		if err == nil {
			headers, err = s.headers(value.RequestIdentity, config.Headers)
		}
		if err != nil {
			fail(err)
			return
		}
		endpoint, err := url.Parse(s.endpoint(""))
		if err != nil {
			fail(err)
			return
		}
		if err := rest.ValidateTarget(s.client, endpoint.String()); err != nil {
			fail(err)
			return
		}
		markers := map[string]bool{}
		if marker := query.Get("marker"); marker != "" {
			markers[marker] = true
		}
		consumed := 0
		for {
			endpoint.RawQuery = query.Encode()
			response, err := s.do(ctx, "GET", endpoint.String(), nil, headers, 200)
			if err != nil {
				fail(err)
				return
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(response.Body, &fields); err != nil {
				fail(response.Fail(err))
				return
			}
			raw, present := fields["subscriptions"]
			var rows []json.RawMessage
			if !present || len(raw) == 0 || strings.TrimSpace(string(raw))[0] != '[' {
				fail(response.Fail(fmt.Errorf("subscriptions array is required")))
				return
			}
			if err := json.Unmarshal(raw, &rows); err != nil {
				fail(response.Fail(err))
				return
			}
			marker := ""
			for _, row := range rows {
				if err := ctx.Err(); err != nil {
					fail(response.Fail(err))
					return
				}
				var item Subscription
				if err := json.Unmarshal(row, &item); err != nil {
					fail(response.Fail(err))
					return
				}
				item.Header, item.StatusCode = response.Header.Clone(), response.StatusCode
				marker = item.ID
				consumed++
				if !yield(&item, nil) {
					return
				}
				if value.MaxItems > 0 && consumed >= value.MaxItems {
					if err := ctx.Err(); err != nil {
						fail(response.Fail(err))
					}
					return
				}
			}
			if err := ctx.Err(); err != nil {
				fail(response.Fail(err))
				return
			}
			if value.Paginated != nil && !*value.Paginated || len(rows) == 0 || limit > 0 && len(rows) < limit {
				return
			}
			if strings.TrimSpace(marker) == "" {
				fail(response.Fail(fmt.Errorf("%w: last subscription requires an id or subscription_id marker", resource.ErrInvalidOption)))
				return
			}
			if markers[marker] {
				fail(response.Fail(resource.ErrPaginationCycle))
				return
			}
			markers[marker] = true
			limit = len(rows)
			query.Set("limit", strconv.Itoa(limit))
			query.Set("marker", marker)
		}
	}
}
func (s *QueueScope) All(ctx context.Context, options ...ListOption) ([]*Subscription, error) {
	var values []*Subscription
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
