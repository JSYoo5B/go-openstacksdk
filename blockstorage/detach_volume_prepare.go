package blockstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	volumes "github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/volumes"
	servers "github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedDetach struct {
	nova                 attachSource
	cinder               *attachSource
	serverID, volumeID   string
	deleteURL, volumeURL string
	options              preparedDetachOptions
	observed             error
}

func captureDetach(ctx context.Context, nova, cinder *gophercloud.ServiceClient, input DetachVolumeRequest, options []DetachVolumeOption) (*preparedDetach, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	if err := validateAttachRef(input.Server); err != nil {
		return nil, fmt.Errorf("server: %w", err)
	}
	if err := validateAttachRef(input.Volume); err != nil {
		return nil, fmt.Errorf("volume: %w", err)
	}
	novaSource, err := captureAttachSource(ctx, nova, "compute")
	if err != nil {
		return nil, err
	}
	p := &preparedDetach{nova: novaSource}
	if cinder != nil {
		cinderSource, err := captureAttachSource(ctx, cinder, "volume")
		if err != nil {
			return nil, err
		}
		p.cinder = &cinderSource
	}
	// Capture every supplied source before external options. Only the final
	// complete policy decides whether an omitted Cinder source is permitted.
	p.options, err = applyDetachOptions(options, func() error { return p.guard(ctx) })
	if err != nil {
		return nil, attachContextError(ctx, errors.Join(err, p.guard(ctx)))
	}
	if (p.options.wait || input.Volume.IsName()) && p.cinder == nil {
		return nil, attachInvalid("Cinder service client is required for detach waiting or volume name resolution")
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	p.serverID, err = servers.New(&p.nova.client).Resources.ResolveID(ctx, input.Server)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return nil, fmt.Errorf("resolve server: %w", err)
	}
	if err := validateAttachRef(resource.ID(p.serverID)); err != nil {
		return nil, fmt.Errorf("resolve server: %w", err)
	}
	p.volumeID = input.Volume.String()
	if input.Volume.IsName() {
		if err := p.guard(ctx); err != nil {
			return nil, err
		}
		p.volumeID, err = volumes.New(&p.cinder.client).Resources.ResolveID(ctx, input.Volume)
		if err = errors.Join(err, p.guard(ctx)); err != nil {
			return nil, fmt.Errorf("resolve volume: %w", err)
		}
	}
	if err := validateAttachRef(resource.ID(p.volumeID)); err != nil {
		return nil, fmt.Errorf("resolve volume: %w", err)
	}
	p.deleteURL = p.nova.client.ServiceURL("servers", url.PathEscape(p.serverID), "os-volume_attachments", url.PathEscape(p.volumeID))
	if err := validateAttachTarget(&p.nova.client, p.deleteURL); err != nil {
		return nil, err
	}
	if p.cinder != nil {
		p.volumeURL = p.cinder.client.ServiceURL("volumes", url.PathEscape(p.volumeID))
		if err := validateAttachTarget(&p.cinder.client, p.volumeURL); err != nil {
			return nil, err
		}
	}
	return p, p.guard(ctx)
}

func (p *preparedDetach) guard(ctx context.Context) error {
	if p.observed != nil {
		return attachContextError(ctx, p.observed)
	}
	err := p.nova.check(ctx)
	if p.cinder != nil {
		err = errors.Join(err, p.cinder.check(ctx))
	}
	if err != nil {
		p.observed = err
	}
	return attachContextError(ctx, err)
}

func (p *preparedDetach) exchange(ctx context.Context, method string) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	var client *gophercloud.ServiceClient
	var target string
	var codes []int
	switch method {
	case http.MethodDelete:
		client, target = &p.nova.client, p.deleteURL
		codes = []int{http.StatusAccepted, http.StatusNoContent}
	case http.MethodGet:
		if p.cinder == nil || !p.options.wait {
			return nil, attachInvalid("Cinder observation is unavailable for this detach policy")
		}
		client, target = &p.cinder.client, p.volumeURL
		codes = []int{http.StatusOK}
	default:
		return nil, attachInvalid("detach exchange changes its fixed phase method")
	}
	response, err := rest.DoJSON(ctx, client, method, target, nil, nil, codes...)
	if observed := p.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, attachContextError(ctx, err)
}
