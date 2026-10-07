package cloudlimits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudlocation"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Getter is internal orchestration glue owned by Connection, never a builder
// required from SDK callers. Cinder selection follows completed project lookup.
type Getter func(context.Context) (*gophercloud.ServiceClient, error)

type workflow struct{ identity, cinder *cloudread.Source }

func ValidateInput(ctx context.Context, input Input) error {
	if err := cloudread.Context(ctx); err != nil {
		return err
	}
	if !utf8.ValidString(input.NameOrID) {
		return fmt.Errorf("%w: project identity must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, char := range input.NameOrID {
		if unicode.IsControl(char) {
			return fmt.Errorf("%w: project identity must not contain controls", resource.ErrInvalidOption)
		}
	}
	return nil
}

func wrap(ctx context.Context, err error) error {
	return request.Wrap("GetVolumeLimits", "volume limits", cloudread.ContextError(ctx, err))
}

func (p *workflow) guard(ctx context.Context) error {
	err := cloudread.Context(ctx)
	for _, source := range []*cloudread.Source{p.identity, p.cinder} {
		if source != nil {
			err = errors.Join(err, source.Guard(ctx))
		}
	}
	return cloudread.ContextError(ctx, err)
}

func (p *workflow) get(ctx context.Context, source *cloudread.Source, target string) (*rest.Response, error) {
	return rest.DoJSONGuarded(ctx, &source.Client, p.guard, http.MethodGet, target, nil, nil, sourceCodes()...)
}

func ReadSelected(ctx context.Context, cinder, identity *gophercloud.ServiceClient, input Input, options []Option) (*Result, error) {
	if err := ValidateInput(ctx, input); err != nil {
		return nil, wrap(ctx, err)
	}
	captured, err := cloudread.Capture(ctx, cinder, "volume")
	if err != nil {
		return nil, wrap(ctx, err)
	}
	p := &workflow{cinder: captured}
	if input.NameOrID != "" {
		p.identity, err = cloudread.Capture(ctx, identity, "identity")
		if err != nil {
			return nil, wrap(ctx, err)
		}
	}
	policy, err := Prepare(ctx, options, p.guard)
	if err != nil {
		return nil, wrap(ctx, err)
	}
	if policy.Location == nil {
		id, err := cloudlocation.ProjectID(captured.Client.ProviderClient)
		if err != nil {
			return nil, wrap(ctx, err)
		}
		policy.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: bytes.Clone(id)}}
	}
	return p.read(ctx, input, policy, nil)
}

func ReadFrom(ctx context.Context, input Input, policy Options, cinderGetter, identityGetter Getter) (*Result, error) {
	if err := ValidateInput(ctx, input); err != nil {
		return nil, wrap(ctx, err)
	}
	p := &workflow{}
	if input.NameOrID != "" {
		client, err := identityGetter(ctx)
		if err != nil {
			return nil, wrap(ctx, err)
		}
		p.identity, err = cloudread.Capture(ctx, client, "identity")
		if err != nil {
			return nil, wrap(ctx, err)
		}
	}
	return p.read(ctx, input, cloneOptions(policy), cinderGetter)
}

func (p *workflow) read(ctx context.Context, input Input, policy Options, cinderGetter Getter) (*Result, error) {
	result := &Result{}
	if input.NameOrID != "" {
		result.Project = &ProjectResult{}
		if err := p.resolveProject(ctx, input.NameOrID, result.Project); err != nil {
			return result, wrap(ctx, err)
		}
		result.RequestedProjectID = bytes.Clone(result.Project.ID)
	}
	if p.cinder == nil {
		client, err := cinderGetter(ctx)
		if err = errors.Join(err, p.guard(ctx)); err != nil {
			return result, wrap(ctx, err)
		}
		p.cinder, err = cloudread.Capture(ctx, client, "volume")
		if err != nil {
			return result, wrap(ctx, err)
		}
	}
	location, err := locationJSON(policy.Location)
	if err != nil {
		return result, wrap(ctx, err)
	}
	queryValues, err := cloudfilter.RequestQueryValues(result.RequestedProjectID)
	if err != nil {
		return result, wrap(ctx, fmt.Errorf("%w: project query: %w", resource.ErrInvalidOption, err))
	}
	target := p.cinder.Client.ServiceURL("limits")
	if len(queryValues) > 0 {
		target += "?" + url.Values{"project_id": queryValues}.Encode()
	}
	wire, err := p.get(ctx, p.cinder, target)
	result.Observed = observed(wire)
	if err != nil {
		return result, wrap(ctx, err)
	}
	raw, err := responseObject(wire, "limits", true)
	if err != nil {
		return result, wrap(ctx, err)
	}
	logical, err := normalizeLimits(raw, location)
	if err != nil {
		return result, wrap(ctx, wire.Fail(err))
	}
	value, err := rawResource(raw, wire)
	if err != nil {
		return result, wrap(ctx, err)
	}
	if err := p.guard(ctx); err != nil {
		return result, wrap(ctx, wire.Fail(err))
	}
	result.Limits, result.Value = value, logical
	return result, nil
}

func locationJSON(location *resource.CloudLocation) (json.RawMessage, error) {
	if location == nil {
		return nil, fmt.Errorf("%w: limits location snapshot is required", resource.ErrInvalidOption)
	}
	owned := location.Clone()
	for _, text := range []*string{owned.Cloud, owned.RegionName, owned.Project.Name, owned.Project.DomainID, owned.Project.DomainName} {
		if text != nil && !utf8.ValidString(*text) {
			return nil, fmt.Errorf("%w: limits location must be UTF-8", resource.ErrInvalidOption)
		}
	}
	for _, raw := range []json.RawMessage{owned.Zone, owned.Project.ID} {
		if raw != nil && (!utf8.Valid(raw) || !json.Valid(raw)) {
			return nil, fmt.Errorf("%w: limits location fields must be complete UTF-8 JSON", resource.ErrInvalidOption)
		}
	}
	raw, err := json.Marshal(owned)
	if err != nil {
		return nil, fmt.Errorf("%w: limits location: %w", resource.ErrInvalidOption, err)
	}
	return raw, nil
}
