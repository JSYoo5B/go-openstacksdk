package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// UpdateResult retains the accepted response without fabricating an Action.
// The published PATCH contract does not declare an action-object response.
type UpdateResult struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

type UpdateOpts struct {
	Status string `json:"status"`
	Force  *bool  `json:"-"`
}
type UpdateOption = request.Option[UpdateOpts]

func WithUpdateOptions(value UpdateOpts) UpdateOption {
	value.Force = senlin.Bool(value.Force)
	return func(config *request.Config[UpdateOpts]) error {
		config.Options = UpdateOpts{Status: value.Status, Force: senlin.Bool(value.Force)}
		return nil
	}
}
func WithUpdateForce(value bool) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		copy := value
		config.Options.Force = &copy
		return nil
	}
}
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}
func WithUpdateQuery(key, value string) UpdateOption {
	return request.WithQuery[UpdateOpts](key, value)
}
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}

// Update accepts the published CANCELLED request at numeric microversion>=1.12.
// Force is a query parameter, distinct from the action body. The result contains
// original accepted evidence; use Get explicitly to read subsequent state.
func (a *API) Update(ctx context.Context, ref resource.Ref, value UpdateOpts, options ...UpdateOption) (*UpdateResult, error) {
	client := a.RawClient()
	if err := senlin.RequireVersion(ctx, client, 12); err != nil {
		return nil, request.Wrap("Update", "clustering.actions", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, true, true)
	}
	if err == nil && config.Options.Status != "CANCELLED" {
		err = fmt.Errorf("%w: Senlin action update status must be CANCELLED", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	if err == nil {
		if config.Options.Force != nil {
			query.Set("force", strconv.FormatBool(*config.Options.Force))
		}
		for key, values := range config.Query {
			if key == "force" || key == "status" {
				err = fmt.Errorf("%w: query %q is a concrete action update option", resource.ErrInvalidOption, key)
				break
			}
			query[key] = append([]string(nil), values...)
		}
	}
	var body json.RawMessage
	if err == nil {
		bodyConfig := config
		bodyConfig.Query = nil
		body, err = senlin.Body(bodyConfig, "action", "force", "id", "name", "target", "target_id", "action", "cause", "owner", "user", "project", "domain", "interval", "start_time", "end_time", "timeout", "status_reason", "inputs", "outputs", "data", "depends_on", "depended_by", "created_at", "updated_at", "cluster_id")
	}
	if err != nil {
		return nil, request.Wrap("Update", "clustering.actions", err)
	}
	extraHeaders := maps.Clone(config.Headers)
	queryString := query.Encode()
	identity, err := rest.Collection(spec(client)).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.actions", err)
	}
	if err := senlin.RequireVersion(ctx, client, 12); err != nil {
		return nil, request.Wrap("Update", "clustering.actions", err)
	}
	target := client.ServiceURL("actions", url.PathEscape(identity))
	if queryString != "" {
		target += "?" + queryString
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPatch, target, body, extraHeaders, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.actions", err)
	}
	return &UpdateResult{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// Cancel is the common status=CANCELLED update with optional force/query/header.
func (a *API) Cancel(ctx context.Context, ref resource.Ref, options ...UpdateOption) (*UpdateResult, error) {
	return a.Update(ctx, ref, UpdateOpts{Status: "CANCELLED"}, options...)
}
