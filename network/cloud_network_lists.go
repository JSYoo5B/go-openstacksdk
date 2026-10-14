package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type NetworkCloudListOpts struct {
	Filters          json.RawMessage
	Headers          map[string]string
	Microversion     *string
	CurrentLocation  *resource.CloudLocation
	ServiceAvailable *bool
}
type NetworkCloudListOption func(*NetworkCloudListOpts) error

func copyNetworkCloudList(v NetworkCloudListOpts) NetworkCloudListOpts {
	v.Filters = bytes.Clone(v.Filters)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.CurrentLocation != nil {
		x := v.CurrentLocation.Clone()
		v.CurrentLocation = &x
	}
	if v.ServiceAvailable != nil {
		x := *v.ServiceAvailable
		v.ServiceAvailable = &x
	}
	return v
}
func WithNetworkCloudListOpts(v NetworkCloudListOpts) NetworkCloudListOption {
	owned := copyNetworkCloudList(v)
	return func(c *NetworkCloudListOpts) error { *c = copyNetworkCloudList(owned); return nil }
}
func WithNetworkCloudFilters(raw json.RawMessage) NetworkCloudListOption {
	owned := bytes.Clone(raw)
	return func(c *NetworkCloudListOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithNetworkCloudHeaders(v map[string]string) NetworkCloudListOption {
	owned := maps.Clone(v)
	return func(c *NetworkCloudListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithNetworkCloudMicroversion(v string) NetworkCloudListOption {
	return func(c *NetworkCloudListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithNetworkCloudCurrentLocation(v resource.CloudLocation) NetworkCloudListOption {
	owned := v.Clone()
	return func(c *NetworkCloudListOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}
func WithNetworkCloudServiceAvailable(v bool) NetworkCloudListOption {
	return func(c *NetworkCloudListOpts) error { x := v; c.ServiceAvailable = &x; return nil }
}

// Networks are populated only for ordinary generator output. Values/JSON can
// contain arbitrary Proxy list() items; nested generators remain opaque/lazy.
// Errors discard collected public outputs and retain actual proof in the error.
type NetworkCloudRecords struct {
	Networks []*networks.NetworkRecord
	Values   []any
	JSON     json.RawMessage
}

func networkCloudWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "networks", cloudread.ContextError(ctx, err))
}
func (s *Service) networkCloudLists(ctx context.Context, identity string, search bool, options []NetworkCloudListOption) (*NetworkCloudRecords, error) {
	op := "ListNetworkRecords"
	if search {
		op = "SearchNetworkRecords"
	}
	fail := func(e error) (*NetworkCloudRecords, error) { return nil, networkCloudWrap(ctx, op, e) }
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	var source *cloudread.Source
	var sourceErr error
	var selectedAPI *networkv2.Service
	// Capture configured source/header facts before option callbacks. A missing
	// service needs no source when list's concrete absence option is selected.
	if s != nil && s.client != nil && s.API != nil && s.API.RawClient() == s.client {
		selectedAPI = s.API
		source, sourceErr = cloudread.Capture(ctx, s.client, "network")
	}
	guard := func(ctx context.Context) error {
		if err := cloudread.Context(ctx); err != nil {
			return err
		}
		if source == nil {
			return nil
		}
		if s.API != selectedAPI || s.API.RawClient() != s.client {
			return qosReadInvalid("selected network service/API changed")
		}
		return source.Guard(ctx)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := copyNetworkCloudList(NetworkCloudListOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return fail(err)
		}
		if apply == nil {
			return fail(qosReadInvalid("nil network cloud list option"))
		}
		next := copyNetworkCloudList(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return fail(err)
		}
		cfg = copyNetworkCloudList(next)
	}
	if !search && cfg.ServiceAvailable != nil && !*cfg.ServiceAvailable {
		return &NetworkCloudRecords{Networks: []*networks.NetworkRecord{}, Values: []any{}, JSON: json.RawMessage("[]")}, nil
	}
	members, err := qosPolicyCloudPairs(cfg.Filters, search)
	if err != nil {
		return fail(err)
	}
	if sourceErr != nil {
		return fail(sourceErr)
	}
	if source == nil {
		return fail(qosReadInvalid("configured Neutron service required"))
	}
	ordered := make([]cloudfilter.JSONMember, 0, len(members)+1)
	if search && identity != "" {
		raw, _ := json.Marshal(identity)
		ordered = append(ordered, cloudfilter.JSONMember{Key: "name", Value: raw})
	}
	positions := make(map[string]int)
	for i, m := range ordered {
		positions[m.Key] = i
	}
	for _, m := range members {
		if i, ok := positions[m.Key]; ok {
			ordered[i].Value = bytes.Clone(m.Value)
		} else {
			positions[m.Key] = len(ordered)
			ordered = append(ordered, m)
		}
	}
	leaf := make([]networks.RecordListOption, 0, len(ordered)+2)
	for _, m := range ordered {
		leaf = append(leaf, networks.WithCloudRecordListQuery(m.Key, m.Value))
	}
	if cfg.Headers != nil {
		leaf = append(leaf, networks.WithRecordListHeaders(cfg.Headers))
	}
	if cfg.Microversion != nil {
		leaf = append(leaf, networks.WithRecordListMicroversion(*cfg.Microversion))
	}
	api := networks.New(&source.Client)
	proxy, err := api.ProxyCloudListRecords(ctx, cfg.CurrentLocation, leaf...)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(err)
	}
	records := make([]*networks.NetworkRecord, 0)
	values := make([]any, 0)
	switch value := proxy.Value.(type) {
	case *networks.RecordListGenerator:
		for row, cause := range value.Records {
			if cause = errors.Join(cause, guard(ctx)); cause != nil {
				return fail(cause)
			}
			records = append(records, row)
			values = append(values, row.Resource)
		}
	case []any:
		values = append(values, value...)
	case string:
		for _, r := range value {
			values = append(values, string(r))
		}
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			values = append(values, key)
		}
	default:
		return fail(qosReadInvalid("Proxy result %T is not iterable by cloud list()", proxy.Value))
	}
	if err := guard(ctx); err != nil {
		return fail(err)
	}
	result := &NetworkCloudRecords{Networks: records, Values: values}
	if !networkCloudOpaque(values) {
		raw, err := json.Marshal(values)
		if err != nil {
			return fail(err)
		}
		result.JSON = bytes.Clone(raw)
	}
	if err := guard(ctx); err != nil {
		return fail(err)
	}
	return result, nil
}
func networkCloudOpaque(v any) bool {
	switch x := v.(type) {
	case *networks.RecordListGenerator:
		return true
	case []any:
		for _, v := range x {
			if networkCloudOpaque(v) {
				return true
			}
		}
	case map[string]any:
		for _, v := range x {
			if networkCloudOpaque(v) {
				return true
			}
		}
	}
	return false
}
func (s *Service) ListNetworkRecords(ctx context.Context, options ...NetworkCloudListOption) (*NetworkCloudRecords, error) {
	return s.networkCloudLists(ctx, "", false, options)
}
func (s *Service) SearchNetworkRecords(ctx context.Context, identity string, options ...NetworkCloudListOption) (*NetworkCloudRecords, error) {
	return s.networkCloudLists(ctx, identity, true, options)
}
