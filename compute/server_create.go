package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

// CreateServerRequest holds required inputs for the server creation workflow.
// Names are resolved by the SDK; explicit IDs bypass lookup requests.
// Image is required unless WithBootVolume selects an existing boot volume.
type CreateServerRequest struct {
	Name   string
	Image  resource.Ref
	Flavor resource.Ref
}

type createServerOptions struct {
	base                          servers.CreateOpts
	networks                      []resource.Ref
	fields                        map[string]any
	wait                          bool
	waitOptions                   []resource.WaitOption
	bootVolume                    *resource.Ref
	deleteBootVolumeOnTermination *bool
}

type CreateServerOption func(*createServerOptions) error

// WithBootVolume boots from an existing volume. Leave CreateServerRequest.Image
// empty when using this option. Names are resolved through Block Storage; IDs
// bypass lookup requests. Nova validates whether the volume is bootable and
// available. The volume is preserved when the server is deleted by default.
func WithBootVolume(ref resource.Ref) CreateServerOption {
	return func(o *createServerOptions) error {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("boot volume: %w", err)
		}
		o.bootVolume = &ref
		return nil
	}
}

// WithDeleteBootVolumeOnTermination controls Nova's deletion of the existing
// boot volume when the server is deleted. It requires WithBootVolume. The
// default is false; this option does not delete a volume on creation failure.
func WithDeleteBootVolumeOnTermination(enabled bool) CreateServerOption {
	return func(o *createServerOptions) error {
		o.deleteBootVolumeOnTermination = &enabled
		return nil
	}
}

func WithMetadata(metadata map[string]string) CreateServerOption {
	metadata = maps.Clone(metadata)
	return func(o *createServerOptions) error { o.base.Metadata = maps.Clone(metadata); return nil }
}

func WithKeyName(name string) CreateServerOption {
	return func(o *createServerOptions) error { o.fields["key_name"] = name; return nil }
}

// WithUserData takes raw bytes; Gophercloud handles base64 encoding.
func WithUserData(data []byte) CreateServerOption {
	data = append([]byte(nil), data...)
	return func(o *createServerOptions) error { o.base.UserData = append([]byte(nil), data...); return nil }
}

func WithConfigDrive(enabled bool) CreateServerOption {
	return func(o *createServerOptions) error { o.base.ConfigDrive = &enabled; return nil }
}

func WithAvailabilityZone(zone string) CreateServerOption {
	return func(o *createServerOptions) error { o.base.AvailabilityZone = zone; return nil }
}

func WithSecurityGroups(names ...string) CreateServerOption {
	names = append([]string(nil), names...)
	return func(o *createServerOptions) error {
		o.base.SecurityGroups = append([]string(nil), names...)
		return nil
	}
}

// WithNetworks replaces the network selection. Defaults are left to Nova.
func WithNetworks(refs ...resource.Ref) CreateServerOption {
	refs = append([]resource.Ref(nil), refs...)
	return func(o *createServerOptions) error {
		if len(refs) == 0 {
			return invalid("network selection must not be empty")
		}
		for _, ref := range refs {
			if err := ref.Validate(); err != nil {
				return err
			}
		}
		o.networks = append([]resource.Ref(nil), refs...)
		return nil
	}
}

// WithWait waits for ACTIVE after successful creation. Wait options are
// validated before creation. Without it, Create returns the asynchronous response.
func WithWait(opts ...resource.WaitOption) CreateServerOption {
	opts = append([]resource.WaitOption(nil), opts...)
	return func(o *createServerOptions) error {
		if err := resource.ValidateWaitOptions(opts...); err != nil {
			return err
		}
		o.wait = true
		o.waitOptions = opts
		return nil
	}
}

// WithField adds an extension field inside the "server" object without a
// builder implementation. Core fields are protected even when omitted.
// Values are snapshotted as JSON; later caller mutation cannot change the option.
// Extension schemas and microversion requirements are validated by OpenStack.
func WithField(key string, value any) CreateServerOption {
	encoded, encodeErr := json.Marshal(value)
	return func(o *createServerOptions) error {
		if strings.TrimSpace(key) == "" {
			return invalid("extension field key must not be empty")
		}
		if reservedServerFields[key] {
			return invalid("extension field %q conflicts with a core field", key)
		}
		if encodeErr != nil {
			return invalid("extension field %q is not JSON serializable: %v", key, encodeErr)
		}
		var copied json.RawMessage = append([]byte(nil), encoded...)
		o.fields[key] = copied
		return nil
	}
}

var reservedServerFields = func() map[string]bool {
	reserved := map[string]bool{"security_groups": true, "user_data": true, "networks": true, "key_name": true, "block_device_mapping": true}
	t := reflect.TypeOf(servers.CreateOpts{})
	for i := 0; i < t.NumField(); i++ {
		key := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if key != "" && key != "-" {
			reserved[key] = true
		}
	}
	return reserved
}()

// serverBody keeps Gophercloud's serialization, including userdata and security
// groups, while providing a reusable built-in extension builder.
type serverBody struct {
	base   servers.CreateOpts
	fields map[string]any
}

func (b serverBody) ToServerCreateMap() (map[string]any, error) {
	body, err := b.base.ToServerCreateMap()
	if err != nil {
		return nil, err
	}
	server, ok := body["server"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected Gophercloud server body")
	}
	maps.Copy(server, b.fields)
	return body, nil
}

// Create resolves named dependencies, creates an image- or volume-backed server and
// optionally waits for ACTIVE. It does not roll back a server after wait failure:
// the created resource is returned alongside the error so it can be inspected.
// WithBootVolume supports an existing volume; creating a new boot volume from
// an image and floating IP setup are separate workflows.
func (s *Servers) Create(ctx context.Context, request CreateServerRequest, opts ...CreateServerOption) (*Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Name) == "" {
		return nil, invalid("server name must not be empty")
	}
	if err := request.Flavor.Validate(); err != nil {
		return nil, fmt.Errorf("flavor: %w", err)
	}
	o := createServerOptions{base: servers.CreateOpts{Name: request.Name}, fields: make(map[string]any)}
	for _, apply := range opts {
		if apply == nil {
			return nil, invalid("nil create option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	hasImage := request.Image != (resource.Ref{})
	if hasImage == (o.bootVolume != nil) {
		return nil, invalid("exactly one of image or boot volume is required")
	}
	if o.bootVolume == nil && o.deleteBootVolumeOnTermination != nil {
		return nil, invalid("boot volume deletion policy requires WithBootVolume")
	}
	if hasImage {
		if err := request.Image.Validate(); err != nil {
			return nil, fmt.Errorf("image: %w", err)
		}
	} else {
		volumeID := o.bootVolume.String()
		if o.bootVolume.IsName() {
			if s.dependencies.Volume == nil {
				return nil, fmt.Errorf("%w: volume resolver is unavailable", resource.ErrUnsupported)
			}
			id, err := s.dependencies.Volume(ctx, *o.bootVolume)
			if err != nil {
				return nil, s.wrap("resolve boot volume", err)
			}
			if err := resource.ID(id).Validate(); err != nil {
				return nil, s.wrap("resolve boot volume", err)
			}
			volumeID = id
		}
		deleteOnTermination := o.deleteBootVolumeOnTermination != nil && *o.deleteBootVolumeOnTermination
		o.base.BlockDevice = []servers.BlockDevice{{
			SourceType: servers.SourceVolume, DestinationType: servers.DestinationVolume,
			UUID: volumeID, BootIndex: 0, DeleteOnTermination: deleteOnTermination,
		}}
	}
	o.base.ImageRef = request.Image.String()
	if request.Image.IsName() {
		if s.dependencies.Image == nil {
			return nil, fmt.Errorf("%w: image resolver is unavailable", resource.ErrUnsupported)
		}
		id, err := s.dependencies.Image(ctx, request.Image)
		if err != nil {
			return nil, s.wrap("resolve image", err)
		}
		o.base.ImageRef = id
	}
	o.base.FlavorRef = request.Flavor.String()
	if request.Flavor.IsName() {
		flavor, err := s.flavors.Find(ctx, request.Flavor)
		if err != nil {
			return nil, s.wrap("resolve flavor", err)
		}
		o.base.FlavorRef = flavor.ID
	}
	if len(o.networks) > 0 {
		networks := make([]servers.Network, 0, len(o.networks))
		for _, ref := range o.networks {
			id := ref.String()
			if ref.IsName() {
				if s.dependencies.Network == nil {
					return nil, fmt.Errorf("%w: network resolver is unavailable", resource.ErrUnsupported)
				}
				resolved, err := s.dependencies.Network(ctx, ref)
				if err != nil {
					return nil, s.wrap("resolve network", err)
				}
				id = resolved
			}
			networks = append(networks, servers.Network{UUID: id})
		}
		o.base.Networks = networks
	}
	created, err := servers.Create(ctx, s.client, serverBody{base: o.base, fields: o.fields}, nil).Extract()
	if err != nil {
		return nil, s.wrap("create", err)
	}
	if !o.wait {
		return created, nil
	}
	ready, err := s.Wait(ctx, resource.ID(created.ID), "ACTIVE", o.waitOptions...)
	if err != nil {
		return created, s.wrap("create/wait", err)
	}
	return ready, nil
}

// Ensure the extension adapter satisfies the upstream contract inside the SDK.
var _ servers.CreateOptsBuilder = serverBody{}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func (s *Servers) wrap(op string, err error) error {
	return &resource.OperationError{Operation: op, Resource: "server", Cause: err}
}
