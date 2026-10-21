package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The methods below are the Go counterparts of the openstacksdk cloud helpers
// delete_coe_cluster, update_coe_cluster, delete_cluster_template and
// update_cluster_template. They resolve a single string the way the helpers do
// and then call the ordinary resource operations. Mutations are issued once and
// never replayed; an acknowledgement is not completion.

func invalidCOEReference(kind string) error {
	return fmt.Errorf("%w: %s reference must not be empty", resource.ErrInvalidOption, kind)
}

// selectCOECluster mirrors the cloud helper's search over the complete cluster
// list: a value matches a cluster when it equals the UUID or the name, or when
// it matches either as a Python fnmatch-style glob. More than one match is an
// ambiguity; no match returns nil.
func (s *Service) selectCOECluster(ctx context.Context, nameOrID string) (*clusters.Cluster, error) {
	if nameOrID == "" {
		return nil, invalidCOEReference("cluster")
	}
	var rows []*clusters.Cluster
	var raw []json.RawMessage
	for value, err := range s.Clusters.List(ctx) {
		if err != nil {
			return nil, err
		}
		row, err := json.Marshal(map[string]string{"id": value.UUID, "name": value.Name})
		if err != nil {
			return nil, err
		}
		rows = append(rows, value)
		raw = append(raw, row)
	}
	selected, err := cloudfilter.Select(raw, nameOrID, nil, nil)
	if err != nil {
		return nil, err
	}
	switch len(selected.Indices) {
	case 0:
		return nil, nil
	case 1:
		return rows[selected.Indices[0]], nil
	}
	ids := make([]string, 0, len(selected.Indices))
	for _, index := range selected.Indices {
		ids = append(ids, rows[index].UUID)
	}
	return nil, &resource.AmbiguousError{Resource: "clusters", Name: nameOrID, IDs: ids}
}

// DeleteCOECluster resolves nameOrID against the cluster list and sends one
// DELETE for the selected UUID. It reports false without a DELETE when nothing
// matches and true after an accepted DELETE; a 404 on the DELETE itself is also
// reported as true because the cloud helper ignores a missing cluster at that
// step. Multiple matches fail with resource.ErrAmbiguous. Acceptance does not
// mean the cluster is gone; use Clusters.WaitForDeletion.
func (s *Service) DeleteCOECluster(ctx context.Context, nameOrID string) (bool, error) {
	cluster, err := s.selectCOECluster(ctx, nameOrID)
	if err != nil || cluster == nil {
		return false, err
	}
	if err := s.Clusters.Delete(ctx, cluster.UUID); err != nil && !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return false, err
	}
	return true, nil
}

// UpdateCOECluster resolves nameOrID like DeleteCOECluster and sends one PATCH
// with the explicit JSON patch operations for the selected UUID. A missing
// cluster is a resource.NotFoundError. It returns the UUID acknowledged by the
// server; completion is observed separately.
func (s *Service) UpdateCOECluster(ctx context.Context, nameOrID string, ops []clusters.UpdateOpts) (string, error) {
	cluster, err := s.selectCOECluster(ctx, nameOrID)
	if err != nil {
		return "", err
	}
	if cluster == nil {
		return "", &resource.NotFoundError{Resource: "clusters", Reference: nameOrID}
	}
	return s.Clusters.Update(ctx, cluster.UUID, ops)
}

// findClusterTemplate mirrors the proxy find: a direct GET of the string as an
// ID is tried first and only a 400, 403 or 404 response falls through to an
// exact id-or-name match over the template list. Other errors are returned.
func (s *Service) findClusterTemplate(ctx context.Context, nameOrID string) (*clustertemplates.ClusterTemplate, error) {
	if nameOrID == "" {
		return nil, invalidCOEReference("cluster template")
	}
	value, err := s.ClusterTemplates.Get(ctx, url.PathEscape(nameOrID))
	if err == nil {
		return value, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, errors.Join(err, ctxErr)
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) && !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) && !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
		return nil, err
	}
	var match *clustertemplates.ClusterTemplate
	var ids []string
	for candidate, err := range s.ClusterTemplates.List(ctx) {
		if err != nil {
			return nil, err
		}
		if candidate.UUID == nameOrID || candidate.Name == nameOrID {
			ids = append(ids, candidate.UUID)
			if match == nil {
				match = candidate
			}
		}
	}
	if len(ids) > 1 {
		return nil, &resource.AmbiguousError{Resource: "clustertemplates", Name: nameOrID, IDs: ids}
	}
	return match, nil
}

// DeleteClusterTemplate resolves nameOrID (ID first, then exact name) and sends
// one DELETE. It returns false without a DELETE when nothing matches, true after
// an accepted DELETE, and also true when that DELETE reports 404. Duplicate
// names fail with resource.ErrAmbiguous.
func (s *Service) DeleteClusterTemplate(ctx context.Context, nameOrID string) (bool, error) {
	template, err := s.findClusterTemplate(ctx, nameOrID)
	if err != nil || template == nil {
		return false, err
	}
	if err := s.ClusterTemplates.Delete(ctx, template.UUID); err != nil && !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return false, err
	}
	return true, nil
}

// UpdateClusterTemplate resolves nameOrID like DeleteClusterTemplate and sends
// one PATCH with the explicit JSON patch operations. A missing template is a
// resource.NotFoundError.
func (s *Service) UpdateClusterTemplate(ctx context.Context, nameOrID string, ops []clustertemplates.UpdateOpts) (*clustertemplates.ClusterTemplate, error) {
	template, err := s.findClusterTemplate(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, &resource.NotFoundError{Resource: "clustertemplates", Reference: nameOrID}
	}
	return s.ClusterTemplates.Update(ctx, template.UUID, ops)
}
