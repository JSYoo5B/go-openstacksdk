package hosts

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus polls inside the fixed segment. Hosts require an explicit
// typed string attribute; the default status attribute is unsupported.
func (s *SegmentScope) WaitForStatus(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*Host, error) {
	return masakari.WaitForStatus(ctx, rest.Collection(s.spec), ref, status, options...)
}

// WaitForDelete polls the fixed segment until HTTP 404 without submitting DELETE.
func (s *SegmentScope) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return masakari.WaitForDelete(ctx, rest.Collection(s.spec), ref, false, options...)
}
