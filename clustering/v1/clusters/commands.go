package clusters

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Cluster commands return the accepted action and its HTTP evidence. They do
// not fetch or poll the action, and do not describe a completed cluster change.
func clusterCommand[T any](ctx context.Context, a *API, ref resource.Ref, operation, key string, value T, validate func(T) (int, error), options ...request.Option[T]) (*actions.Submission, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	config, err := request.Apply(value, options...)
	minimum := 0
	if err == nil && validate != nil {
		minimum, err = validate(config.Options)
	}
	if err == nil {
		if minimum > 0 {
			err = senlin.RequireVersion(ctx, client, minimum)
		} else {
			err = senlin.Validate(ctx, client)
		}
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, key)
	}
	if err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	headers := maps.Clone(config.Headers)
	identity, err := rest.Collection(spec(client)).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	if minimum > 0 {
		err = senlin.RequireVersion(ctx, client, minimum)
	} else {
		err = senlin.Validate(ctx, client)
	}
	if err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPost, client.ServiceURL("clusters", url.PathEscape(identity), "actions"), body, headers, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	actionID, err := senlin.CommandActionID(client, response)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.clusters", err)
	}
	return &actions.Submission{ActionID: actionID, Location: senlin.LocationValues(response.Header)[0], Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

func commandNodes(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%w: cluster command requires at least one node", resource.ErrInvalidOption)
	}
	for index, value := range values {
		if err := senlin.Identifier(value); err != nil {
			return fmt.Errorf("node %d: %w", index, err)
		}
	}
	return nil
}

func (a *API) ScaleIn(ctx context.Context, ref resource.Ref, opts ScaleInOpts, options ...ScaleInOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "ScaleIn", "scale_in", opts, nil, options...)
}
func (a *API) ScaleOut(ctx context.Context, ref resource.Ref, opts ScaleOutOpts, options ...ScaleOutOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "ScaleOut", "scale_out", opts, nil, options...)
}
func (a *API) Resize(ctx context.Context, ref resource.Ref, opts ResizeOpts, options ...ResizeOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "Resize", "resize", opts, nil, options...)
}
func (a *API) AddNodes(ctx context.Context, ref resource.Ref, opts AddNodesOpts, options ...AddNodesOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "AddNodes", "add_nodes", opts, func(value AddNodesOpts) (int, error) {
		return 0, commandNodes(value.Nodes)
	}, options...)
}
func (a *API) RemoveNodes(ctx context.Context, ref resource.Ref, opts RemoveNodesOpts, options ...RemoveNodesOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "RemoveNodes", "del_nodes", opts, func(value RemoveNodesOpts) (int, error) {
		minimum := 0
		if value.DestroyAfterDeletion.IsSet() {
			minimum = 4
		}
		return minimum, commandNodes(value.Nodes)
	}, options...)
}
func (a *API) ReplaceNodes(ctx context.Context, ref resource.Ref, opts ReplaceNodesOpts, options ...ReplaceNodesOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "ReplaceNodes", "replace_nodes", opts, func(value ReplaceNodesOpts) (int, error) {
		if len(value.Nodes) == 0 {
			return 3, fmt.Errorf("%w: cluster replacement requires at least one node pair", resource.ErrInvalidOption)
		}
		for previous, replacement := range value.Nodes {
			if err := senlin.Identifier(previous); err != nil {
				return 3, err
			}
			if err := senlin.Identifier(replacement); err != nil {
				return 3, err
			}
		}
		return 3, nil
	}, options...)
}
