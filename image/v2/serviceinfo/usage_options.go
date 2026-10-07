package serviceinfo

import "github.com/JSYoo5B/gophercloudsdk/request"

// GetUsageInfoOpts has no wire parameters. Usage belongs to the authenticated
// project and is read without a body or a project selection query.
type GetUsageInfoOpts struct{}

type GetUsageInfoOption = request.Option[GetUsageInfoOpts]

func WithGetUsageInfoHeader(key, value string) GetUsageInfoOption {
	return request.WithHeader[GetUsageInfoOpts](key, value)
}

func prepareGetUsageInfo(options []GetUsageInfoOption) (map[string]string, error) {
	config, err := applyOptions(GetUsageInfoOpts{}, options, func(value GetUsageInfoOpts) GetUsageInfoOpts { return value })
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	return infoHeaders(config.Headers, false, "")
}
