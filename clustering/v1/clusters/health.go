package clusters

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/actions"
	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Check submits a health check with an empty parameter object by default.
// Profile-specific inputs remain inside that object and are server validated.
func (a *API) Check(ctx context.Context, ref resource.Ref, options ...CheckOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "Check", "check", CheckOpts{}, nil, options...)
}

// Recover submits a recovery command. Explicit Check and CheckCapacity values,
// including false or null, require versions 1.6 and 1.7 respectively.
func (a *API) Recover(ctx context.Context, ref resource.Ref, value RecoverOpts, options ...RecoverOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "Recover", "recover", value, func(value RecoverOpts) (int, error) {
		minimum := 0
		if value.Check.IsSet() {
			minimum = 6
		}
		if value.CheckCapacity.IsSet() {
			minimum = 7
		}
		return minimum, senlin.OptionalObject(value.OperationParams, "operation_params")
	}, options...)
}

// PerformOperation submits a profile operation at version 1.4 or newer. The
// controller selects nodes using Filters and supplies Params to their profile.
func (a *API) PerformOperation(ctx context.Context, ref resource.Ref, operation string, value PerformOperationOpts, options ...PerformOperationOption) (*actions.Submission, error) {
	if err := senlin.RequireVersion(ctx, a.RawClient(), 4); err != nil {
		return nil, request.Wrap("PerformOperation", "clustering.clusters", err)
	}
	if err := senlin.Required(operation); err != nil {
		return nil, request.Wrap("PerformOperation", "clustering.clusters", err)
	}
	return clusterCommandPath(ctx, a, ref, "PerformOperation", "ops", operation, value, func(value PerformOperationOpts) (int, error) {
		if err := senlin.OptionalObject(value.Filters, "filters"); err != nil {
			return 4, err
		}
		return 4, senlin.OptionalObject(value.Params, "params")
	}, options...)
}
