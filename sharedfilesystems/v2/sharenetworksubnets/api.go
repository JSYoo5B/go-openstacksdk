// Package sharenetworksubnets owns Manila's share network subnet API
// (microversion 2.51) and subnet metadata (2.78), absent from Gophercloud
// v2.15.0. Subnets always belong to one share network: bind it once with
// InNetwork, which performs no request.
package sharenetworksubnets

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const (
	// MinMicroversion is the first version with subnet routes.
	MinMicroversion = 51
	// MinMetadataMicroversion is the first version with subnet metadata routes.
	MinMetadataMicroversion = 78
)

type Evidence = manilaown.Evidence

// Subnet is the typed view of one share network subnet. MTU stays raw JSON
// because the service may return a number or a string.
type Subnet struct {
	ID               string            `json:"id"`
	AvailabilityZone string            `json:"availability_zone"`
	CIDR             string            `json:"cidr"`
	CreatedAt        string            `json:"created_at"`
	Gateway          string            `json:"gateway"`
	IPVersion        *int64            `json:"ip_version"`
	MTU              json.RawMessage   `json:"mtu"`
	NetworkType      string            `json:"network_type"`
	NeutronNetID     string            `json:"neutron_net_id"`
	NeutronSubnetID  string            `json:"neutron_subnet_id"`
	SegmentationID   *int64            `json:"segmentation_id"`
	ShareNetworkID   string            `json:"share_network_id"`
	ShareNetworkName string            `json:"share_network_name"`
	UpdatedAt        string            `json:"updated_at"`
	Metadata         map[string]string `json:"metadata"`
	Evidence         Evidence          `json:"-"`
}

// API creates network-bound scopes.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Scope is one share network's subnet set. Resources shares the Collection
// policies (ID reads, lists, deletion waits); a subnet has no name or status.
type Scope struct {
	api       *API
	networkID string
	Resources *resource.Collection[Subnet]
}

func wrap(op string, err error) error { return request.Wrap(op, "sharenetworksubnets", err) }

// InNetwork fixes the parent share network ID (one unescaped path segment).
// It never looks the network up.
func (a *API) InNetwork(ctx context.Context, networkID string) (*Scope, error) {
	if a == nil || a.client == nil {
		return nil, wrap("InNetwork", fmt.Errorf("%w: share network subnet API is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap("InNetwork", err)
	}
	if err := resource.ID(networkID).Validate(); err != nil {
		return nil, wrap("InNetwork", err)
	}
	s := &Scope{api: a, networkID: networkID}
	s.Resources = resource.NewCollection(resource.Adapter[Subnet]{
		Kind:         "sharenetworksubnets",
		IdentityFind: true,
		Get:          s.Get,
		ID:           func(v *Subnet) string { return v.ID },
		Name:         func(*Subnet) string { return "" },
		Delete:       s.Delete,
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*Subnet, error] {
			call := manilaown.Call{Client: a.client, Floor: MinMicroversion, Method: http.MethodGet, URL: manilaown.WithQuery(s.base(), q), OK: []int{http.StatusOK}}
			return manilaown.Iterate[Subnet](ctx, call, "share_network_subnets", control)
		},
	})
	return s, nil
}

func (s *Scope) NetworkID() string {
	if s == nil {
		return ""
	}
	return s.networkID
}

func (s *Scope) base() string {
	return s.api.client.ServiceURL("share-networks", s.networkID, "subnets")
}

func (s *Scope) check(ctx context.Context) error {
	if s == nil || s.api == nil || s.api.client == nil {
		return fmt.Errorf("%w: share network subnet scope is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

func (s *Scope) call(method, target string, floor int, body any, ok ...int) manilaown.Call {
	return manilaown.Call{Client: s.api.client, Floor: floor, Method: method, URL: target, Body: body, OK: ok}
}

// ListOpts has no server filter in the pinned source; extension queries go
// through WithListQuery.
type ListOpts struct{}

type ListOption = request.Option[ListOpts]

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func WithListMaxItems(value int) ListOption {
	return manilaown.WithListControl[ListOpts](func(c *resource.ListControl) { c.MaxItems = value })
}

// List yields the subnets of one GET .../subnets response.
func (s *Scope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Subnet, error] {
	if err := s.check(ctx); err != nil {
		return func(yield func(*Subnet, error) bool) { yield(nil, wrap("List", err)) }
	}
	route := func(request.Config[ListOpts]) string { return s.base() }
	return manilaown.ListFor[Subnet](ctx, s.api.client, MinMicroversion, "share_network_subnets", route, resource.ListControl{}, options)
}

func (s *Scope) Get(ctx context.Context, id string) (*Subnet, error) {
	if err := s.check(ctx); err != nil {
		return nil, wrap("Get", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("Get", err)
	}
	resp, err := s.call(http.MethodGet, s.base()+"/"+id, MinMicroversion, nil, http.StatusOK).Do(ctx)
	if err != nil {
		return nil, wrap("Get", err)
	}
	value, err := manilaown.One[Subnet](resp, "share_network_subnet")
	return value, wrap("Get", err)
}

// CreateOpts is the body of POST .../subnets, sent under the envelope key
// "share-network-subnet". Metadata needs microversion 2.78 on the service.
type CreateOpts struct {
	NeutronNetID     request.Optional[string] `json:"neutron_net_id,omitzero"`
	NeutronSubnetID  request.Optional[string] `json:"neutron_subnet_id,omitzero"`
	AvailabilityZone request.Optional[string] `json:"availability_zone,omitzero"`
	Metadata         map[string]string        `json:"metadata,omitzero"`
}

type CreateOption = request.Option[CreateOpts]

func WithCreateOptions(value CreateOpts) CreateOption {
	if value.Metadata != nil {
		copy := make(map[string]string, len(value.Metadata))
		for k, v := range value.Metadata {
			copy[k] = v
		}
		value.Metadata = copy
	}
	return request.WithOptions(value)
}

func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

type protectedCreate struct {
	CreateOpts
	ID             string `json:"id"`
	ShareNetworkID string `json:"share_network_id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	Envelope       any    `json:"share-network-subnet"`
}

// Create sends one POST (202 or 200). The subnet may still be provisioning
// when acknowledged; a failure is never replayed.
func (s *Scope) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Subnet, error) {
	if err := s.check(ctx); err != nil {
		return nil, wrap("Create", err)
	}
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = manilaown.Envelope("share-network-subnet", cfg.Options, cfg.Fields, protectedCreate{})
	}
	if err != nil {
		return nil, wrap("Create", err)
	}
	resp, err := s.call(http.MethodPost, s.base(), MinMicroversion, body, 200, 202).Do(ctx)
	if err != nil {
		return nil, wrap("Create", err)
	}
	value, err := manilaown.One[Subnet](resp, "share_network_subnet")
	return value, wrap("Create", err)
}

// Delete sends one DELETE (200, 202 or 204); a 404 stays native.
func (s *Scope) Delete(ctx context.Context, id string) error {
	if err := s.check(ctx); err != nil {
		return wrap("Delete", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap("Delete", err)
	}
	_, err := s.call(http.MethodDelete, s.base()+"/"+id, MinMicroversion, nil, 200, 202, 204).Do(ctx)
	return wrap("Delete", err)
}

// Remove deletes by ID reference and ignores a missing subnet by default.
func (s *Scope) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	if err := s.check(ctx); err != nil {
		return wrap("Remove", err)
	}
	return s.Resources.Delete(ctx, ref, options...)
}

func (s *Scope) WaitForDeletion(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	if err := s.check(ctx); err != nil {
		return wrap("WaitForDeletion", err)
	}
	return s.Resources.WaitDeleted(ctx, ref, options...)
}

func (s *Scope) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Subnet, error) {
	if err := s.check(ctx); err != nil {
		return nil, wrap("Find", err)
	}
	return s.Resources.Find(ctx, ref, options...)
}

func (s *Scope) metadataURL(id string, key ...string) string {
	target := s.base() + "/" + id + "/metadata"
	for _, k := range key {
		target += "/" + url.PathEscape(k)
	}
	return target
}

// GetMetadata reads GET .../metadata (200). An absent or null metadata member
// yields an empty non-nil map.
func (s *Scope) GetMetadata(ctx context.Context, id string) (map[string]string, error) {
	if err := s.check(ctx); err != nil {
		return nil, wrap("GetMetadata", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("GetMetadata", err)
	}
	resp, err := s.call(http.MethodGet, s.metadataURL(id), MinMetadataMicroversion, nil, 200).Do(ctx)
	if err != nil {
		return nil, wrap("GetMetadata", err)
	}
	out, err := manilaown.MetadataMap(resp, "metadata")
	return out, wrap("GetMetadata", err)
}

// GetMetadatum reads GET .../metadata/{key} (200) and returns the "meta" map.
func (s *Scope) GetMetadatum(ctx context.Context, id, key string) (map[string]string, error) {
	if err := s.check(ctx); err != nil {
		return nil, wrap("GetMetadatum", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap("GetMetadatum", err)
	}
	if key == "" {
		return nil, wrap("GetMetadatum", fmt.Errorf("%w: metadata key is required", resource.ErrInvalidOption))
	}
	resp, err := s.call(http.MethodGet, s.metadataURL(id, key), MinMetadataMicroversion, nil, 200).Do(ctx)
	if err != nil {
		return nil, wrap("GetMetadatum", err)
	}
	out, err := manilaown.MetadataMap(resp, "meta")
	return out, wrap("GetMetadatum", err)
}

func (s *Scope) writeMetadata(ctx context.Context, op, method, id string, metadata map[string]string) (map[string]string, error) {
	if err := s.check(ctx); err != nil {
		return nil, wrap(op, err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, wrap(op, err)
	}
	copy := make(map[string]string, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	resp, err := s.call(method, s.metadataURL(id), MinMetadataMicroversion, map[string]any{"metadata": copy}, 200).Do(ctx)
	if err != nil {
		return nil, wrap(op, err)
	}
	out, err := manilaown.MetadataMap(resp, "metadata")
	return out, wrap(op, err)
}

// SetMetadata merges keys with POST .../metadata. A nil map sends {}. The
// returned map is the service's resulting metadata (Python echoes the input).
func (s *Scope) SetMetadata(ctx context.Context, id string, metadata map[string]string) (map[string]string, error) {
	return s.writeMetadata(ctx, "SetMetadata", http.MethodPost, id, metadata)
}

// ReplaceMetadata replaces all metadata with PUT. A nil or empty map clears it.
func (s *Scope) ReplaceMetadata(ctx context.Context, id string, metadata map[string]string) (map[string]string, error) {
	return s.writeMetadata(ctx, "ReplaceMetadata", http.MethodPut, id, metadata)
}

// DeleteMetadata sends one DELETE per distinct key in first-seen order. The
// first failure (a 404 included) stops the loop; earlier deletions are not
// rolled back. A nil or empty list sends nothing: Python's keys=None clearing
// is ReplaceMetadata with an empty map here. Success is 200.
func (s *Scope) DeleteMetadata(ctx context.Context, id string, keys []string) error {
	if err := s.check(ctx); err != nil {
		return wrap("DeleteMetadata", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return wrap("DeleteMetadata", err)
	}
	seen := make(map[string]bool, len(keys))
	var ordered []string
	for _, key := range keys {
		if key == "" {
			return wrap("DeleteMetadata", fmt.Errorf("%w: metadata key is required", resource.ErrInvalidOption))
		}
		if !seen[key] {
			seen[key] = true
			ordered = append(ordered, key)
		}
	}
	for _, key := range ordered {
		if _, err := s.call(http.MethodDelete, s.metadataURL(id, key), MinMetadataMicroversion, nil, 200).Do(ctx); err != nil {
			return wrap("DeleteMetadata", err)
		}
	}
	return nil
}
