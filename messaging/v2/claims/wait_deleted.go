package claims

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForDeletion polls GET until the claim is missing, like openstacksdk's
// wait_for_delete on a Claim. A native 404 completes the wait; other HTTP and
// transport errors, cancellation and timeouts fail it. The default policy is
// the shared resource wait policy; pass resource.WithTimeout and
// resource.WithPollInterval to select other values. Waiting does not delete
// the claim or release it remotely.
func (a *API) WaitForDeletion(ctx context.Context, queueName, claimID string, options ...resource.WaitOption) error {
	collection := resource.NewCollection(resource.Adapter[Claim]{
		Kind: "claims",
		Get:  func(ctx context.Context, id string) (*Claim, error) { return a.Get(ctx, queueName, id) },
		ID:   func(*Claim) string { return claimID },
	})
	return collection.WaitDeleted(ctx, resource.ID(claimID), options...)
}
