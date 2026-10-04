package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudlocation"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type reader struct {
	schema        readSchema
	source        *cloudread.Source
	location      resource.CloudLocation
	memberFailure error
}

func capture(ctx context.Context, client *gophercloud.ServiceClient) (*reader, error) {
	return captureWithSchema(ctx, client, snapshotReadSchema())
}

func captureWithSchema(ctx context.Context, client *gophercloud.ServiceClient, schema readSchema) (*reader, error) {
	source, err := cloudread.Capture(ctx, client, "volume")
	if err != nil {
		return nil, err
	}
	return &reader{source: source, schema: schema}, nil
}

// Own the recorded authentication scope after original options, before HTTP.
// A subsequent native authentication or retry never changes this cloud view.
func (p *reader) ownLocation(ctx context.Context, supplied *resource.CloudLocation) error {
	if supplied != nil {
		p.location = supplied.Clone()
	} else {
		id, err := cloudlocation.ProjectID(p.source.Client.ProviderClient)
		if err != nil {
			return err
		}
		p.location = resource.CloudLocation{Project: resource.CloudProject{ID: bytes.Clone(id)}}
	}
	if err := ValidateLocation(&p.location); err != nil {
		return err
	}
	return p.source.Guard(ctx)
}

// ValidateLocation is shared with Connection's service-free preparation.
// The resource descriptor supplies Zone; the configured Zone is not used.
func ValidateLocation(location *resource.CloudLocation) error {
	if location == nil {
		return invalid("Cinder resource location is required")
	}
	for _, text := range []*string{location.Cloud, location.RegionName, location.Project.Name, location.Project.DomainID, location.Project.DomainName} {
		if text != nil && !utf8.ValidString(*text) {
			return invalid("Cinder resource location text must be UTF-8")
		}
	}
	_, err := location.ForResource(nil, nil)
	return err
}

// ValidateListOptions checks raw controls without selecting a service.
func ValidateListOptions(value ListOptions) error {
	_, err := compileList(cloneList(value))
	return err
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func wrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume snapshot", cloudread.ContextError(ctx, err))
}

// ValidateID deliberately admits only one unescaped UTF-8 member segment.
// Automatic name lookup uses the shared safe-route/list-only decision instead.
func ValidateID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return invalid("snapshot ID must be UTF-8")
	}
	for _, char := range id {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return invalid("snapshot ID must not contain whitespace or controls")
		}
	}
	return nil
}

func (p *reader) target(parts ...string) (string, error) {
	target := p.source.Client.ServiceURL(parts...)
	if err := rest.ValidateTarget(&p.source.Client, target); err != nil {
		return "", err
	}
	return target, nil
}

type entry struct {
	view     json.RawMessage
	resource *resource.RawResource
	id, name string
	seeded   bool
}
