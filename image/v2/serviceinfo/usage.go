package serviceinfo

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// GetUsageInfo reads the authenticated project's quota usage once. The server
// supplies resource names and values; this call does not enforce quota policy.
func (a *API) GetUsageInfo(ctx context.Context, options ...GetUsageInfoOption) (*UsageInfo, error) {
	prepared, err := a.capture(ctx)
	if err != nil {
		return nil, wrapInfoError(ctx, "GetUsageInfo", err)
	}
	headers, err := prepareGetUsageInfo(append([]GetUsageInfoOption(nil), options...))
	if err == nil {
		err = prepared.check(ctx)
	}
	if err != nil {
		return nil, wrapInfoError(ctx, "GetUsageInfo", err)
	}
	prepared.addHeaders(headers)
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, prepared.client.ServiceURL("info", "usage"), nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapInfoError(ctx, "GetUsageInfo", err)
	}
	if err := validateResponse(response); err != nil {
		return nil, wrapInfoError(ctx, "GetUsageInfo", response.Fail(err))
	}
	value, err := rest.Decode(response, "", func(value *UsageInfo) *resource.Metadata { return &value.Metadata })
	return value, wrapInfoError(ctx, "GetUsageInfo", err)
}
