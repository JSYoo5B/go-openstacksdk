package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SearchOpts are the local filters of the COE search and get helpers.
type SearchOpts struct {
	Filters *json.RawMessage
}

// SearchOption configures SearchCOEClusters, GetCOECluster,
// SearchClusterTemplates and GetClusterTemplate.
type SearchOption func(*SearchOpts) error

// WithSearchFilters supplies the local filters: a JSON object of exact, nested
// matches or a JSON string holding a JMESPath expression. Filter keys are the
// Python Resource attribute names listed in python-comparison.md.
func WithSearchFilters(filters json.RawMessage) SearchOption {
	owned := bytes.Clone(filters)
	return func(o *SearchOpts) error {
		o.Filters = nil
		if owned != nil {
			v := json.RawMessage(bytes.Clone(owned))
			o.Filters = &v
		}
		return nil
	}
}

func applySearchOptions(options []SearchOption) (SearchOpts, error) {
	var opts SearchOpts
	for _, apply := range options {
		if apply == nil {
			return opts, fmt.Errorf("%w: nil search option", resource.ErrInvalidOption)
		}
		if err := apply(&opts); err != nil {
			return opts, err
		}
	}
	return opts, nil
}

func searchTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

func clusterRow(c *clusters.Cluster) map[string]any {
	return map[string]any{
		"id": c.UUID, "uuid": c.UUID, "name": c.Name, "api_address": c.APIAddress,
		"cluster_template_id": c.ClusterTemplateID, "coe_version": c.COEVersion,
		"create_timeout": c.CreateTimeout, "created_at": searchTime(c.CreatedAt),
		"discovery_url": c.DiscoveryURL, "fixed_network": c.FixedNetwork,
		"fixed_subnet": c.FixedSubnet, "flavor_id": c.FlavorID,
		"is_floating_ip_enabled": c.FloatingIPEnabled, "is_master_lb_enabled": c.MasterLBEnabled,
		"keypair": c.KeyPair, "labels": c.Labels, "master_addresses": c.MasterAddresses,
		"master_count": c.MasterCount, "master_flavor_id": c.MasterFlavorID,
		"node_addresses": c.NodeAddresses, "node_count": c.NodeCount, "stack_id": c.StackID,
		"status": c.Status, "status_reason": c.StatusReason, "updated_at": searchTime(c.UpdatedAt),
		"project_id": c.ProjectID,
	}
}

func templateRow(t *clustertemplates.ClusterTemplate) map[string]any {
	return map[string]any{
		"id": t.UUID, "uuid": t.UUID, "name": t.Name, "apiserver_port": t.APIServerPort,
		"cluster_distro": t.ClusterDistro, "coe": t.COE, "created_at": searchTime(t.CreatedAt),
		"docker_storage_driver": t.DockerStorageDriver, "docker_volume_size": t.DockerVolumeSize,
		"dns_nameserver": t.DNSNameServer, "external_network_id": t.ExternalNetworkID,
		"fixed_network": t.FixedNetwork, "fixed_subnet": t.FixedSubnet, "flavor_id": t.FlavorID,
		"http_proxy": t.HTTPProxy, "https_proxy": t.HTTPSProxy, "image_id": t.ImageID,
		"insecure_registry": t.InsecureRegistry, "is_floating_ip_enabled": t.FloatingIPEnabled,
		"is_hidden": t.Hidden, "is_master_lb_enabled": t.MasterLBEnabled,
		"is_tls_disabled": t.TLSDisabled, "is_public": t.Public, "is_registry_enabled": t.RegistryEnabled,
		"keypair_id": t.KeyPairID, "labels": t.Labels, "master_flavor_id": t.MasterFlavorID,
		"network_driver": t.NetworkDriver, "no_proxy": t.NoProxy, "server_type": t.ServerType,
		"updated_at": searchTime(t.UpdatedAt), "volume_driver": t.VolumeDriver,
		"project_id": t.ProjectID,
	}
}

func selectRows(rows []map[string]any, nameOrID string, opts SearchOpts) (cloudfilter.Result, error) {
	raw := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		b, err := json.Marshal(row)
		if err != nil {
			return cloudfilter.Result{}, err
		}
		raw[i] = b
	}
	selected, err := cloudfilter.Select(raw, nameOrID, opts.Filters, nil)
	if err != nil {
		return selected, errors.Join(fmt.Errorf("%w: local search", resource.ErrInvalidOption), err)
	}
	return selected, nil
}

// ClusterSearch is the result of SearchCOEClusters. When a string filter
// yields an arbitrary JSON value, Expression is true, Value holds it and
// Clusters is empty.
type ClusterSearch struct {
	Value      json.RawMessage
	Expression bool
	Clusters   []*clusters.Cluster
}

// SearchCOEClusters mirrors search_coe_clusters: one complete GET /clusters
// listing (every page), then the identifier selection (exact or fnmatch-style
// glob on UUID or name; an empty nameOrID selects all) and the optional local
// filters. No cluster is returned for a failed listing.
func (s *Service) SearchCOEClusters(ctx context.Context, nameOrID string, options ...SearchOption) (*ClusterSearch, error) {
	opts, err := applySearchOptions(options)
	if err != nil {
		return nil, err
	}
	var listed []*clusters.Cluster
	var rows []map[string]any
	for value, err := range s.Clusters.List(ctx) {
		if err != nil {
			return nil, err
		}
		listed = append(listed, value)
		rows = append(rows, clusterRow(value))
	}
	selected, err := selectRows(rows, nameOrID, opts)
	if err != nil {
		return nil, err
	}
	result := &ClusterSearch{Value: bytes.Clone(selected.Value), Expression: selected.Expression}
	if !selected.Expression {
		for _, i := range selected.Indices {
			result.Clusters = append(result.Clusters, listed[i])
		}
	}
	return result, nil
}

// GetCOECluster mirrors get_coe_cluster: SearchCOEClusters, nil for no match
// and a resource.AmbiguousError for several matches.
func (s *Service) GetCOECluster(ctx context.Context, nameOrID string, options ...SearchOption) (*clusters.Cluster, error) {
	found, err := s.SearchCOEClusters(ctx, nameOrID, options...)
	if err != nil {
		return nil, err
	}
	if found.Expression {
		return nil, fmt.Errorf("%w: a JMESPath filter result is not a cluster", resource.ErrInvalidOption)
	}
	switch len(found.Clusters) {
	case 0:
		return nil, nil
	case 1:
		return found.Clusters[0], nil
	}
	ids := make([]string, len(found.Clusters))
	for i, c := range found.Clusters {
		ids[i] = c.UUID
	}
	return nil, &resource.AmbiguousError{Resource: "clusters", Name: nameOrID, IDs: ids}
}

// ClusterTemplateSearch is the result of SearchClusterTemplates; see ClusterSearch.
type ClusterTemplateSearch struct {
	Value      json.RawMessage
	Expression bool
	Templates  []*clustertemplates.ClusterTemplate
}

// SearchClusterTemplates mirrors search_cluster_templates: the complete
// template listing, glob identifier selection and optional local filters. The
// Python detail argument is ignored there as well.
func (s *Service) SearchClusterTemplates(ctx context.Context, nameOrID string, options ...SearchOption) (*ClusterTemplateSearch, error) {
	opts, err := applySearchOptions(options)
	if err != nil {
		return nil, err
	}
	var listed []*clustertemplates.ClusterTemplate
	var rows []map[string]any
	for value, err := range s.ClusterTemplates.List(ctx) {
		if err != nil {
			return nil, err
		}
		listed = append(listed, value)
		rows = append(rows, templateRow(value))
	}
	selected, err := selectRows(rows, nameOrID, opts)
	if err != nil {
		return nil, err
	}
	result := &ClusterTemplateSearch{Value: bytes.Clone(selected.Value), Expression: selected.Expression}
	if !selected.Expression {
		for _, i := range selected.Indices {
			result.Templates = append(result.Templates, listed[i])
		}
	}
	return result, nil
}

// GetClusterTemplate mirrors get_cluster_template. Without WithSearchFilters it
// is the proxy find (GET by ID, then an exact ID or name match over the list);
// with filters it is SearchClusterTemplates with nil for no match and a
// resource.AmbiguousError for several. An empty reference is invalid.
func (s *Service) GetClusterTemplate(ctx context.Context, nameOrID string, options ...SearchOption) (*clustertemplates.ClusterTemplate, error) {
	opts, err := applySearchOptions(options)
	if err != nil {
		return nil, err
	}
	if opts.Filters == nil {
		return s.findClusterTemplate(ctx, nameOrID)
	}
	found, err := s.SearchClusterTemplates(ctx, nameOrID, options...)
	if err != nil {
		return nil, err
	}
	if found.Expression {
		return nil, fmt.Errorf("%w: a JMESPath filter result is not a cluster template", resource.ErrInvalidOption)
	}
	switch len(found.Templates) {
	case 0:
		return nil, nil
	case 1:
		return found.Templates[0], nil
	}
	ids := make([]string, len(found.Templates))
	for i, t := range found.Templates {
		ids[i] = t.UUID
	}
	return nil, &resource.AmbiguousError{Resource: "clustertemplates", Name: nameOrID, IDs: ids}
}
