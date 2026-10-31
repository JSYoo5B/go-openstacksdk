package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudserver"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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
		Guard:            c.recordListContext,
		Compute:          c.ComputeV2,
		Location:         c.CurrentLocation,
		ExpandInterfaces: c.ExpandServerInterfaces,
		Hostvars:         c.cloudServerHostvars,
		FloatingIPSource: func() compute.FloatingIPSource { return c.options.serverAddresses.FloatingIPSource() },
		GetFloatingIP:    c.GetFloatingIP,
		DeleteFloatingIP: c.DeleteFloatingIP,
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
