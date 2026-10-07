package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gophercloudsdk/internal/nativefind"
	"gophercloudsdk/internal/rest"
	"maps"
	"reflect"
	"strconv"
	"strings"

	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

// CreateServerRequest holds required inputs for the server creation workflow.
// Names are resolved by the SDK; explicit IDs bypass lookup requests.
// Image is required unless WithBootVolume selects an existing boot volume.
// WithBootVolumeSize uses Image to create a new volume before booting.
type CreateServerRequest struct {
	Name   string
	Image  resource.Ref
	Flavor resource.Ref
}

type createServerOptions struct {
	base                          servers.CreateOpts
	networkInterfaces             []ServerNetworkInterface
	networkMode                   string
	fields                        map[string]any
	wait                          bool
	waitOptions                   []resource.WaitOption
	waitPolicy                    *resource.WaitPolicy
	bootVolume                    *resource.Ref
	bootVolumeSize                int
	bootVolumeType                string
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

// WithBootVolumeSize boots from a new volume created from the request's Image.
// The positive size is expressed in GiB. It cannot be combined with
// WithBootVolume, which selects an existing volume. Nova creates the volume
// as part of the server request; the SDK does not make a separate Cinder POST.
func WithBootVolumeSize(sizeGiB int) CreateServerOption {
	return func(o *createServerOptions) error {
		if sizeGiB <= 0 {
			return invalid("boot volume size must be positive")
		}
		o.bootVolumeSize = sizeGiB
		return nil
	}
}

// WithBootVolumeType chooses the type of a new boot volume. It requires
// WithBootVolumeSize and Compute microversion 2.67 or later. The configured
// type name is sent directly; Cinder validates whether the type is available.
func WithBootVolumeType(name string) CreateServerOption {
	return func(o *createServerOptions) error {
		if strings.TrimSpace(name) == "" {
			return invalid("boot volume type must not be empty")
		}
		o.bootVolumeType = name
		return nil
	}
}

// WithDeleteBootVolumeOnTermination controls Nova's deletion of the boot volume
// when the server is deleted. It requires WithBootVolume or WithBootVolumeSize.
// The default is false; the SDK does not delete a volume on creation failure.
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

// WithWait waits for ACTIVE after successful creation. Wait options are
// validated before creation. Without it, Create returns the asynchronous response.
func WithWait(opts ...resource.WaitOption) CreateServerOption {
	opts = append([]resource.WaitOption(nil), opts...)
	return func(o *createServerOptions) error {
		policy, err := resource.PrepareWaitOptionsFor[servers.Server](opts...)
		if err != nil {
			return err
		}
		o.wait = true
		o.waitPolicy = &policy
		o.waitOptions = []resource.WaitOption{resource.WithWaitPolicy(policy)}
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
// WithBootVolume uses an existing volume; WithBootVolumeSize creates a new
// volume from Image. Floating IP setup is a separate workflow.
func (s *Servers) Create(ctx context.Context, request CreateServerRequest, opts ...CreateServerOption) (*Server, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o, err := s.prepareCreateServerOptions(request, opts...)
	if err != nil {
		return nil, err
	}
	created, err := s.createServerPrepared(ctx, request, o)
	if err != nil {
		return nil, err
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

// Pure input preparation is shared by asynchronous Create and compound workflows.
// Each caller owns the resulting values; option closures are applied once.
func (s *Servers) prepareCreateServerOptions(request CreateServerRequest, opts ...CreateServerOption) (createServerOptions, error) {
	var o createServerOptions
	if strings.TrimSpace(request.Name) == "" {
		return o, invalid("server name must not be empty")
	}
	if err := request.Flavor.Validate(); err != nil {
		return o, fmt.Errorf("flavor: %w", err)
	}
	o = createServerOptions{base: servers.CreateOpts{Name: request.Name}, fields: make(map[string]any)}
	for _, apply := range opts {
		if apply == nil {
			return o, invalid("nil create option")
		}
		if err := apply(&o); err != nil {
			return o, err
		}
	}
	if err := o.validateNetworkVersion(s.client.Microversion); err != nil {
		return o, err
	}
	hasImage := request.Image != (resource.Ref{})
	if hasImage == (o.bootVolume != nil) {
		return o, invalid("exactly one of image or boot volume is required")
	}
	if o.bootVolume != nil && o.bootVolumeSize > 0 {
		return o, invalid("existing boot volume and new boot volume size are mutually exclusive")
	}
	if o.bootVolume == nil && o.bootVolumeSize == 0 && o.deleteBootVolumeOnTermination != nil {
		return o, invalid("boot volume deletion policy requires WithBootVolume or WithBootVolumeSize")
	}
	if o.bootVolumeType != "" {
		if o.bootVolumeSize == 0 {
			return o, invalid("boot volume type requires WithBootVolumeSize")
		}
		if !microversionAtLeast(s.client.Microversion, 2, 67) {
			return o, fmt.Errorf("%w: boot volume type requires Compute microversion 2.67 or later (client uses %q)", resource.ErrUnsupported, s.client.Microversion)
		}
	}
	if hasImage {
		if err := request.Image.Validate(); err != nil {
			return o, fmt.Errorf("image: %w", err)
		}
	}
	return o, nil
}

// Resolve dependencies and POST without waiting, preserving the actual creation
// response so the caller can validate subsequent observations against its ID.
func (s *Servers) prepareServerCreateBody(ctx context.Context, request CreateServerRequest, o createServerOptions) (serverBody, error) {
	if err := checkServerCreation(ctx); err != nil {
		return serverBody{}, err
	}
	if o.bootVolume != nil {
		volumeID := o.bootVolume.String()
		if o.bootVolume.IsName() {
			if s.dependencies.Volume == nil {
				return serverBody{}, fmt.Errorf("%w: volume resolver is unavailable", resource.ErrUnsupported)
			}
			id, err := s.dependencies.Volume(ctx, *o.bootVolume)
			err = errors.Join(err, checkServerCreation(ctx))
			if err != nil {
				return serverBody{}, s.wrap("resolve boot volume", err)
			}
			if err := resource.ID(id).Validate(); err != nil {
				return serverBody{}, s.wrap("resolve boot volume", err)
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
			return serverBody{}, fmt.Errorf("%w: image resolver is unavailable", resource.ErrUnsupported)
		}
		id, err := s.dependencies.Image(ctx, request.Image)
		err = errors.Join(err, checkServerCreation(ctx))
		if err != nil {
			return serverBody{}, s.wrap("resolve image", err)
		}
		if o.bootVolumeSize > 0 || rest.HasOperationGuard(ctx) {
			if err := resource.ID(id).Validate(); err != nil {
				return serverBody{}, s.wrap("resolve image", err)
			}
		}
		o.base.ImageRef = id
	}
	if o.bootVolumeSize > 0 {
		deleteOnTermination := o.deleteBootVolumeOnTermination != nil && *o.deleteBootVolumeOnTermination
		o.base.BlockDevice = []servers.BlockDevice{{
			SourceType: servers.SourceImage, DestinationType: servers.DestinationVolume,
			UUID: o.base.ImageRef, BootIndex: 0, DeleteOnTermination: deleteOnTermination,
			VolumeSize: o.bootVolumeSize, VolumeType: o.bootVolumeType,
		}}
		o.base.ImageRef = ""
	}
	o.base.FlavorRef = request.Flavor.String()
	if request.Flavor.IsName() {
		if err := checkServerCreation(ctx); err != nil {
			return serverBody{}, err
		}
		if rest.HasOperationGuard(ctx) {
			id, err := nativefind.ResolveName(ctx, s.client, "flavors/detail", "flavor", "flavors", request.Flavor,
				func(f *Flavor) string { return f.ID }, func(f *Flavor) string { return f.Name }, false, nil)
			if err != nil {
				return serverBody{}, s.wrap("resolve flavor", err)
			}
			o.base.FlavorRef = id
		} else {
			flavor, err := s.flavors.Find(ctx, request.Flavor)
			if err != nil {
				return serverBody{}, s.wrap("resolve flavor", err)
			}
			o.base.FlavorRef = flavor.ID
		}
	}
	if err := s.prepareServerNetworks(ctx, &o); err != nil {
		return serverBody{}, err
	}
	return serverBody{base: o.base, fields: o.fields}, checkServerCreation(ctx)
}

func checkServerCreation(ctx context.Context) error {
	return errors.Join(ctx.Err(), context.Cause(ctx), rest.CheckOperationGuard(ctx))
}

func (s *Servers) createServerPrepared(ctx context.Context, request CreateServerRequest, o createServerOptions) (*Server, error) {
	body, err := s.prepareServerCreateBody(ctx, request, o)
	if err != nil {
		return nil, err
	}
	created, err := servers.Create(ctx, s.client, body, nil).Extract()
	if err != nil {
		return nil, s.wrap("create", err)
	}
	return created, nil
}

// Ensure the extension adapter satisfies the upstream contract inside the SDK.
var _ servers.CreateOptsBuilder = serverBody{}

// Microversions contain exactly two numeric components, rather than floats:
// 2.100 is newer than 2.67, while 2.9 is older.
func microversionAtLeast(value string, major, minor uint64) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	gotMajor, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return false
	}
	gotMinor, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return false
	}
	return gotMajor > major || gotMajor == major && gotMinor >= minor
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func (s *Servers) wrap(op string, err error) error {
	return &resource.OperationError{Operation: op, Resource: "server", Cause: err}
}
