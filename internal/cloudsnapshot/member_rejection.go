package cloudsnapshot

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// A compatible status may enable name lookup only after fault-free native
// rejection handling. The shared observer keeps native HTTP status policy.
func (p *reader) memberGet(ctx context.Context, target string) (*rest.Response, error) {
	return rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.source.Guard, http.MethodGet, target, nil, nil,
		rest.RejectionPolicy{Codes: []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound}}, sourceCodes()...)
}
