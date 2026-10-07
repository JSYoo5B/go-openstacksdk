package clusterpolicies

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// WaitForStatus has no default status field and fails before HTTP. Callers may
// select another exported string attribute through ordinary wait options.
func (s *Scope) WaitForStatus(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*ClusterPolicy, error) {
	return senlin.WaitForStatus(ctx, s.collection(), ref, status, options...)
}

// WaitForDelete polls the fixed scope for a missing policy binding. It does not
// detach a policy or delete a resource, and defaults to a 120-second timeout.
func (s *Scope) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return senlin.WaitForDelete(ctx, s.collection(), ref, false, options...)
}
