package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CloudServerActionOpts owns the wait, address and floating-IP controls shared
// by DeleteServer and RebuildServer. Timeout defaults to Python's 180 seconds.
type CloudServerActionOpts struct {
	Wait      bool
	Timeout   time.Duration
	Unlimited bool
	// WaitOptions apply after the timeout (for example a poll interval).
	WaitOptions []resource.WaitOption

	// DeleteIPs and DeleteIPRetry map delete_ips and delete_ip_retry
	// (default one retry). FloatingIPQuery and FloatingIPDelete tune the two
	// floating-IP helpers used by DeleteIPs.
	DeleteIPs        bool
	DeleteIPRetry    *int
	FloatingIPQuery  []compute.FloatingIPQueryOption
	FloatingIPDelete []compute.FloatingIPDeleteOption

	// AdminPassword and Rebuild tune RebuildServer; Bare and Address expand
	// the result like the typed getters.
	AdminPassword string
	Rebuild       []servers.RebuildOption
	Bare          bool
	Address       []compute.ServerAddressOption
	// Detailed expands the rebuild result into GetOpenStackVars hostvars.
	Detailed bool
}

// CloudServerActionOption mutates CloudServerActionOpts.
type CloudServerActionOption func(*CloudServerActionOpts) error

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

func applyCloudServerActionOptions(options []CloudServerActionOption) (CloudServerActionOpts, error) {
	cfg := CloudServerActionOpts{Timeout: 180 * time.Second}
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return cfg, fmt.Errorf("%w: nil cloud server option", resource.ErrInvalidOption)
		}
		if err := apply(&cfg); err != nil {
			return cfg, err
		}
	}
	if !cfg.Unlimited && cfg.Timeout <= 0 {
		return cfg, fmt.Errorf("%w: server wait timeout must be positive", resource.ErrInvalidOption)
	}
	return cfg, nil
}

func (cfg CloudServerActionOpts) waitOptions() []resource.WaitOption {
	first := resource.WithTimeout(cfg.Timeout)
	if cfg.Unlimited {
		first = resource.WithUnlimitedWait()
	}
	return append([]resource.WaitOption{first}, cfg.WaitOptions...)
}

// cloudServerFloatingAddresses mirrors meta.find_nova_interfaces(addresses,
// ext_tag='floating'): rows lacking OS-EXT-IPS:type are skipped and the
// original network and row order is kept.
func cloudServerFloatingAddresses(raw json.RawMessage) ([]string, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: server addresses must be an object: %w", resource.ErrInvalidOption, err)
	}
	var out []string
	for _, member := range members {
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(member.Value, &rows); err != nil {
			return nil, fmt.Errorf("%w: addresses of network %q must be a list of objects: %w", resource.ErrInvalidOption, member.Key, err)
		}
		for _, row := range rows {
			tag, ok := row["OS-EXT-IPS:type"]
			if !ok {
				continue
			}
			var kind string
			if json.Unmarshal(tag, &kind) != nil || kind != "floating" {
				continue
			}
			var addr string
			if err := json.Unmarshal(row["addr"], &addr); err != nil {
				return nil, fmt.Errorf("%w: floating address row of network %q has no string addr", resource.ErrInvalidOption, member.Key)
			}
			out = append(out, addr)
		}
	}
	return out, nil
}

func cloudServerRecordString(record *servers.ServerFindRecord, key string) string {
	if record == nil || record.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(record.Resource.Body[key], &text)
	return text
}

func (c *Connection) cloudServerDeleteFloatingIPs(ctx context.Context, record *servers.ServerFindRecord, cfg CloudServerActionOpts) error {
	addresses := record.Resource.Body["addresses"]
	if truthy, err := cloudfilter.PythonJSONTruthy(bytes.Clone(addresses)); err != nil || !truthy {
		return err
	}
	floating, err := cloudServerFloatingAddresses(addresses)
	if err != nil {
		return err
	}
	retries := 1
	if cfg.DeleteIPRetry != nil {
		retries = *cfg.DeleteIPRetry
	}
	serverID := cloudServerRecordString(record, "id")
	for _, address := range floating {
		filter, _ := json.Marshal(map[string]string{"floating_ip_address": address})
		query := append(slices.Clone(cfg.FloatingIPQuery), compute.WithFloatingIPQueryFilters(filter))
		found, err := c.GetFloatingIP(ctx, compute.GetFloatingIPRequest{}, query...)
		if err != nil {
			if errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				continue
			}
			return err
		}
		if found == nil || found.FloatingIP == nil || found.FloatingIP.Resource == nil {
			continue
		}
		var id string
		if err := json.Unmarshal(found.FloatingIP.Resource.Body["id"], &id); err != nil || id == "" {
			return fmt.Errorf("%w: floating IP %s has no string ID", resource.ErrInvalidOption, address)
		}
		deleteOptions := append(slices.Clone(cfg.FloatingIPDelete), compute.WithFloatingIPDeleteRetries(retries))
		deleted, err := c.DeleteFloatingIP(ctx, compute.DeleteFloatingIPRequest{ID: id}, deleteOptions...)
		if err != nil {
			return err
		}
		if deleted == nil || !deleted.Deleted {
			return fmt.Errorf("tried to delete floating IP %s associated with server %s but it was not deleted; the server was not deleted", address, serverID)
		}
	}
	return nil
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
	wrap := func(err error) error {
		return request.Wrap("DeleteServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerActionOptions(options)
	if err != nil {
		return false, wrap(err)
	}
	if err := c.recordListContext(ctx); err != nil {
		return false, wrap(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return false, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return false, wrap(err)
	}
	record, err := service.Servers.FindRecord(ctx, nameOrID, servers.WithRecordFindLocation(location))
	if err != nil {
		return false, wrap(err)
	}
	if record == nil {
		return false, nil
	}
	id := cloudServerRecordString(record, "id")
	if id == "" {
		return false, wrap(fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption))
	}
	// _has_floating_ips: floating_ip_source none skips the floating cleanup.
	if cfg.DeleteIPs && c.options.serverAddresses.FloatingIPSource() != compute.FloatingIPNone {
		if err := c.cloudServerDeleteFloatingIPs(ctx, record, cfg); err != nil {
			return false, wrap(err)
		}
	}
	if err := service.Servers.Delete(ctx, id); err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return false, nil
		}
		return false, wrap(err)
	}
	if !cfg.Wait {
		return true, nil
	}
	return true, wrap(service.Servers.WaitForDelete(ctx, resource.ID(id), cfg.waitOptions()...))
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
	wrap := func(err error) error {
		return request.Wrap("RebuildServer", "compute", cloudread.ContextError(ctx, err))
	}
	cfg, err := applyCloudServerActionOptions(options)
	if err != nil {
		return nil, wrap(err)
	}
	if serverID == "" || imageID == "" {
		return nil, wrap(fmt.Errorf("%w: server ID and image ID are required", resource.ErrInvalidOption))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	opts := servers.RebuildOpts{ImageRef: imageID, AdminPass: cfg.AdminPassword}
	rebuilt, err := service.Servers.Rebuild(ctx, serverID, opts, cfg.Rebuild...)
	if err != nil {
		return nil, wrap(err)
	}
	result := &CloudServerResult{Server: rebuilt, Sent: true, AdminPassword: rebuilt.AdminPass}
	read := CloudServerReadOpts{Bare: cfg.Bare, Address: cfg.Address, Detailed: cfg.Detailed}
	if cfg.Wait {
		password := rebuilt.AdminPass
		if password == "" {
			password = cfg.AdminPassword
		}
		waitID := rebuilt.ID
		if waitID == "" {
			waitID = serverID
		}
		waited, err := service.Servers.WaitForServer(ctx, resource.ID(waitID), cfg.waitOptions()...)
		if err != nil {
			return result, wrap(err)
		}
		if waited.Status == "ACTIVE" {
			waited.AdminPass = password
		}
		result.Server, result.AdminPassword = waited, password
	}
	return result, wrap(c.cloudServerExpand(ctx, result, read))
}
