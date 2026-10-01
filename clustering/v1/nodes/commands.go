package nodes

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Check submits a health check. Parameters remain inside the check object;
// plugin-specific names cannot replace the selected node or command envelope.
func (a *API) Check(ctx context.Context, ref resource.Ref, options ...CheckOption) (*actions.Submission, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Check", "clustering.nodes", err)
	}
	config, err := request.Apply(CheckOpts{}, options...)
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, "check")
	}
	if err != nil {
		return nil, request.Wrap("Check", "clustering.nodes", err)
	}
	return submitNodeCommand(ctx, client, ref, "Check", "actions", body, maps.Clone(config.Headers), 0)
}

// Recover submits a recovery command. An explicit Check value, including false
// or null, requires version 1.6; an omitted value leaves the server default.
func (a *API) Recover(ctx context.Context, ref resource.Ref, value RecoverOpts, options ...RecoverOption) (*actions.Submission, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Recover", "clustering.nodes", err)
	}
	config, err := request.Apply(value, options...)
	if err == nil {
		err = senlin.OptionalObject(config.Options.OperationParams, "operation_params")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, "recover")
	}
	if err != nil {
		return nil, request.Wrap("Recover", "clustering.nodes", err)
	}
	minimum := 0
	if config.Options.Check.IsSet() {
		minimum = 6
	}
	return submitNodeCommand(ctx, client, ref, "Recover", "actions", body, maps.Clone(config.Headers), minimum)
}

// PerformOperation submits a profile-specific operation at version 1.4 or
// newer. The service defines the operation name and parameter schema.
func (a *API) PerformOperation(ctx context.Context, ref resource.Ref, operation string, options ...PerformOperationOption) (*actions.Submission, error) {
	client := a.RawClient()
	if err := senlin.RequireVersion(ctx, client, 4); err != nil {
		return nil, request.Wrap("PerformOperation", "clustering.nodes", err)
	}
	if err := senlin.Required(operation); err != nil {
		return nil, request.Wrap("PerformOperation", "clustering.nodes", err)
	}
	config, err := request.Apply(PerformOperationOpts{}, options...)
	var body json.RawMessage
	if err == nil {
		body, err = senlin.CommandBody(config, operation)
	}
	if err != nil {
		return nil, request.Wrap("PerformOperation", "clustering.nodes", err)
	}
	return submitNodeCommand(ctx, client, ref, "PerformOperation", "ops", body, maps.Clone(config.Headers), 4)
}

func validateNodeCommand(ctx context.Context, client *gophercloud.ServiceClient, minimum int) error {
	if minimum == 0 {
		return senlin.Validate(ctx, client)
	}
	return senlin.RequireVersion(ctx, client, minimum)
}

func submitNodeCommand(ctx context.Context, client *gophercloud.ServiceClient, ref resource.Ref, operation, path string, body json.RawMessage, headers map[string]string, minimum int) (*actions.Submission, error) {
	if err := validateNodeCommand(ctx, client, minimum); err != nil {
		return nil, request.Wrap(operation, "clustering.nodes", err)
	}
	identity, err := rest.Collection(spec(client)).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.nodes", err)
	}
	if err := validateNodeCommand(ctx, client, minimum); err != nil {
		return nil, request.Wrap(operation, "clustering.nodes", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodPost, client.ServiceURL("nodes", url.PathEscape(identity), path), body, headers, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.nodes", err)
	}
	actionID, err := senlin.CommandActionID(client, response)
	if err != nil {
		return nil, request.Wrap(operation, "clustering.nodes", err)
	}
	return &actions.Submission{
		ActionID: actionID, Location: senlin.LocationValues(response.Header)[0],
		Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}, nil
}
