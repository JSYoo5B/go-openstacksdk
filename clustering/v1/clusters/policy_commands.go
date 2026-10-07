package clusters

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/actions"
	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// AttachPolicy submits a binding change. PolicyID is a controller identity in
// the body; the SDK does not fetch the policy or wait for the accepted action.
func (a *API) AttachPolicy(ctx context.Context, ref resource.Ref, opts AttachPolicyOpts, options ...AttachPolicyOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "AttachPolicy", "policy_attach", opts, func(value AttachPolicyOpts) (int, error) {
		return 0, senlin.Required(value.PolicyID)
	}, options...)
}

// DetachPolicy sends policy_id alone by default. WithDetachPolicyField can
// supply deployment extensions beyond Python's policy_id-only request.
func (a *API) DetachPolicy(ctx context.Context, ref resource.Ref, opts DetachPolicyOpts, options ...DetachPolicyOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "DetachPolicy", "policy_detach", opts, func(value DetachPolicyOpts) (int, error) {
		return 0, senlin.Required(value.PolicyID)
	}, options...)
}

// UpdatePolicy changes the binding, rather than the policy resource itself.
// It retains the policy_update command used by the pinned Python SDK.
func (a *API) UpdatePolicy(ctx context.Context, ref resource.Ref, opts UpdatePolicyOpts, options ...UpdatePolicyOption) (*actions.Submission, error) {
	return clusterCommand(ctx, a, ref, "UpdatePolicy", "policy_update", opts, func(value UpdatePolicyOpts) (int, error) {
		return 0, senlin.Required(value.PolicyID)
	}, options...)
}
