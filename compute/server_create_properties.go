package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

// schedulerHintsKey is the top-level request key used by Python's Server
// resource. Nova accepts it as an alias of os:scheduler_hints.
const schedulerHintsKey = "OS-SCH-HNT:scheduler_hints"

// serverCreateExtras holds create inputs that need resolution or cross-option
// validation after all options have been applied.
type serverCreateExtras struct {
	hints       map[string]json.RawMessage
	group       *resource.Ref
	dataVolumes []resource.Ref
	description *string
	files       []ServerFile
	refresh     bool
}

// ServerFile is one legacy personality file. Contents are base64-encoded by
// the SDK. Nova removed personality in Compute microversion 2.57.
type ServerFile struct {
	Path     string
	Contents []byte
}

// WithSchedulerHints sends arbitrary scheduler hints in the top-level
// OS-SCH-HNT:scheduler_hints object. Values are snapshotted as JSON when the
// option is created. A later call replaces earlier hints; WithServerGroup
// always wins for the "group" key, as Python's group parameter does.
// A non-nil empty map sends an explicit empty hints object.
func WithSchedulerHints(hints map[string]any) CreateServerOption {
	var encodeErr error
	var snapshot map[string]json.RawMessage
	if hints != nil {
		snapshot = make(map[string]json.RawMessage, len(hints))
		for key, value := range hints {
			encoded, err := json.Marshal(value)
			if err != nil {
				encodeErr = errors.Join(encodeErr, fmt.Errorf("scheduler hint %q is not JSON serializable: %v", key, err))
				continue
			}
			snapshot[key] = encoded
		}
	}
	return func(o *createServerOptions) error {
		if hints == nil {
			return invalid("scheduler hints must not be nil")
		}
		if encodeErr != nil {
			return invalid("%v", encodeErr)
		}
		for key := range snapshot {
			if strings.TrimSpace(key) == "" {
				return invalid("scheduler hint key must not be empty")
			}
		}
		o.extra.hints = cloneRawHints(snapshot)
		return nil
	}
}

// WithServerGroup places the server in a server group through the "group"
// scheduler hint. IDs are sent without lookup; names are resolved by listing
// Compute server groups, and missing or duplicate names fail before POST.
func WithServerGroup(ref resource.Ref) CreateServerOption {
	return func(o *createServerOptions) error {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("server group: %w", err)
		}
		o.extra.group = &ref
		return nil
	}
}

// WithServerCount sends min_count and max_count. Without it both keys are
// omitted and Nova creates one server; Python's cloud layer always sends 1.
// Only the first created server is returned by the POST response.
func WithServerCount(minCount, maxCount int) CreateServerOption {
	return func(o *createServerOptions) error {
		if minCount < 1 || maxCount < minCount {
			return invalid("server count requires 1 <= min (%d) <= max (%d)", minCount, maxCount)
		}
		o.base.Min, o.base.Max = minCount, maxCount
		return nil
	}
}

// WithDescription sends the server description, including an explicit empty
// string. It requires Compute microversion 2.19 or later.
func WithDescription(description string) CreateServerOption {
	return func(o *createServerOptions) error { o.extra.description = &description; return nil }
}

// WithAdminPass sends a caller-supplied adminPass. Empty values are rejected;
// Python's cloud layer silently omits them.
func WithAdminPass(password string) CreateServerOption {
	return func(o *createServerOptions) error {
		if password == "" {
			return invalid("admin password must not be empty")
		}
		o.base.AdminPass = password
		return nil
	}
}

// WithDiskConfig sends OS-DCF:diskConfig. Only AUTO and MANUAL are accepted.
func WithDiskConfig(config servers.DiskConfig) CreateServerOption {
	return func(o *createServerOptions) error {
		if config != servers.Auto && config != servers.Manual {
			return invalid("disk config must be AUTO or MANUAL")
		}
		o.base.DiskConfig = config
		return nil
	}
}

// WithAccessIPv4 sends accessIPv4 after validating the IPv4 literal.
func WithAccessIPv4(address string) CreateServerOption {
	return func(o *createServerOptions) error {
		ip := net.ParseIP(address)
		if ip == nil || ip.To4() == nil || strings.Contains(address, ":") {
			return invalid("access IPv4 %q is not an IPv4 address", address)
		}
		o.base.AccessIPv4 = address
		return nil
	}
}

// WithAccessIPv6 sends accessIPv6 after validating the IPv6 literal.
func WithAccessIPv6(address string) CreateServerOption {
	return func(o *createServerOptions) error {
		ip := net.ParseIP(address)
		if ip == nil || !strings.Contains(address, ":") {
			return invalid("access IPv6 %q is not an IPv6 address", address)
		}
		o.base.AccessIPv6 = address
		return nil
	}
}

// WithTags sends server tags. It requires Compute microversion 2.52 or later.
// Tags are sent in the supplied order; Nova validates count and characters.
func WithTags(tags ...string) CreateServerOption {
	tags = append([]string(nil), tags...)
	return func(o *createServerOptions) error {
		if len(tags) == 0 {
			return invalid("tags must not be empty")
		}
		for _, tag := range tags {
			if tag == "" {
				return invalid("tag must not be empty")
			}
		}
		o.base.Tags = append([]string(nil), tags...)
		return nil
	}
}

// WithHostname sends the guest hostname. It requires Compute 2.90 or later.
func WithHostname(hostname string) CreateServerOption {
	return func(o *createServerOptions) error {
		if strings.TrimSpace(hostname) == "" {
			return invalid("hostname must not be empty")
		}
		o.base.Hostname = hostname
		return nil
	}
}

// WithHypervisorHostname requests a compute node. It requires Compute 2.74
// or later and administrator permission by default policy.
func WithHypervisorHostname(hostname string) CreateServerOption {
	return func(o *createServerOptions) error {
		if strings.TrimSpace(hostname) == "" {
			return invalid("hypervisor hostname must not be empty")
		}
		o.base.HypervisorHostname = hostname
		return nil
	}
}

// WithPersonality injects legacy personality files. Contents are snapshotted
// and base64-encoded. Compute microversion 2.57 and later reject it locally.
func WithPersonality(files ...ServerFile) CreateServerOption {
	copied := make([]ServerFile, len(files))
	for i, file := range files {
		copied[i] = ServerFile{Path: file.Path, Contents: append([]byte(nil), file.Contents...)}
	}
	return func(o *createServerOptions) error {
		if len(copied) == 0 {
			return invalid("personality files must not be empty")
		}
		for _, file := range copied {
			if strings.TrimSpace(file.Path) == "" {
				return invalid("personality file path must not be empty")
			}
		}
		o.extra.files = make([]ServerFile, len(copied))
		for i, file := range copied {
			o.extra.files[i] = ServerFile{Path: file.Path, Contents: append([]byte(nil), file.Contents...)}
		}
		return nil
	}
}

// WithDataVolumes attaches existing volumes at boot, after the root disk.
// Each volume is sent as a block_device_mapping_v2 entry with boot_index -1
// and delete_on_termination false. When the server boots from Image, the SDK
// first adds a local image mapping with boot_index 0 and
// delete_on_termination true, as Python's cloud create_server does. Names
// resolve through Block Storage; IDs bypass lookup.
func WithDataVolumes(refs ...resource.Ref) CreateServerOption {
	refs = append([]resource.Ref(nil), refs...)
	return func(o *createServerOptions) error {
		if len(refs) == 0 {
			return invalid("data volumes must not be empty")
		}
		for i, ref := range refs {
			if err := ref.Validate(); err != nil {
				return fmt.Errorf("data volume %d: %w", i, err)
			}
		}
		o.extra.dataVolumes = append([]resource.Ref(nil), refs...)
		return nil
	}
}

func cloneRawHints(hints map[string]json.RawMessage) map[string]json.RawMessage {
	if hints == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(hints))
	for key, value := range hints {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
}

func (o *createServerOptions) validateExtras(version string) error {
	need := func(feature string, minor uint64) error {
		if !microversionAtLeast(version, 2, minor) {
			return fmt.Errorf("%w: %s requires Compute microversion 2.%d or later (client uses %q)", resource.ErrUnsupported, feature, minor, version)
		}
		return nil
	}
	if o.extra.description != nil {
		if err := need("description", 19); err != nil {
			return err
		}
	}
	if len(o.base.Tags) > 0 {
		if err := need("tags", 52); err != nil {
			return err
		}
	}
	if o.base.HypervisorHostname != "" {
		if err := need("hypervisor hostname", 74); err != nil {
			return err
		}
	}
	if o.base.Hostname != "" {
		if err := need("hostname", 90); err != nil {
			return err
		}
	}
	if len(o.extra.files) > 0 && microversionAtLeast(version, 2, 57) {
		return fmt.Errorf("%w: personality was removed in Compute microversion 2.57 (client uses %q)", resource.ErrUnsupported, version)
	}
	return nil
}

type serverGroupName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// prepareServerCreateExtras resolves group and data volume references after
// the core body is prepared. Every lookup completes before the POST.
func (s *Servers) prepareServerCreateExtras(ctx context.Context, o *createServerOptions) (map[string]json.RawMessage, error) {
	if o.extra.description != nil {
		o.fields["description"] = *o.extra.description
	}
	if len(o.extra.files) > 0 {
		o.base.Personality = make(servers.Personality, len(o.extra.files))
		for i, file := range o.extra.files {
			o.base.Personality[i] = &servers.File{Path: file.Path, Contents: append([]byte(nil), file.Contents...)}
		}
	}
	hints := cloneRawHints(o.extra.hints)
	if o.extra.group != nil {
		groupID := o.extra.group.String()
		if o.extra.group.IsName() {
			if err := checkServerCreation(ctx); err != nil {
				return nil, err
			}
			id, err := nativefind.ResolveName(ctx, s.client, "os-server-groups", "server group", "server_groups", *o.extra.group,
				func(g *serverGroupName) string { return g.ID }, func(g *serverGroupName) string { return g.Name }, false, nil)
			if err != nil {
				return nil, s.wrap("resolve server group", err)
			}
			if err := resource.ID(id).Validate(); err != nil {
				return nil, s.wrap("resolve server group", err)
			}
			groupID = id
		}
		if hints == nil {
			hints = map[string]json.RawMessage{}
		}
		encoded, _ := json.Marshal(groupID)
		hints["group"] = encoded
	}
	if len(o.extra.dataVolumes) > 0 {
		if o.bootVolume == nil && o.bootVolumeSize == 0 {
			o.base.BlockDevice = append(o.base.BlockDevice, servers.BlockDevice{
				SourceType: servers.SourceImage, DestinationType: servers.DestinationLocal,
				UUID: o.base.ImageRef, BootIndex: 0, DeleteOnTermination: true,
			})
		}
		for i, ref := range o.extra.dataVolumes {
			volumeID := ref.String()
			if ref.IsName() {
				if s.dependencies.Volume == nil {
					return nil, fmt.Errorf("%w: volume resolver is unavailable", resource.ErrUnsupported)
				}
				id, err := s.dependencies.Volume(ctx, ref)
				err = errors.Join(err, checkServerCreation(ctx))
				if err != nil {
					return nil, s.wrap(fmt.Sprintf("resolve data volume %d", i), err)
				}
				if err := resource.ID(id).Validate(); err != nil {
					return nil, s.wrap(fmt.Sprintf("resolve data volume %d", i), err)
				}
				volumeID = id
			}
			o.base.BlockDevice = append(o.base.BlockDevice, servers.BlockDevice{
				SourceType: servers.SourceVolume, DestinationType: servers.DestinationVolume,
				UUID: volumeID, BootIndex: -1, DeleteOnTermination: false,
			})
		}
	}
	return hints, checkServerCreation(ctx)
}

// WithCreateRefresh follows Python's cloud create_server with wait=False: after
// an accepted POST, Create performs one GET of the created server and returns
// that refreshed model. An ERROR server returns ServerFaultError with the
// refreshed Server. A failed GET returns the POST response with the error; the
// POST is never repeated. It cannot be combined with WithWait.
func WithCreateRefresh() CreateServerOption {
	return func(o *createServerOptions) error { o.extra.refresh = true; return nil }
}

func (s *Servers) refreshCreatedServer(ctx context.Context, created *Server) (*Server, error) {
	if err := resource.ID(created.ID).Validate(); err != nil {
		return created, s.wrap("create/refresh", err)
	}
	response, err := rest.DoJSON(ctx, s.client, http.MethodGet, s.client.ServiceURL("servers", url.PathEscape(created.ID)), nil, nil, 200, 203)
	if err != nil {
		return created, s.wrap("create/refresh", err)
	}
	refreshed, err := rest.Decode[Server](response, "server", func(*Server) *resource.Metadata { return &resource.Metadata{} })
	if err != nil {
		return created, s.wrap("create/refresh", err)
	}
	if refreshed.ID != created.ID {
		return created, s.wrap("create/refresh", response.Fail(invalid("Nova response does not match server %q", created.ID)))
	}
	if strings.EqualFold(refreshed.Status, "ERROR") {
		fault := serverFaultInfo{}
		if object, objectErr := response.Object("server"); objectErr == nil {
			fault = rawServerFault(object)
		}
		return refreshed, newServerFaultError(refreshed, &fault)
	}
	return refreshed, nil
}
