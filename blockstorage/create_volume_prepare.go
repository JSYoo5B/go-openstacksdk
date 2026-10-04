package blockstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	images "gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

type createVolumeImageSource struct {
	source               *gophercloud.ServiceClient
	client               gophercloud.ServiceClient
	provider             *gophercloud.ProviderClient
	endpoint, base, kind string
	version              string
}

type preparedCreateVolume struct {
	cinder                          attachSource
	glance                          *createVolumeImageSource
	options                         preparedCreateVolumeOptions
	volumeID                        string
	createURL, volumeURL, actionURL string
	observed                        error
}

type createVolumePhase uint8

const (
	createVolumeCreation createVolumePhase = iota
	createVolumeObservation
	createVolumeBootableAction
)

func captureCreateVolume(ctx context.Context, cinder, glance *gophercloud.ServiceClient, input CreateVolumeRequest, options []CreateVolumeOption) (*preparedCreateVolume, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	// External options cannot change the selected image through the caller's
	// pointer after its reference has passed preflight validation.
	var image *resource.Ref
	if input.Image != nil {
		owned := *input.Image
		if err := validateAttachRef(owned); err != nil {
			return nil, fmt.Errorf("image: %w", err)
		}
		image = &owned
	}
	cinderSource, err := captureAttachSource(ctx, cinder, "volume")
	if err != nil {
		return nil, err
	}
	p := &preparedCreateVolume{cinder: cinderSource}
	if glance != nil {
		imageSource, err := captureCreateVolumeImageSource(ctx, glance)
		if err != nil {
			return nil, err
		}
		p.glance = &imageSource
	}
	p.options, err = applyCreateVolumeOptions(options, func() error { return p.guard(ctx) })
	if err != nil {
		return nil, attachContextError(ctx, errors.Join(err, p.guard(ctx)))
	}
	if image != nil && image.IsName() && p.glance == nil {
		return nil, attachInvalid("Glance service client is required for image name resolution")
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	volume, err := attachmentObject(p.options.body["volume"])
	if err != nil {
		return nil, attachInvalid("volume creation attributes must contain a volume object: %v", err)
	}
	// The cloud helper always supplies size, including zero and negative
	// values; the Cinder server owns validation of the requested size.
	volume["size"], err = json.Marshal(input.Size)
	if err != nil {
		return nil, err
	}
	if image != nil {
		imageID := image.String()
		if image.IsName() {
			imageID, err = images.New(&p.glance.client).Resources.ResolveID(ctx, *image)
			if err = errors.Join(err, p.guard(ctx)); err != nil {
				return nil, fmt.Errorf("resolve image: %w", err)
			}
		}
		if err := validateAttachRef(resource.ID(imageID)); err != nil {
			return nil, fmt.Errorf("resolve image: %w", err)
		}
		volume["imageRef"], err = json.Marshal(imageID)
		if err != nil {
			return nil, err
		}
	}
	p.options.body["volume"], err = json.Marshal(volume)
	if err != nil {
		return nil, err
	}
	p.createURL = p.cinder.client.ServiceURL("volumes")
	if err := validateAttachTarget(&p.cinder.client, p.createURL); err != nil {
		return nil, err
	}
	return p, p.guard(ctx)
}

func captureCreateVolumeImageSource(ctx context.Context, source *gophercloud.ServiceClient) (createVolumeImageSource, error) {
	headers, err := validateCreateVolumeImageSource(ctx, source)
	if err != nil {
		return createVolumeImageSource{}, err
	}
	client := *source
	client.MoreHeaders = maps.Clone(headers)
	return createVolumeImageSource{
		source: source, client: client, provider: source.ProviderClient,
		endpoint: source.Endpoint, base: source.ResourceBase,
		kind: source.Type, version: source.Microversion,
	}, nil
}

func validateCreateVolumeImageSource(ctx context.Context, source *gophercloud.ServiceClient) (map[string]string, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	if source == nil || source.ProviderClient == nil {
		return nil, attachInvalid("image service client is required")
	}
	if source.Type != "" && source.Type != "image" {
		return nil, fmt.Errorf("%w: image service client is required", resource.ErrUnsupported)
	}
	if !attachText(source.Microversion) {
		return nil, attachInvalid("invalid image service microversion")
	}
	for index, endpoint := range []string{source.Endpoint, source.ResourceBaseURL()} {
		if err := validateAttachEndpoint(endpoint, index == 1); err != nil {
			return nil, err
		}
	}
	if err := rest.ValidateTarget(source, source.ResourceBaseURL()); err != nil {
		return nil, err
	}
	// Native Glance uses the generic image version header. Its ordinary
	// extension headers remain captured, while Nova/Cinder version headers
	// cannot override this service's selected role or microversion.
	return canonicalAttachHeaders(source.MoreHeaders, "image", source.Microversion)
}

func (source *createVolumeImageSource) check(ctx context.Context) error {
	if source.source == nil || source.source.ProviderClient != source.provider || source.source.Endpoint != source.endpoint || source.source.ResourceBase != source.base || source.source.Type != source.kind || source.source.Microversion != source.version {
		return attachInvalid("image creation source or route changed")
	}
	_, err := validateCreateVolumeImageSource(ctx, source.source)
	return err
}

func (p *preparedCreateVolume) guard(ctx context.Context) error {
	if p.observed != nil {
		return attachContextError(ctx, p.observed)
	}
	err := p.cinder.check(ctx)
	if p.glance != nil {
		err = errors.Join(err, p.glance.check(ctx))
	}
	if err != nil {
		p.observed = err
	}
	return attachContextError(ctx, err)
}

func validateCreateVolumeIdentity(volume *VolumeInfo, fixedID string) error {
	if volume == nil || volume.ID == nil {
		return attachInvalid("Cinder volume response must contain a nonnull canonical ID")
	}
	if err := validateAttachRef(resource.ID(*volume.ID)); err != nil {
		return fmt.Errorf("Cinder volume response ID: %w", err)
	}
	if fixedID != "" && *volume.ID != fixedID {
		return attachInvalid("Cinder volume response must identify the created volume %q", fixedID)
	}
	if volume.Status == nil {
		return attachInvalid("Cinder volume response must contain a nonnull status")
	}
	return nil
}

func (p *preparedCreateVolume) bindVolume(ctx context.Context, volume *VolumeInfo) error {
	if err := validateCreateVolumeIdentity(volume, ""); err != nil {
		return err
	}
	if err := p.guard(ctx); err != nil {
		return err
	}
	p.volumeID = *volume.ID
	p.volumeURL = p.cinder.client.ServiceURL("volumes", url.PathEscape(p.volumeID))
	p.actionURL = p.cinder.client.ServiceURL("volumes", url.PathEscape(p.volumeID), "action")
	for _, target := range []string{p.volumeURL, p.actionURL} {
		if err := validateAttachTarget(&p.cinder.client, target); err != nil {
			return err
		}
	}
	return p.guard(ctx)
}

func (p *preparedCreateVolume) exchange(ctx context.Context, phase createVolumePhase) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	var method, target string
	var body any
	var code int
	switch phase {
	case createVolumeCreation:
		method, target, body, code = http.MethodPost, p.createURL, p.options.body, http.StatusAccepted
	case createVolumeObservation:
		if p.volumeID == "" || !p.options.wait {
			return nil, attachInvalid("volume creation observation requires a bound volume and waiting policy")
		}
		method, target, code = http.MethodGet, p.volumeURL, http.StatusOK
	case createVolumeBootableAction:
		if p.volumeID == "" || p.options.policy.Bootable == nil || !*p.options.policy.Bootable {
			return nil, attachInvalid("volume bootable action requires a bound volume and true bootable policy")
		}
		method, target, code = http.MethodPost, p.actionURL, http.StatusOK
		body = map[string]any{"os-set_bootable": map[string]any{"bootable": true}}
	default:
		return nil, attachInvalid("volume creation exchange changes its fixed phase")
	}
	response, err := rest.DoJSON(ctx, &p.cinder.client, method, target, body, nil, code)
	if observed := p.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, attachContextError(ctx, err)
}
