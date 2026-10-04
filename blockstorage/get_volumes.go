package blockstorage

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
)

type preparedGetVolumes struct {
	cinder     attachSource
	input      GetVolumesRequest
	options    GetVolumesOpts
	initialURL *url.URL
	observed   error
}

// GetVolumes reads the complete detailed Cinder volume list, then selects every
// attachment occurrence matching the supplied server identity. Device values
// do not affect selection. Duplicate occurrences intentionally share one owned
// volume pointer. Volumes is assigned only after complete success; actual page
// evidence remains available on an error. The helper performs no Nova lookup,
// per-volume GET, refresh, wait or orchestration retry.
func GetVolumes(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumesRequest, options ...GetVolumesOption) (*GetVolumesResult, error) {
	p, err := captureGetVolumes(ctx, cinder, input, options)
	if err != nil {
		return nil, wrapGetVolumesError(ctx, err)
	}
	result := &GetVolumesResult{}
	entries, err := p.materialize(ctx, result)
	if err != nil {
		return result, wrapGetVolumesError(ctx, err)
	}
	selected, err := p.selectVolumes(ctx, entries)
	if err != nil {
		return result, wrapGetVolumesError(ctx, err)
	}
	if err := p.guard(ctx); err != nil {
		return result, wrapGetVolumesError(ctx, err)
	}
	result.Volumes = selected
	return result, nil
}

func captureGetVolumes(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumesRequest, options []GetVolumesOption) (*preparedGetVolumes, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	source, err := captureAttachSource(ctx, cinder, "volume")
	if err != nil {
		return nil, err
	}
	if source.client.Type == "" {
		source.client.Type = "volumev3"
	}
	target := source.client.ServiceURL("volumes", "detail")
	if err := validateAttachTarget(&source.client, target); err != nil {
		return nil, err
	}
	initial, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	p := &preparedGetVolumes{cinder: source, input: input, initialURL: initial}
	p.options, err = applyGetVolumesOptions(options, func() error { return p.guard(ctx) })
	if err != nil {
		return nil, attachContextError(ctx, errors.Join(err, p.guard(ctx)))
	}
	return p, p.guard(ctx)
}

func (p *preparedGetVolumes) guard(ctx context.Context) error {
	if p.observed != nil {
		return attachContextError(ctx, p.observed)
	}
	err := p.cinder.check(ctx)
	if err != nil {
		p.observed = err
	}
	return attachContextError(ctx, err)
}

func (p *preparedGetVolumes) exchange(ctx context.Context, target *url.URL) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	if _, err := p.pageKey(target); err != nil {
		return nil, err
	}
	response, err := rest.DoJSON(ctx, &p.cinder.client, http.MethodGet, target.String(), nil, nil, http.StatusOK)
	if observed := p.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, attachContextError(ctx, err)
}

func wrapGetVolumesError(ctx context.Context, err error) error {
	return request.Wrap("GetVolumes", "volume", attachContextError(ctx, err))
}
