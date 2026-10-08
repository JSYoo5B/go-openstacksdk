package blockstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	volumes "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedDeleteVolume struct {
	cinder                          attachSource
	ref                             resource.Ref
	options                         preparedDeleteVolumeOptions
	modern                          bool
	volumeID                        string
	volumeURL, actionURL, deleteURL string
	observed                        error
}

type deleteVolumePhase uint8

const (
	deleteVolumeObservation deleteVolumePhase = iota
	deleteVolumeMutation
)

func captureDeleteVolume(ctx context.Context, cinder *gophercloud.ServiceClient, input DeleteVolumeRequest, options []DeleteVolumeOption) (*preparedDeleteVolume, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	if err := validateAttachRef(input.Volume); err != nil {
		return nil, fmt.Errorf("volume: %w", err)
	}
	source, err := captureAttachSource(ctx, cinder, "volume")
	if err != nil {
		return nil, err
	}
	modern, err := deleteVolumeModern(source.version)
	if err != nil {
		return nil, err
	}
	// A selected microversion must use Cinder's native version headers even
	// when the original source left its service Type empty. Its original
	// Type is still captured and checked, without changing the shared client.
	if source.client.Type == "" {
		source.client.Type = "volumev3"
	}
	p := &preparedDeleteVolume{cinder: source, ref: input.Volume, modern: modern}
	p.options, err = applyDeleteVolumeOptions(options, func() error { return p.guard(ctx) })
	if err != nil {
		return nil, attachContextError(ctx, errors.Join(err, p.guard(ctx)))
	}
	return p, p.guard(ctx)
}

func deleteVolumeModern(version string) (bool, error) {
	if version == "" {
		return false, nil
	}
	if !strings.HasPrefix(version, "3.") {
		return false, attachInvalid("volume deletion microversion must be empty or canonical 3.<minor>")
	}
	minor := version[2:]
	if minor == "" || len(minor) > 1 && minor[0] == '0' {
		return false, attachInvalid("volume deletion microversion must be empty or canonical 3.<minor>")
	}
	for _, character := range []byte(minor) {
		if character < '0' || character > '9' {
			return false, attachInvalid("volume deletion microversion must be empty or canonical ASCII 3.<minor>")
		}
	}
	value, err := strconv.Atoi(minor)
	if err != nil {
		return false, attachInvalid("volume deletion microversion minor overflows")
	}
	return value >= 23, nil
}

func (p *preparedDeleteVolume) guard(ctx context.Context) error {
	if p.observed != nil {
		return attachContextError(ctx, p.observed)
	}
	err := p.cinder.check(ctx)
	if err != nil {
		p.observed = err
	}
	return attachContextError(ctx, err)
}

// resolve runs only after the caller owns a result. A direct logical missing
// Name has no physical 404 evidence; wrapped HTTP/transport errors stay errors.
func (p *preparedDeleteVolume) resolve(ctx context.Context) (bool, error) {
	if err := p.guard(ctx); err != nil {
		return false, err
	}
	id := p.ref.String()
	if p.ref.IsName() {
		var err error
		id, err = volumes.New(&p.cinder.client).Resources.ResolveID(ctx, p.ref)
		guardErr := p.guard(ctx)
		if missing, direct := err.(*resource.NotFoundError); direct && missing.Cause == nil && guardErr == nil {
			return false, nil
		}
		if err = errors.Join(err, guardErr); err != nil {
			return false, fmt.Errorf("resolve volume: %w", err)
		}
	}
	if err := validateAttachRef(resource.ID(id)); err != nil {
		return false, fmt.Errorf("resolve volume: %w", err)
	}
	p.volumeID = id
	p.volumeURL = p.cinder.client.ServiceURL("volumes", url.PathEscape(id))
	p.actionURL = p.cinder.client.ServiceURL("volumes", url.PathEscape(id), "action")
	p.deleteURL = p.volumeURL + "?" + deleteVolumeQuery(p.modern, p.options.policy.Force)
	for _, target := range []string{p.volumeURL, p.actionURL} {
		if err := validateAttachTarget(&p.cinder.client, target); err != nil {
			return false, err
		}
	}
	if err := validateDeleteVolumeMutationTarget(&p.cinder.client, p.deleteURL, p.modern, p.options.policy.Force); err != nil {
		return false, err
	}
	return true, p.guard(ctx)
}

func deleteVolumeQuery(modern, force bool) string {
	query := url.Values{"cascade": {"false"}}
	if modern {
		query.Set("force", strconv.FormatBool(force))
	}
	return query.Encode()
}

// The only query permitted by this workflow is its finite selected-version
// policy. The existing query-free attachment target validator stays unchanged.
func validateDeleteVolumeMutationTarget(source *gophercloud.ServiceClient, target string, modern, force bool) error {
	if err := rest.ValidateTarget(source, target); err != nil {
		return err
	}
	parsed, err := url.Parse(target)
	if err != nil || !attachText(target) || !attachText(parsed.Path) || parsed.ForceQuery || parsed.RawQuery != deleteVolumeQuery(modern, force) {
		return attachInvalid("volume deletion target changes its fixed query policy")
	}
	return nil
}

func validateDeleteVolumeIdentity(volume *VolumeInfo, fixedID string) error {
	if volume == nil || volume.ID == nil {
		return attachInvalid("Cinder volume response must contain a nonnull canonical ID")
	}
	if err := validateAttachRef(resource.ID(*volume.ID)); err != nil {
		return fmt.Errorf("Cinder volume response ID: %w", err)
	}
	if *volume.ID != fixedID {
		return attachInvalid("Cinder volume response must identify the selected volume %q", fixedID)
	}
	// Deletion does not depend on the current status. A nullable or missing
	// status can still be followed by a clean disappearance observation.
	return nil
}

func (p *preparedDeleteVolume) exchange(ctx context.Context, phase deleteVolumePhase) (*rest.Response, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	if p.volumeID == "" {
		return nil, attachInvalid("volume deletion exchange requires a fixed selected ID")
	}
	var method, target string
	var body any
	var codes []int
	switch phase {
	case deleteVolumeObservation:
		method, target = http.MethodGet, p.volumeURL
		codes = []int{http.StatusOK, http.StatusNotFound}
	case deleteVolumeMutation:
		if p.options.policy.Force && !p.modern {
			method, target = http.MethodPost, p.actionURL
			body = map[string]any{"os-force_delete": nil}
			codes = []int{http.StatusCreated, http.StatusAccepted, http.StatusNotFound}
		} else {
			method, target = http.MethodDelete, p.deleteURL
			codes = []int{http.StatusAccepted, http.StatusNoContent, http.StatusNotFound}
		}
	default:
		return nil, attachInvalid("volume deletion exchange changes its fixed phase")
	}
	response, err := rest.DoJSON(ctx, &p.cinder.client, method, target, body, nil, codes...)
	if observed := p.guard(ctx); observed != nil {
		err = errors.Join(err, observed)
		if response != nil {
			err = response.Fail(err)
		}
	}
	return response, attachContextError(ctx, err)
}
