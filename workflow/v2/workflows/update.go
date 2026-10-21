package workflows

import (
	"context"
	"fmt"
	"io"
	"net/http"

	request "github.com/JSYoo5B/go-openstacksdk/request"
	resource "github.com/JSYoo5B/go-openstacksdk/resource"
	gophercloud "github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/workflow/v2/workflows"
)

// UpdateOpts selects the PUT /workflows request. Definition is the complete
// Mistral v2 workflow definition sent as text/plain; Scope and Namespace are
// optional query fields that are omitted when empty. Mistral identifies the
// workflows to replace from the names inside the definition, not from a path
// segment, so Update takes no workflow reference.
type UpdateOpts struct {
	Scope      string `q:"scope"`
	Namespace  string `q:"namespace"`
	Definition io.Reader
}

// UpdateOption configures Update with the SDK's concrete option mechanism.
type UpdateOption = request.Option[UpdateOpts]

func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.WithOptions(value) }
func WithUpdateQuery(key, value string) UpdateOption {
	return request.WithQuery[UpdateOpts](key, value)
}

// Update replaces the workflows named in the definition. It sends exactly one
// PUT with Content-Type text/plain and no JSON Accept header, requires 200 and
// decodes the returned workflow list. A nil Definition fails before HTTP. The
// call is never replayed automatically: after an uncertain failure the
// definition may or may not have been applied, and the one-shot Definition
// reader has been consumed. Local cancellation does not roll back remote work.
func (a *API) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) ([]Workflow, error) {
	cfg, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err == nil && cfg.Options.Definition == nil {
		err = fmt.Errorf("%w: workflow definition is required", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, request.Wrap("Update", "workflows", err)
	}
	q, err := gophercloud.BuildQueryString(cfg.Options)
	if err != nil {
		return nil, request.Wrap("Update", "workflows", err)
	}
	query, err := request.ExtendQuery(q.String(), cfg.Query)
	if err != nil {
		return nil, request.Wrap("Update", "workflows", err)
	}
	var result upstream.CreateResult
	resp, err := a.client.Put(ctx, a.client.ServiceURL("workflows")+query, cfg.Options.Definition, &result.Body, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK},
		MoreHeaders: map[string]string{
			"Content-Type": "text/plain",
			"Accept":       "",
		},
	})
	_, result.Header, result.Err = gophercloud.ParseResponse(resp, err)
	values, err := result.Extract()
	return values, request.Wrap("Update", "workflows", err)
}
