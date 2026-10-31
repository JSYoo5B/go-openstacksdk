package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/usage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudcompute"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudserver"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/serverhostvars"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// CloudServerReadOpts owns the local controls of the cloud server reads; see
// compute.CloudServerReadOpts.
type CloudServerReadOpts = compute.CloudServerReadOpts

// CloudServerReadOption mutates CloudServerReadOpts.
type CloudServerReadOption = compute.CloudServerReadOption

// WithCloudServerFilters sets owned JSON filters.
func WithCloudServerFilters(value json.RawMessage) CloudServerReadOption {
	owned := bytes.Clone(value)
	return func(o *CloudServerReadOpts) error {
		o.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			o.Filters = &raw
		}
		return nil
	}
}

// WithCloudServerExpression sets a JMESPath filter expression.
func WithCloudServerExpression(expression string) CloudServerReadOption {
	raw, _ := json.Marshal(expression)
	return WithCloudServerFilters(raw)
}

// WithCloudServerListOptions forwards the Python-named list controls (all
// projects, details, query pushdown, paging caps, headers, microversion).
func WithCloudServerListOptions(options ...servers.RecordFindOption) CloudServerReadOption {
	owned := slices.Clone(options)
	return func(o *CloudServerReadOpts) error { o.List = append(o.List, owned...); return nil }
}

// WithCloudServerBare skips address expansion of typed getters.
func WithCloudServerBare(bare bool) CloudServerReadOption {
	return func(o *CloudServerReadOpts) error { o.Bare = bare; return nil }
}

// WithCloudServerAddressOptions forwards options to ExpandServerInterfaces.
func WithCloudServerAddressOptions(options ...compute.ServerAddressOption) CloudServerReadOption {
	owned := slices.Clone(options)
	return func(o *CloudServerReadOpts) error { o.Address = append(o.Address, owned...); return nil }
}

// CloudServerSearchResult is the SearchServerRecords result; see
// compute.CloudServerSearchResult.
type CloudServerSearchResult = compute.CloudServerSearchResult

// CloudServerResult is the outcome of the typed server getters and actions;
// see compute.CloudServerResult.
type CloudServerResult = compute.CloudServerResult

// CloudServerUpdate lists the attributes UpdateServer accepts; see
// compute.CloudServerUpdate.
type CloudServerUpdate = compute.CloudServerUpdate

// CloudServerSearch is the SearchServers result; see compute.CloudServerSearch.
type CloudServerSearch = compute.CloudServerSearch

// CloudServerActionOpts owns the wait, address and floating-IP controls shared
// by DeleteServer and RebuildServer; see compute.CloudServerActionOpts.
type CloudServerActionOpts = compute.CloudServerActionOpts

// CloudServerActionOption mutates CloudServerActionOpts.
type CloudServerActionOption = compute.CloudServerActionOption

// cloudServerDependencies supplies the Connection facilities of the cloud
// server reads and actions.
func (c *Connection) cloudServerDependencies() cloudserver.Dependencies {
	return cloudserver.Dependencies{
		Guard:             c.recordListContext,
		Compute:           c.ComputeV2,
		Location:          c.CurrentLocation,
		ExpandInterfaces:  c.ExpandServerInterfaces,
		Hostvars:          c.cloudServerHostvars,
		FloatingIPSource:  func() compute.FloatingIPSource { return c.options.serverAddresses.FloatingIPSource() },
		GetFloatingIP:     c.GetFloatingIP,
		DeleteFloatingIP:  c.DeleteFloatingIP,
		HasSecurityGroups: c.hasSecurityGroups,
		GetSecurityGroup: func(ctx context.Context, group string) (*network.SecurityGroupReadResult, error) {
			return c.GetSecurityGroupRecord(ctx, group)
		},
	}
}

// ListServerRecords maps list_servers with bare=True: one eager consumption of
// GET /servers/detail (WithCloudServerListOptions selects details, all
// projects and query pushdown) without address expansion. On a failure the
// rows consumed before it are returned beside the error.
func (c *Connection) ListServerRecords(ctx context.Context, options ...CloudServerReadOption) ([]*servers.ServerFindRecord, error) {
	return cloudserver.ListRecords(ctx, c.cloudServerDependencies(), options)
}

// SearchServerRecords maps search_servers: list every server, then apply the
// exact/glob name-or-ID phase and the local dictionary or JMESPath filters.
// Nothing is pushed to Nova unless it is an explicit list option.
func (c *Connection) SearchServerRecords(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudServerSearchResult, error) {
	return cloudserver.SearchRecords(ctx, c.cloudServerDependencies(), nameOrID, options)
}

// GetServerByID maps get_server_by_id: GET /servers/{id}; a 404 is (nil, nil)
// as Python's None, and a found server is expanded with
// ExpandServerInterfaces unless WithCloudServerBare(true) is set. Any other
// failure, including an expansion failure beside the retained server, is returned.
func (c *Connection) GetServerByID(ctx context.Context, id string, options ...CloudServerReadOption) (*CloudServerResult, error) {
	return cloudserver.GetByID(ctx, c.cloudServerDependencies(), id, options)
}

// UpdateServer maps update_server: find the server strictly by ID or name
// (a missing server is resource.ErrNotFound and nothing is sent), then PUT only
// the supplied attributes. With no attribute Python's commit sends nothing; this
// returns the found record with Sent false and a nil Server.
func (c *Connection) UpdateServer(ctx context.Context, nameOrID string, update CloudServerUpdate, options ...CloudServerReadOption) (*CloudServerResult, error) {
	return cloudserver.Update(ctx, c.cloudServerDependencies(), nameOrID, update, options)
}

// WithCloudServerDetailed selects Python's detailed=True expansion
// (GetOpenStackVars hostvars) instead of the interface expansion.
func WithCloudServerDetailed(detailed bool) CloudServerReadOption {
	return func(o *CloudServerReadOpts) error { o.Detailed = detailed; return nil }
}

// WithCloudServerActionDetailed selects the detailed=True expansion of the
// RebuildServer result.
func WithCloudServerActionDetailed(detailed bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Detailed = detailed; return nil }
}

// GetServer maps get_server: find_server(name_or_id, details=True) through
// FindRecord (a missing server is (nil, nil)), then the default expansion:
// interfaces, hostvars with WithCloudServerDetailed(true), nothing with
// WithCloudServerBare(true). With WithCloudServerFilters (Python's deprecated
// filters argument) every server is listed bare and filtered locally; several
// matches are a resource.AmbiguousError and a JMESPath value that is not a
// row selection is rejected. Expansion failures are returned beside the found
// server.
func (c *Connection) GetServer(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudServerResult, error) {
	return cloudserver.Get(ctx, c.cloudServerDependencies(), nameOrID, options)
}

// ListServers maps list_servers with its default bare=False: one eager
// consumption of the server list (as ListServerRecords), then every row is
// expanded in order with the interface expansion, the hostvars of
// WithCloudServerDetailed(true), or nothing with WithCloudServerBare(true).
// On a list or expansion failure the results built before it are returned
// beside the error; nothing is retried.
func (c *Connection) ListServers(ctx context.Context, options ...CloudServerReadOption) ([]*CloudServerResult, error) {
	return cloudserver.List(ctx, c.cloudServerDependencies(), options)
}

// SearchServers maps search_servers with its default bare=False: ListServers,
// then the exact/glob name-or-ID phase and the local filters over each
// expanded view (CloudServerResultView). Filters never reach Nova.
func (c *Connection) SearchServers(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudServerSearch, error) {
	return cloudserver.Search(ctx, c.cloudServerDependencies(), nameOrID, options)
}

// CloudServerResultView renders the JSON mapping Python filters and returns
// for one server: the hostvars mapping when detailed, else the record's
// Source Resource view (or the typed server JSON) with the interface
// fields of the address expansion set over it.
func CloudServerResultView(result *CloudServerResult) (json.RawMessage, error) {
	return compute.CloudServerResultView(result)
}

// WithCloudServerWait enables the post-request wait.
func WithCloudServerWait(wait bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Wait = wait; return nil }
}

// WithCloudServerTimeout replaces the 180-second wait budget.
func WithCloudServerTimeout(timeout time.Duration) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Timeout, o.Unlimited = timeout, false; return nil }
}

// WithCloudServerUnlimitedWait maps a Python timeout of None.
func WithCloudServerUnlimitedWait() CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Unlimited = true; return nil }
}

// WithCloudServerWaitOptions appends wait options that apply after the timeout.
func WithCloudServerWaitOptions(options ...resource.WaitOption) CloudServerActionOption {
	owned := slices.Clone(options)
	return func(o *CloudServerActionOpts) error { o.WaitOptions = append(o.WaitOptions, owned...); return nil }
}

// WithCloudServerDeleteIPs deletes the floating IPs found in the server's
// addresses before the server is deleted.
func WithCloudServerDeleteIPs(enabled bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.DeleteIPs = enabled; return nil }
}

// WithCloudServerDeleteIPRetry sets the floating-IP delete retry count.
func WithCloudServerDeleteIPRetry(retries int) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { x := retries; o.DeleteIPRetry = &x; return nil }
}

// WithCloudServerFloatingIPOptions forwards options to the floating-IP lookup and delete.
func WithCloudServerFloatingIPOptions(query []compute.FloatingIPQueryOption, remove []compute.FloatingIPDeleteOption) CloudServerActionOption {
	q, d := slices.Clone(query), slices.Clone(remove)
	return func(o *CloudServerActionOpts) error {
		o.FloatingIPQuery = append(o.FloatingIPQuery, q...)
		o.FloatingIPDelete = append(o.FloatingIPDelete, d...)
		return nil
	}
}

// WithCloudServerAdminPassword sets the rebuild admin password; an empty value is omitted.
func WithCloudServerAdminPassword(password string) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.AdminPassword = password; return nil }
}

// WithCloudServerRebuildOptions forwards extension options to servers.API.Rebuild.
func WithCloudServerRebuildOptions(options ...servers.RebuildOption) CloudServerActionOption {
	owned := slices.Clone(options)
	return func(o *CloudServerActionOpts) error { o.Rebuild = append(o.Rebuild, owned...); return nil }
}

// WithCloudServerActionBare skips address expansion of the rebuild result.
func WithCloudServerActionBare(bare bool) CloudServerActionOption {
	return func(o *CloudServerActionOpts) error { o.Bare = bare; return nil }
}

// WithCloudServerActionAddressOptions forwards options to ExpandServerInterfaces.
func WithCloudServerActionAddressOptions(options ...compute.ServerAddressOption) CloudServerActionOption {
	owned := slices.Clone(options)
	return func(o *CloudServerActionOpts) error { o.Address = append(o.Address, owned...); return nil }
}

// DeleteServer maps delete_server. The server is found like find_server with
// its defaults (details, current project). A missing server is (false, nil)
// with no DELETE. With WithCloudServerDeleteIPs each floating address found in
// the server's addresses is looked up and deleted first, through the Neutron or
// Nova floating-IP backend the Connection selects; an address whose lookup
// reports not found or finds nothing is skipped, and a floating IP that is not
// deleted aborts before the server DELETE. When the Connection's
// floating_ip_source is none (ServerAddressPolicy or clouds.yaml), the
// floating-IP cleanup is skipped without any request, as Python's
// _has_floating_ips gate does. A 404 on the DELETE is (false, nil).
// With WithCloudServerWait the call then observes absence for the timeout
// (default 180 seconds). The DELETE is never replayed; once it was accepted a
// wait failure returns (true, err) because the remote delete was requested.
func (c *Connection) DeleteServer(ctx context.Context, nameOrID string, options ...CloudServerActionOption) (bool, error) {
	return cloudserver.Delete(ctx, c.cloudServerDependencies(), nameOrID, options)
}

// RebuildServer maps rebuild_server: POST the rebuild action with imageRef and,
// when nonempty, adminPass. An empty image ID is rejected before HTTP (Python
// raises TypeError without it). With WithCloudServerWait the server is polled
// until ACTIVE (ERROR fails) for the timeout, and the admin password returned by
// the action, else the supplied one, is restored on the waited server because a
// later GET never repeats it. Without wait the action response is returned.
// A failed wait returns the action result beside the error and never replays
// the rebuild. The result is expanded unless WithCloudServerActionBare(true).
func (c *Connection) RebuildServer(ctx context.Context, serverID, imageID string, options ...CloudServerActionOption) (*CloudServerResult, error) {
	return cloudserver.Rebuild(ctx, c.cloudServerDependencies(), serverID, imageID, options)
}

// CloudServerHostvars is the get_hostvars_from_server result; see
// compute.CloudServerHostvars.
type CloudServerHostvars = compute.CloudServerHostvars

// CloudServerHostvarsFlavor is the flavor name expansion; see
// compute.CloudServerHostvarsFlavor.
type CloudServerHostvarsFlavor = compute.CloudServerHostvarsFlavor

// CloudServerHostvarsImage is the image name expansion; see
// compute.CloudServerHostvarsImage.
type CloudServerHostvarsImage = compute.CloudServerHostvarsImage

// CloudServerHostvarsVolume is one attached volume; see
// compute.CloudServerHostvarsVolume.
type CloudServerHostvarsVolume = compute.CloudServerHostvarsVolume

// CloudServerMeta is the get_server_meta result; see compute.CloudServerMeta.
type CloudServerMeta = compute.CloudServerMeta

// GetOpenStackVars maps get_openstack_vars (meta.get_hostvars_from_server)
// for a supplied server. In Python order it expands the interfaces with
// ExpandServerInterfaces, resolves the flavor name (FindIdentity of the
// flavor ID, else the embedded original_name), lists the server security
// groups (GET servers/{id} and servers/{id}/os-security-groups; failures are
// recorded), resolves the image name with GetImageRecordName, and lists the
// attached volumes with GetVolumes (a missing block-storage endpoint skips
// it; other failures are recorded). Interface, flavor and image failures are
// returned beside the partial result. Nothing is mutated or replayed.
func (c *Connection) GetOpenStackVars(ctx context.Context, server *servers.Server, options ...compute.ServerAddressOption) (*CloudServerHostvars, error) {
	wrap := func(err error) error {
		return request.Wrap("GetOpenStackVars", "compute", cloudread.ContextError(ctx, err))
	}
	if c == nil {
		return nil, wrap(invalid("connection is required"))
	}
	if server == nil {
		return nil, wrap(invalid("server is required"))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	hostvars, err := c.cloudServerHostvars(ctx, server, nil, slices.Clone(options))
	return hostvars, wrap(err)
}

// GetServerMeta maps get_server_meta: the GetOpenStackVars hostvars and the
// inventory groups of meta.get_groups_from_server.
func (c *Connection) GetServerMeta(ctx context.Context, server *servers.Server, options ...compute.ServerAddressOption) (*CloudServerMeta, error) {
	hostvars, err := c.GetOpenStackVars(ctx, server, options...)
	if err != nil {
		if hostvars == nil {
			return nil, err
		}
		return &CloudServerMeta{Hostvars: hostvars}, err
	}
	return &CloudServerMeta{Hostvars: hostvars, Groups: CloudServerGroups(hostvars)}, nil
}

// CloudServerGroups computes meta.get_groups_from_server from hostvars; see
// compute.CloudServerGroups.
func CloudServerGroups(hostvars *CloudServerHostvars) []string {
	return compute.CloudServerGroups(hostvars)
}

func (c *Connection) cloudServerHostvars(ctx context.Context, server *servers.Server, record *servers.ServerFindRecord, address []compute.ServerAddressOption) (*CloudServerHostvars, error) {
	return serverhostvars.Build(ctx, serverhostvars.Dependencies{
		Location:         c.CurrentLocation,
		ExpandInterfaces: c.ExpandServerInterfaces,
		Compute:          c.ComputeV2,
		Image:            c.Image,
		Volumes: func(ctx context.Context, serverID string) ([]*resource.RawResource, error) {
			listed, err := c.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})
			if err != nil {
				return nil, err
			}
			return listed.Volumes, nil
		},
	}, server, record, address)
}

// addressNetworkService returns nil (no Neutron) for a missing endpoint and for
// a network service disabled by ServiceEnabled, the has_service('network')
// half of Python's _use_neutron_floating.
func (c *Connection) addressNetworkService(ctx context.Context) (*network.Service, error) {
	if !c.ServiceEnabled("network") {
		return nil, ctx.Err()
	}
	service, err := c.Network(ctx)
	if err != nil && c.networkLookupUnavailable(ctx, err) {
		return nil, ctx.Err()
	}
	return service, err
}

func (c *Connection) addressComputeClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, err
	}
	if service == nil || service.API == nil || service.Servers == nil || service.RawClient() == nil {
		return nil, invalid("raw compute service is required")
	}
	api, servers, collection, flavors, client := service.API, service.Servers, service.Servers.Collection, service.Flavors, service.RawClient()
	apiServers := api.Servers
	provider := c.provider
	rest.RegisterOperationSource(ctx, func(ctx context.Context) error {
		c.mu.Lock()
		cached := c.compute
		c.mu.Unlock()
		if c.provider != provider || cached != service || service.API != api || service.Servers != servers || servers.Collection != collection || service.Flavors != flavors || service.RawClient() != client || api.Servers != apiServers || api.RawClient() != client {
			return invalid("lazy compute service source changed")
		}
		return nil
	})
	return service.RawClient(), nil
}

func (c *Connection) addressFacade() *compute.Service {
	return compute.New(nil, compute.Dependencies{
		CloudLocation: c.CurrentLocation,
		NetworkRoles:  c.GetNetworkRoles, AddressNetworks: c.addressNetworkService, AddressCompute: c.addressComputeClient,
		NetworkPolicy: c.options.networkRoles, ServerAddresses: c.options.serverAddresses,
	})
}

// GetServerPublicIP calculates IPv4 from a supplied Nova model. It needs no
// Compute endpoint and only discovers network roles when selection needs them.
func (c *Connection) GetServerPublicIP(ctx context.Context, server *compute.Server, options ...compute.ServerAddressOption) (string, error) {
	if c == nil {
		return "", invalid("connection is required")
	}
	return c.addressFacade().GetServerPublicIP(ctx, server, options...)
}

func (c *Connection) GetServerPrivateIP(ctx context.Context, server *compute.Server, options ...compute.ServerAddressOption) (string, error) {
	if c == nil {
		return "", invalid("connection is required")
	}
	return c.addressFacade().GetServerPrivateIP(ctx, server, options...)
}

// ExpandServerInterfaces returns an owned view; shared Network roles are cached
// by Connection while address supplementation is refreshed on every call.
func (c *Connection) ExpandServerInterfaces(ctx context.Context, server *compute.Server, options ...compute.ServerAddressOption) (*compute.ServerAddressView, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().ExpandServerInterfaces(ctx, server, options...)
}

// PlanServerFloatingIP keeps known skips free of endpoint discovery. A nil
// address refresh or Nova supplementation requires a lazy Compute endpoint;
// Neutron classification shares Connection's guarded role snapshot.
func (c *Connection) PlanServerFloatingIP(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.AutomaticFloatingIPOption) (*compute.ServerFloatingIPDecision, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().PlanServerFloatingIP(ctx, input, options...)
}

func (c *Connection) EnsureServerFloatingIP(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.AutomaticFloatingIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().EnsureServerFloatingIP(ctx, input, options...)
}

// AddIPsToServer provides standalone pool/list/automatic assignment with lazy
// services, async attachment and a single 60-second budget by default.
func (c *Connection) AddIPsToServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPsToServer(ctx, input, options...)
}

// AddIPList attaches positional addresses in order. Empty is a no-op and cannot
// fall through to automatic allocation; Compute discovery remains lazy.
func (c *Connection) AddIPList(ctx context.Context, server *compute.Server, addresses []string, options ...compute.ServerIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPList(ctx, server, addresses, options...)
}

// GetActiveServer checks supplied metadata and keeps known branches free of
// endpoint discovery. Needed Neutron assignment obtains only lazy services.
func (c *Connection) GetActiveServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerReadyOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().GetActiveServer(ctx, input, options...)
}

// WaitForServer validates policy before obtaining a raw Compute endpoint. The
// library supplies lazy service bindings and conditional floating-IP behavior.
func (c *Connection) WaitForServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerReadyOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().WaitForServer(ctx, input, options...)
}

// CreateWithAutomaticFloatingIP discovers Compute lazily, then delegates all
// creation, readiness and automatic-address policy to the owned service lane.
func (c *Connection) CreateWithAutomaticFloatingIP(ctx context.Context, request compute.CreateServerRequest, options compute.AutomaticServerCreateOptions) (*compute.AutomaticServerCreateResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if c == nil {
		return nil, invalid("connection is required")
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	return service.CreateWithAutomaticFloatingIP(ctx, request, options)
}

// CreateRefreshedServer is Python's cloud create_server with wait=False on
// the Connection-owned Compute service: POST, one GET, the ERROR fault check
// and interface expansion with Connection-configured address policy.
func (c *Connection) CreateRefreshedServer(ctx context.Context, request compute.CreateServerRequest, serverOptions []compute.CreateServerOption, addressOptions ...compute.ServerAddressOption) (*compute.RefreshedServerCreateResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	if ctx == nil {
		return nil, invalid("context is required")
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, err
	}
	return service.CreateRefreshedServer(ctx, request, serverOptions, addressOptions...)
}

func (c *Connection) ListFloatingIPs(ctx context.Context, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().ListFloatingIPs(ctx, options...)
}

func (c *Connection) SearchFloatingIPs(ctx context.Context, input compute.SearchFloatingIPsRequest, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().SearchFloatingIPs(ctx, input, options...)
}

func (c *Connection) GetFloatingIP(ctx context.Context, input compute.GetFloatingIPRequest, options ...compute.FloatingIPQueryOption) (*compute.GetFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().GetFloatingIP(ctx, input, options...)
}

func (c *Connection) GetFloatingIPByID(ctx context.Context, input compute.GetFloatingIPByIDRequest, options ...compute.FloatingIPQueryOption) (*compute.GetFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().GetFloatingIPByID(ctx, input, options...)
}

func (c *Connection) ListFloatingIPPools(ctx context.Context, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPPoolQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().ListFloatingIPPools(ctx, options...)
}

func (c *Connection) SearchFloatingIPPools(ctx context.Context, input compute.SearchFloatingIPPoolsRequest, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPPoolQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().SearchFloatingIPPools(ctx, input, options...)
}

// AvailableFloatingIP supplies lazy service discovery and configured source
// policy for free-first availability. A Server is optional; this is not Ensure.
func (c *Connection) AvailableFloatingIP(ctx context.Context, input compute.AvailableFloatingIPRequest, options ...compute.AvailableFloatingIPOption) (*compute.AvailableFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AvailableFloatingIP(ctx, input, options...)
}

func (c *Connection) CreateFloatingIP(ctx context.Context, input compute.CreateFloatingIPRequest, options ...compute.FloatingIPCreateOption) (*compute.CreateFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().CreateFloatingIP(ctx, input, options...)
}

func (c *Connection) DeleteFloatingIP(ctx context.Context, input compute.DeleteFloatingIPRequest, options ...compute.FloatingIPDeleteOption) (*compute.DeleteFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DeleteFloatingIP(ctx, input, options...)
}

func (c *Connection) DeleteUnattachedFloatingIPs(ctx context.Context, options ...compute.FloatingIPDeleteOption) (*compute.DeleteUnattachedFloatingIPsResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DeleteUnattachedFloatingIPs(ctx, options...)
}

// DetachIPFromServer implements detach_ip_from_server over lazily discovered
// Network and Compute services. See compute.Service.DetachIPFromServer for the
// branch, error and no-replay contract.
func (c *Connection) DetachIPFromServer(ctx context.Context, input compute.DetachFloatingIPRequest, options ...compute.FloatingIPQueryOption) (*compute.DetachFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DetachIPFromServer(ctx, input, options...)
}

// AddAutoIP implements add_auto_ip over lazily discovered Network and Compute
// services. See compute.Service.AddAutoIP for the reuse, attach, wait and
// timeout-cleanup contract.
func (c *Connection) AddAutoIP(ctx context.Context, server *compute.Server, options ...compute.AddAutoIPOption) (*compute.AddAutoIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddAutoIP(ctx, server, options...)
}

// AddIPFromPool implements the pool branch of add_ips_to_server over lazily
// discovered Network and Compute services. See compute.Service.AddIPFromPool
// for the reuse, refresh, attach and wait contract.
func (c *Connection) AddIPFromPool(ctx context.Context, server *compute.Server, pool resource.Ref, options ...compute.AddIPFromPoolOption) (*compute.AddAutoIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPFromPool(ctx, server, pool, options...)
}

// cloudComputeDependencies supplies the Connection facilities of the cloud
// compute helpers.
func (c *Connection) cloudComputeDependencies() cloudcompute.Dependencies {
	return cloudcompute.Dependencies{
		Guard:         c.recordListContext,
		ComputeV2:     c.ComputeV2,
		Compute:       c.Compute,
		Location:      c.CurrentLocation,
		FindProjectID: c.findQuotaProjectID,
		ProjectLimits: c.ProjectLimits,
	}
}

// UpdateAggregate finds a host aggregate by Nova ID-or-name (strict: a missing
// aggregate is resource.ErrNotFound and no update is sent) and then sends one
// PUT with only the attributes supplied through the update options.
func (c *Connection) UpdateAggregate(ctx context.Context, nameOrID string, options ...aggregates.UpdateRecordOption) (*aggregates.AggregateRecord, error) {
	return cloudcompute.UpdateAggregate(ctx, c.cloudComputeDependencies(), nameOrID, options)
}

// DeleteAggregate deletes a host aggregate and reports whether a delete
// happened. An identity made only of digits is taken as an aggregate ID and
// deleted without any lookup; any other identity is first found by name or ID.
// A missing aggregate, including a 404 on the DELETE itself, returns
// (false, nil). Every other failure is returned and the DELETE is never replayed.
func (c *Connection) DeleteAggregate(ctx context.Context, nameOrID string) (bool, error) {
	return cloudcompute.DeleteAggregate(ctx, c.cloudComputeDependencies(), nameOrID)
}

// CloudAggregateSearchResult carries the ordinary matches and the JSON view the
// filters ran over. Value is the filtered JSON array or, for a JMESPath
// expression, arbitrary output with Expression true and no Aggregates.
type CloudAggregateSearchResult struct {
	Value      json.RawMessage
	Aggregates []*aggregates.Aggregate
	Inventory  []*aggregates.Aggregate
	Expression bool
}

// SearchAggregates maps search_aggregates: list every host aggregate, then apply
// the exact/glob name-or-ID phase and the local dictionary or JMESPath filters.
// No filter is sent to Nova. The filters run over the Source attribute view
// built from each row Nova returned: Python attribute names (deleted as
// is_deleted), null for absent members, the hosts, metadata and is_deleted
// conversions of openstack/fields.py, and the computed location. Only the
// filter options apply; list, bare and address options are rejected before
// HTTP.
func (c *Connection) SearchAggregates(ctx context.Context, nameOrID string, options ...CloudServerReadOption) (*CloudAggregateSearchResult, error) {
	result, err := cloudcompute.SearchAggregates(ctx, c.cloudComputeDependencies(), nameOrID, options)
	return (*CloudAggregateSearchResult)(result), err
}

// GetComputeLimits reads Nova absolute limits. An empty nameOrID reads the
// limits of the authenticated project without a tenant query and without any
// Identity request. A non-empty value is resolved once by Keystone ID-or-name
// lookup (strict) and sent as the tenant_id query. The whole LimitsResource is
// returned; its embedded Absolute data is the part a cloud helper exposes.
func (c *Connection) GetComputeLimits(ctx context.Context, nameOrID string, options ...limits.GetOption) (*limits.LimitsResource, error) {
	return cloudcompute.GetLimits(ctx, c.cloudComputeDependencies(), nameOrID, options)
}

// ComputeUsageRange bounds a tenant usage query. A zero Start or End is not
// sent. Values are converted to UTC before they are formatted.
type ComputeUsageRange struct {
	Start, End time.Time
}

// ParseComputeUsageDate parses an ISO 8601 date for ComputeUsageRange. It
// accepts a date, a date-time with a "T" or space separator, optional
// fractional seconds, and an optional "Z" or numeric offset. A value without an
// offset is taken as UTC. Anything else is rejected with resource.ErrInvalidOption.
func ParseComputeUsageDate(value string) (time.Time, error) {
	return cloudcompute.ParseUsageDate(value)
}

// GetComputeUsage resolves the project by strict Keystone ID-or-name lookup and
// reads its tenant usage with one request: the first value of the usage stream.
// Pagination links are not followed. The date range, when given, is sent as
// UTC start/end query values.
func (c *Connection) GetComputeUsage(ctx context.Context, nameOrID string, dates ComputeUsageRange, options ...usage.SingleTenantOption) (*usage.TenantUsage, error) {
	return cloudcompute.GetUsage(ctx, c.cloudComputeDependencies(), nameOrID, dates.Start, dates.End, options)
}

func (c *Connection) SearchServerGroups(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudSearchResult, error) {
	return cloudcompute.SearchServerGroups(ctx, c.cloudComputeDependencies(), identity, options)
}

func (c *Connection) GetServerGroup(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudGetResult, error) {
	return cloudcompute.GetServerGroup(ctx, c.cloudComputeDependencies(), identity, options)
}

func (c *Connection) GetServerConsole(ctx context.Context, identity string, options ...compute.ServerConsoleOption) (*compute.ServerConsoleResult, error) {
	return cloudcompute.GetServerConsole(ctx, c.cloudComputeDependencies(), identity, options)
}

// AddServerSecurityGroups maps add_server_security_groups for identities. It
// finds the server, resolves every group (Neutron or Nova per the Connection's
// SecurityGroupCloud selection), and only then sends one addSecurityGroup action
// per group, in order, using the group's name or else its ID. A missing server
// or group returns (false, nil) before any action. An empty group list sends
// nothing and returns false, as Python's empty-list guard does. A failing action stops the
// sequence (earlier groups stay added) and no action is replayed.
// When secgroup_source (WithSecurityGroupSource or clouds.yaml) is neither
// neutron nor nova, network.ErrUnavailableSecurityGroups is returned before any
// request, as Python raises OpenStackCloudUnavailableFeature. Passing
// already-resolved group objects is not modelled.
func (c *Connection) AddServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	return cloudserver.AddSecurityGroups(ctx, c.cloudServerDependencies(), server, groups)
}

// RemoveServerSecurityGroups maps remove_server_security_groups with the same
// lookup phase as AddServerSecurityGroups. A 404 from one removeSecurityGroup
// action is tolerated: the remaining groups are still attempted and the result
// is false. Any other failure stops the sequence.
func (c *Connection) RemoveServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	return cloudserver.RemoveSecurityGroups(ctx, c.cloudServerDependencies(), server, groups)
}
