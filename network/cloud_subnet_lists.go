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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// SubnetReadOpts owns cloud filters and SDK transport/location/service facts.
// NetworkAvailable=false affects List only; Search directly uses the Proxy.
type SubnetReadOpts struct {
	Filters                json.RawMessage
	Headers                map[string]string
	Microversion, BasePath *string
	CurrentLocation        *resource.CloudLocation
	NetworkAvailable       *bool
	headersPresent         bool
}
type SubnetReadOption func(*SubnetReadOpts) error

func cloneSubnetRead(v SubnetReadOpts) SubnetReadOpts {
	v.Filters = bytes.Clone(v.Filters)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.BasePath != nil {
		x := *v.BasePath
		v.BasePath = &x
	}
	if v.CurrentLocation != nil {
		x := v.CurrentLocation.Clone()
		v.CurrentLocation = &x
	}
	if v.NetworkAvailable != nil {
		x := *v.NetworkAvailable
		v.NetworkAvailable = &x
	}
	return v
}
func WithSubnetReadOpts(v SubnetReadOpts) SubnetReadOption {
	v.headersPresent = v.headersPresent || v.Headers != nil
	owned := cloneSubnetRead(v)
	return func(c *SubnetReadOpts) error {
		next := cloneSubnetRead(owned)
		if next.CurrentLocation == nil && c.CurrentLocation != nil {
			x := c.CurrentLocation.Clone()
			next.CurrentLocation = &x
		}
		if next.NetworkAvailable == nil && c.NetworkAvailable != nil {
			x := *c.NetworkAvailable
			next.NetworkAvailable = &x
		}
		*c = next
		return nil
	}
}
func WithSubnetReadFiltersJSON(v json.RawMessage) SubnetReadOption {
	owned := bytes.Clone(v)
	return func(c *SubnetReadOpts) error { c.Filters = bytes.Clone(owned); return nil }
}
func WithSubnetReadHeaders(v map[string]string) SubnetReadOption {
	owned := maps.Clone(v)
	return func(c *SubnetReadOpts) error { c.Headers = maps.Clone(owned); c.headersPresent = true; return nil }
}
func WithSubnetReadMicroversion(v string) SubnetReadOption {
	return func(c *SubnetReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithSubnetReadBasePath(v string) SubnetReadOption {
	return func(c *SubnetReadOpts) error { x := v; c.BasePath = &x; return nil }
}
func WithSubnetReadCurrentLocation(v resource.CloudLocation) SubnetReadOption {
	owned := v.Clone()
	return func(c *SubnetReadOpts) error { x := owned.Clone(); c.CurrentLocation = &x; return nil }
}
func WithSubnetNetworkAvailable(v bool) SubnetReadOption {
	return func(c *SubnetReadOpts) error { x := v; c.NetworkAvailable = &x; return nil }
}

// SubnetRecords distinguishes cloud list() values from actual resource reads.
// Inventory/LastResponse retain partial actual proof on error; Subnets/Value/JSON
// are committed only after a complete conversion. Literals invent no resource.
type SubnetRecords struct {
	Subnets      []*subnets.SubnetRecord
	Inventory    []*subnets.SubnetRecord
	LastResponse *subnets.SubnetRecord
	Value        []any
	JSON         json.RawMessage
}

func subnetReadCopy(v *subnets.SubnetRecord) *subnets.SubnetRecord {
	if v == nil {
		return nil
	}
	return &subnets.SubnetRecord{Resource: v.Resource.Clone(), Wire: v.Wire.Clone(), Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
}
func subnetReadReceipt(err error) *subnets.SubnetRecord {
	var physical *resource.ResponseError
	if errors.As(err, &physical) && physical != nil {
		return &subnets.SubnetRecord{Envelope: bytes.Clone(physical.Body), Header: physical.Header.Clone(), StatusCode: physical.StatusCode}
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) {
		return &subnets.SubnetRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}
func (s *Service) prepareSubnetRead(ctx context.Context, options []SubnetReadOption) (*networkv2.Service, SubnetReadOpts, func(context.Context) error, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, SubnetReadOpts{}, nil, err
	}
	var source *cloudread.Source
	var api *networkv2.Service
	var selectedAPI *networkv2.Service
	var selectedSubnets *subnets.API
	var selectedClient *gophercloud.ServiceClient
	if s != nil && s.client != nil && s.API != nil && s.API.RawClient() == s.client {
		selectedAPI, selectedSubnets, selectedClient = s.API, s.API.Subnets, s.client
		var err error
		source, err = cloudread.Capture(ctx, s.client, "network")
		if err != nil {
			return nil, SubnetReadOpts{}, nil, err
		}
		api = networkv2.New(&source.Client)
	}
	outerGuard := rest.OperationGuard(ctx)
	outerCheck := func(ctx context.Context) error {
		if outerGuard != nil {
			return outerGuard(ctx)
		}
		return nil
	}
	var changed error
	guard := func(ctx context.Context) error {
		if err := cloudread.Context(ctx); err != nil {
			return err
		}
		if source == nil {
			return outerCheck(ctx)
		}
		if changed == nil && (s.API != selectedAPI || s.API.Subnets != selectedSubnets || s.client != selectedClient || s.API.RawClient() != selectedClient) {
			changed = qosReadInvalid("selected subnet list service/API changed")
		}
		return errors.Join(changed, source.Guard(ctx), outerCheck(ctx))
	}
	cfg := cloneSubnetRead(SubnetReadOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, cfg, guard, err
		}
		if apply == nil {
			return nil, cfg, guard, qosReadInvalid("nil subnet read option")
		}
		next := cloneSubnetRead(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, cfg, guard, err
		}
		cfg = cloneSubnetRead(next)
	}
	return api, cfg, guard, guard(ctx)
}

// ListSubnetRecords checks Source network availability before filter evaluation.
// A false availability with a nil service is the owned clean-catalog absence lane.
func (s *Service) ListSubnetRecords(ctx context.Context, options ...SubnetReadOption) (*SubnetRecords, error) {
	return s.subnetCloudList(ctx, "ListSubnetRecords", "", false, options)
}

// SearchSubnetRecords pushes a truthy identity as a literal name query, then
// lets dict.update(filters) replace it. No ID/glob/local cloud search is added.
func (s *Service) SearchSubnetRecords(ctx context.Context, nameOrID string, options ...SubnetReadOption) (*SubnetRecords, error) {
	return s.subnetCloudList(ctx, "SearchSubnetRecords", nameOrID, true, options)
}
func subnetReadOpaque(value any) bool {
	switch v := value.(type) {
	case *subnets.RecordListGenerator:
		return true
	case []any:
		for _, item := range v {
			if subnetReadOpaque(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range v {
			if subnetReadOpaque(item) {
				return true
			}
		}
	}
	return false
}
func subnetReadLocation(v *subnets.SubnetRecord, location *resource.CloudLocation) error {
	if location == nil {
		return nil
	}
	raw, err := cloudNetworkReadLocation(v.Resource.Body, location)
	if err != nil {
		return err
	}
	var computed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &computed); err != nil {
		return err
	}
	zone := v.Resource.Body["availability_zone"]
	if zone == nil {
		zone = json.RawMessage("null")
	}
	computed["zone"] = bytes.Clone(zone)
	raw, err = json.Marshal(computed)
	if err != nil {
		return err
	}
	v.Resource.Body["location"] = raw
	return nil
}
func subnetReadBindGenerators(values []any, location *resource.CloudLocation, guard func(context.Context) error, ctx context.Context) {
	seen := make(map[*subnets.RecordListGenerator]bool)
	var bind func(any)
	bind = func(value any) {
		switch v := value.(type) {
		case *subnets.RecordListGenerator:
			if seen[v] {
				return
			}
			seen[v] = true
			original := v.Records
			v.Records = func(yield func(*subnets.SubnetRecord, error) bool) {
				if err := guard(ctx); err != nil {
					yield(nil, err)
					return
				}
				for row, err := range original {
					if err == nil {
						err = subnetReadLocation(row, location)
					}
					if err = errors.Join(err, guard(ctx)); err != nil {
						if row != nil {
							err = &resource.ResponseError{StatusCode: row.StatusCode, Header: row.Header.Clone(), Body: bytes.Clone(row.Envelope), Cause: err}
						}
						yield(nil, err)
						return
					}
					if !yield(row, nil) {
						return
					}
				}
			}
		case []any:
			for _, item := range v {
				bind(item)
			}
		case map[string]any:
			for _, item := range v {
				bind(item)
			}
		}
	}
	for _, value := range values {
		bind(value)
	}
}
func (s *Service) subnetCloudList(ctx context.Context, op, identity string, search bool, options []SubnetReadOption) (*SubnetRecords, error) {
	wrap := func(err error) error { return request.Wrap(op, "subnets", cloudread.ContextError(ctx, err)) }
	api, cfg, guard, err := s.prepareSubnetRead(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result := &SubnetRecords{Inventory: make([]*subnets.SubnetRecord, 0)}
	fail := func(err error) (*SubnetRecords, error) {
		if proof := subnetReadReceipt(err); proof != nil {
			result.LastResponse = proof
		}
		return result, wrap(err)
	}
	if !search && cfg.NetworkAvailable != nil && !*cfg.NetworkAvailable {
		result.Subnets = []*subnets.SubnetRecord{}
		result.Value = []any{}
		result.JSON = json.RawMessage("[]")
		return result, nil
	}
	if api == nil || api.Subnets == nil {
		return fail(qosReadInvalid("configured subnet network service required"))
	}
	members, err := qosPolicyCloudPairs(cfg.Filters, search)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(err)
	}
	ordered := make([]cloudfilter.JSONMember, 0, len(members)+1)
	if search && identity != "" {
		raw, _ := json.Marshal(identity)
		ordered = append(ordered, cloudfilter.JSONMember{Key: "name", Value: raw})
	}
	positions := make(map[string]int)
	for i, member := range ordered {
		positions[member.Key] = i
	}
	for _, member := range members {
		if i, present := positions[member.Key]; present {
			ordered[i].Value = bytes.Clone(member.Value)
		} else {
			positions[member.Key] = len(ordered)
			ordered = append(ordered, member)
		}
	}
	leaf := make([]subnets.RecordListOption, 0, len(ordered)+3)
	for _, member := range ordered {
		leaf = append(leaf, subnets.WithRecordListQuery(member.Key, member.Value))
	}
	if cfg.headersPresent {
		leaf = append(leaf, subnets.WithRecordListHeaders(cfg.Headers))
	}
	if cfg.Microversion != nil {
		leaf = append(leaf, subnets.WithRecordListMicroversion(*cfg.Microversion))
	}
	if cfg.BasePath != nil {
		leaf = append(leaf, subnets.WithRecordListBasePath(*cfg.BasePath))
	}
	leafCtx := rest.WithOperationGuard(ctx, guard)
	proxy, err := api.Subnets.ProxyListRecords(leafCtx, leaf...)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(err)
	}
	values := make([]any, 0)
	records := make([]*subnets.SubnetRecord, 0)
	switch value := proxy.Value.(type) {
	case *subnets.RecordListGenerator:
		for row, err := range value.Records {
			if row != nil {
				result.LastResponse = subnetReadCopy(row)
			}
			if err = errors.Join(err, guard(ctx)); err != nil {
				return fail(err)
			}
			if err := subnetReadLocation(row, cfg.CurrentLocation); err != nil {
				return fail(&resource.ResponseError{StatusCode: row.StatusCode, Header: row.Header.Clone(), Body: bytes.Clone(row.Envelope), Cause: err})
			}
			result.Inventory = append(result.Inventory, subnetReadCopy(row))
			result.LastResponse = subnetReadCopy(row)
			records = append(records, subnetReadCopy(row))
			values = append(values, row.Resource.Clone())
		}
	case []any:
		values = append(values, value...)
	case string:
		for _, char := range value {
			values = append(values, string(char))
		}
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			values = append(values, key)
		}
	default:
		return fail(qosReadInvalid("Proxy result %T is not iterable by cloud list()", proxy.Value))
	}
	subnetReadBindGenerators(values, cfg.CurrentLocation, guard, ctx)
	if err := guard(ctx); err != nil {
		return fail(err)
	}
	var encoded json.RawMessage
	if !subnetReadOpaque(values) {
		encoded, err = json.Marshal(values)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return fail(err)
		}
	}
	result.Subnets, result.Value, result.JSON = records, values, bytes.Clone(encoded)
	return result, nil
}
