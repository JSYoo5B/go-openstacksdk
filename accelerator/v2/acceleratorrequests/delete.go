package acceleratorrequests

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/accelerator/v2/common"
	"github.com/JSYoo5B/gophercloudsdk/internal/cyborg"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type DeleteOption = request.Option[struct{}]

func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[struct{}](key, value)
}

// DeleteMany is strict: a 404 for one request must not silently skip deleting
// the others. Single-resource Delete retains the shared ignore-missing default.
func (a *API) DeleteMany(ctx context.Context, ids []string, options ...DeleteOption) (*common.Metadata, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: request IDs must not be empty", resource.ErrInvalidOption)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if err := validateID(id); err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: duplicate request ID %s", resource.ErrInvalidOption, id)
		}
		seen[id] = true
	}
	return a.deleteQuery(ctx, url.Values{"arqs": {strings.Join(ids, ",")}}, options...)
}

// DeleteByInstance uses the server's idempotent instance selector. It is a
// distinct operation so an ARQ ID can never be mistaken for an instance UUID.
func (a *API) DeleteByInstance(ctx context.Context, instanceUUID string, options ...DeleteOption) (*common.Metadata, error) {
	if err := validateID(instanceUUID); err != nil {
		return nil, err
	}
	return a.deleteQuery(ctx, url.Values{"instance": {instanceUUID}}, options...)
}

func (a *API) deleteQuery(ctx context.Context, query url.Values, options ...DeleteOption) (*common.Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cyborg.RequireMicroversion(a.client, 0); err != nil {
		return nil, err
	}
	c, err := request.Apply(struct{}{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(c, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	headers, err := cyborg.Headers(c.Headers)
	if err != nil {
		return nil, err
	}
	meta, err := cyborg.Mutate(ctx, a.client, "DELETE", a.client.ServiceURL("accelerator_requests")+"?"+query.Encode(), nil, headers, 204)
	return meta, request.Wrap("delete", "arqs", err)
}
