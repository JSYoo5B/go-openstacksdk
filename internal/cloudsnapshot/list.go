package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// List owns source/query policy before making the first request. Logical rows
// are published together only after successful exhaustion and projection.
func List(ctx context.Context, client *gophercloud.ServiceClient, options ...ListOption) (*ListResult, error) {
	return listResource(ctx, client, snapshotReadSchema(), options...)
}

func listResource(ctx context.Context, client *gophercloud.ServiceClient, schema readSchema, options ...ListOption) (*ListResult, error) {
	p, err := captureWithSchema(ctx, client, schema)
	if err != nil {
		return nil, wrapRead(ctx, schema, schema.listOperation, err)
	}
	owned, err := prepare(ctx, options, cloneList, func() error { return p.source.Guard(ctx) })
	var policy *listPolicy
	if err == nil {
		policy, err = compileListFor(owned, schema)
	}
	if err == nil {
		err = p.ownLocation(ctx, owned.Location)
	}
	if err != nil {
		return nil, wrapRead(ctx, schema, schema.listOperation, err)
	}
	result := &ListResult{}
	var values []*entry
	err = p.readList(ctx, policy, result, func(value *entry) (bool, error) { values = append(values, value); return true, nil })
	if err != nil {
		return result, wrapRead(ctx, schema, schema.listOperation, err)
	}
	rows := make([]json.RawMessage, len(values))
	for i, value := range values {
		rows[i] = value.view
	}
	selected, err := p.projectList(ctx, rows, policy.expression)
	if err != nil {
		return result, wrapRead(ctx, schema, schema.listOperation, fmt.Errorf("%w: list expression: %w", resource.ErrInvalidOption, err))
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Snapshots = make([]*resource.RawResource, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Snapshots[i] = values[index].resource
		}
	}
	return result, nil
}

func (p *reader) readList(ctx context.Context, policy *listPolicy, proof *ListResult, yield func(*entry) (bool, error)) error {
	headers := maps.Clone(policy.headers)
	if headers == nil {
		headers = make(map[string]string)
	}
	hasAccept := false
	for key := range headers {
		if strings.EqualFold(key, "Accept") {
			hasAccept = true
		}
	}
	if !hasAccept {
		headers["Accept"] = "application/json"
	}
	if err := p.source.WithPolicy(ctx, policy.microversion, headers); err != nil {
		return err
	}
	parts := []string{p.schema.route}
	if policy.detailed {
		parts = append(parts, "detail")
	}
	collection, err := p.target(parts...)
	if err != nil {
		return err
	}
	query := cloneQuery(policy.query)
	encoded, err := encodeQuery(query)
	if err != nil {
		return err
	}
	current := collection
	if len(encoded) != 0 {
		current += "?" + encoded.Encode()
	}
	seen := make(map[string]struct{})
	count := 0
	for {
		if err := p.source.Guard(ctx); err != nil {
			return err
		}
		key, err := snapshotPageKey(current, collection)
		if err != nil {
			return err
		}
		if _, exists := seen[key]; exists {
			return &resource.PaginationCycleError{URL: current}
		}
		seen[key] = struct{}{}
		wire, err := p.source.Get(ctx, current, sourceCodes()...)
		if page := observed(wire); page != nil {
			proof.Pages = append(proof.Pages, page)
		}
		if err != nil {
			return err
		}
		fields, rows, err := listObjectsFor(wire, p.schema)
		if err != nil {
			return err
		}
		var lastID json.RawMessage
		for _, row := range rows {
			if err := p.source.Guard(ctx); err != nil {
				return wire.Fail(err)
			}
			// Source checks after HTTP and full response JSON decoding. Reaching
			// the maximum on a page's last row can still cause another request.
			stop, err := maxReached(count, policy.maximum)
			if err != nil {
				return err
			}
			if stop {
				return p.source.Guard(ctx)
			}
			value, err := p.materialize(ctx, row, nil, true, wire)
			if err != nil {
				return err
			}
			var normalized map[string]json.RawMessage
			if err := json.Unmarshal(value.view, &normalized); err != nil {
				return wire.Fail(err)
			}
			lastID = bytes.Clone(normalized["id"])
			matched, err := matchLocal(ctx, value.view, policy.local, func() error { return p.source.Guard(ctx) })
			if err != nil {
				return fmt.Errorf("%w: local Cinder resource filter: %w", resource.ErrInvalidOption, err)
			}
			if matched {
				more, err := yield(value)
				if err != nil {
					return err
				}
				if !more {
					return p.source.Guard(ctx)
				}
			}
			count++
		}
		if len(rows) == 0 || !policy.paginated {
			return p.source.Guard(ctx)
		}
		next, err := resourceNext(fields, wire.Header, p.schema.plural)
		if err != nil {
			return wire.Fail(err)
		}
		target, nextQuery, more, err := snapshotContinuation(current, next, collection, query, policy.limit, lastID)
		if err != nil {
			return wire.Fail(err)
		}
		if !more {
			return p.source.Guard(ctx)
		}
		if _, err := snapshotPageKey(target, collection); err != nil {
			return wire.Fail(err)
		}
		current, query = target, nextQuery
	}
}

func cloneQuery(values map[string]json.RawMessage) map[string]json.RawMessage {
	copy := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		copy[key] = bytes.Clone(raw)
	}
	return copy
}
