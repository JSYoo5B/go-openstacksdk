package clusters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

var clusterName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func validateName(name string) error {
	if err := senlin.Required(name); err != nil {
		return err
	}
	if len(name) >= 255 || !clusterName.MatchString(name) {
		return fmt.Errorf("%w: cluster name must start with an ASCII letter and contain fewer than 255 ASCII letters, digits, underscores, periods or hyphens", resource.ErrInvalidOption)
	}
	return nil
}

var responseFields = []string{"id", "status", "status_reason", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at", "init_at", "data", "nodes", "node_ids", "policies", "dependents", "profile_name", "cluster", "operation"}

func submission(client *gophercloud.ServiceClient, response *rest.Response) (*actions.Submission, error) {
	id, err := senlin.ActionID(client, response)
	if err != nil {
		return nil, err
	}
	return &actions.Submission{ActionID: id, Location: senlin.LocationValues(response.Header)[0], Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

func decodeMutation(client *gophercloud.ServiceClient, response *rest.Response, actionRequired bool) (*Cluster, error) {
	value, err := rest.Decode(response, "cluster", func(value *Cluster) *resource.Metadata { return &value.Metadata })
	if err != nil {
		return nil, err
	}
	if actionRequired || len(senlin.LocationValues(response.Header)) != 0 {
		value.Operation, err = submission(client, response)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Cluster, error) {
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Create", "clustering.clusters", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = validateName(config.Options.Name)
	}
	if err == nil {
		err = senlin.Required(config.Options.ProfileID)
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Config, "config")
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	if err != nil {
		return nil, request.Wrap("Create", "clustering.clusters", err)
	}
	forbidden := append(append([]string(nil), responseFields...), "profile_only", "is_profile_only")
	body, err := senlin.Body(config, "cluster", forbidden...)
	if err != nil {
		return nil, request.Wrap("Create", "clustering.clusters", err)
	}
	headers := maps.Clone(config.Headers)
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Create", "clustering.clusters", err)
	}
	response, err := rest.DoJSON(ctx, a.RawClient(), http.MethodPost, a.RawClient().ServiceURL("clusters"), body, headers, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Create", "clustering.clusters", err)
	}
	value, err := decodeMutation(a.RawClient(), response, response.StatusCode == http.StatusAccepted)
	return value, request.Wrap("Create", "clustering.clusters", err)
}

// Update resolves explicit names once and returns the accepted cluster and its
// action reference. Sizes are creation fields; resizing uses separate actions.
func (a *API) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Cluster, error) {
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		if name, set := config.Options.Name.Get(); set {
			err = validateName(name)
		}
	}
	if err == nil {
		if profile, set := config.Options.ProfileID.Get(); set {
			err = senlin.Required(profile)
		}
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Config, "config")
	}
	if err == nil {
		err = senlin.OptionalObject(config.Options.Metadata, "metadata")
	}
	minimum := 0
	if config.Options.ProfileOnly != nil || config.Options.profileOnlyNull {
		minimum = 6
	}
	if err == nil {
		err = senlin.RequireVersion(ctx, a.RawClient(), minimum)
	}
	if err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	forbidden := append(append([]string(nil), responseFields...), "min_size", "max_size", "desired_capacity", "is_profile_only")
	body, err := senlin.Body(config, "cluster", forbidden...)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	headers := maps.Clone(config.Headers)
	id, err := rest.Collection(spec(a.RawClient())).ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	if err := senlin.RequireVersion(ctx, a.RawClient(), minimum); err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	response, err := rest.DoJSON(ctx, a.RawClient(), http.MethodPatch, a.RawClient().ServiceURL("clusters", url.PathEscape(id)), body, headers, http.StatusAccepted)
	if err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	value, err := decodeMutation(a.RawClient(), response, true)
	return value, request.Wrap("Update", "clustering.clusters", err)
}

// Delete returns the asynchronous action submission rather than waiting for
// deletion. Normal deletion sends no body; force deletion sends {"force":true}.
// Resources.Delete is unsupported because its error-only contract loses this
// response evidence. No microversion gate is inferred for force deletion.
func (a *API) Delete(ctx context.Context, ref resource.Ref, options ...DeleteOption) (*actions.Submission, error) {
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Delete", "clustering.clusters", err)
	}
	config, err := request.Apply(DeleteOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = senlin.Headers(config.Headers)
	}
	if err != nil {
		return nil, request.Wrap("Delete", "clustering.clusters", err)
	}
	ignoreMissing := !config.Options.Force
	if config.Options.IgnoreMissing != nil {
		ignoreMissing = *config.Options.IgnoreMissing
	}
	var body any
	if config.Options.Force {
		body = json.RawMessage(`{"force":true}`)
	}
	headers := maps.Clone(config.Headers)
	id, err := rest.Collection(spec(a.RawClient())).ResolveID(ctx, ref)
	if err != nil {
		if ignoreMissing && errors.Is(err, resource.ErrNotFound) {
			return nil, nil
		}
		return nil, request.Wrap("Delete", "clustering.clusters", err)
	}
	if err := senlin.Validate(ctx, a.RawClient()); err != nil {
		return nil, request.Wrap("Delete", "clustering.clusters", err)
	}
	response, err := rest.DoJSON(ctx, a.RawClient(), http.MethodDelete, a.RawClient().ServiceURL("clusters", url.PathEscape(id)), body, headers, http.StatusAccepted)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			if ignoreMissing {
				return nil, nil
			}
			err = &resource.NotFoundError{Resource: "clustering.clusters", Reference: ref.String(), Cause: err}
		}
		return nil, request.Wrap("Delete", "clustering.clusters", err)
	}
	value, err := submission(a.RawClient(), response)
	return value, request.Wrap("Delete", "clustering.clusters", err)
}
