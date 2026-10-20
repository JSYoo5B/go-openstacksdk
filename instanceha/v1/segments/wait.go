package segments

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// WaitForStatus requires an explicit typed string attribute because segments
// have no status field. The default reports ErrUnsupported before HTTP.
func (a *API) WaitForStatus(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*Segment, error) {
	return masakari.WaitForStatus(ctx, rest.Collection(a.spec), ref, status, options...)
}

// WaitForDelete polls until HTTP 404, with a default 120-second deadline.
func (a *API) WaitForDelete(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	return masakari.WaitForDelete(ctx, rest.Collection(a.spec), ref, false, options...)
}
