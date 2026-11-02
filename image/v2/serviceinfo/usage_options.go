package serviceinfo

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetUsageInfoOpts has no wire parameters. Usage belongs to the authenticated
// project and is read without a body or a project selection query.
type GetUsageInfoOpts struct{}

// GetUsageInfoOption is one option of GetUsageInfo: request.Header or
// request.Custom over *request.Config[GetUsageInfoOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type GetUsageInfoOption = request.GetCall

var getUsageInfoCalls = calls.Spec[request.Config[GetUsageInfoOpts]]{Header: calls.ConfigHeader[GetUsageInfoOpts]}

func prepareGetUsageInfo(options []GetUsageInfoOption) (map[string]string, error) {
	config, err := applyOptions(GetUsageInfoOpts{}, calls.Options(options, getUsageInfoCalls), func(value GetUsageInfoOpts) GetUsageInfoOpts { return value })
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	return infoHeaders(config.Headers, false, "")
}
