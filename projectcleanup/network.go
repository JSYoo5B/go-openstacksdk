package projectcleanup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// networkCleanup follows openstack/network/v2/_proxy.py _service_cleanup:
// VPN resources (only when a VPN service provider is configured), floating
// IPs, security groups except "default", then networks whose ports are
// unused together with their router interfaces, routers, unowned ports and
// subnets, and finally routers without ports. Neutron deletes are
// synchronous, so no stage waits, as in Python.
type networkCleanup struct{}

// Device owners of router interface ports, as listed by Python.
var routerInterfaceOwners = []string{
	"network:router_interface",
	"network:router_interface_distributed",
	"network:ha_router_replicated_interface",
}

// Router-owned ports that Neutron removes itself when the router is deleted
// or its interfaces are removed.
var routerManagedOwners = []string{
	"network:router_gateway",
	"network:router_ha_interface",
	"network:router_centralized_snat",
}

func (networkCleanup) kinds() []kindSpec {
	// Python's VPN resources have no created_at or updated_at attribute.
	return []kindSpec{
		{kind: "vpn_ipsec_site_connection", remove: restRemove("vpn/ipsec-site-connections", "vpn_ipsec_site_connection", 204)},
		{kind: "vpn_service", remove: restRemove("vpn/vpnservices", "vpn_service", 204)},
		{kind: "vpn_endpoint_group", remove: restRemove("vpn/endpoint-groups", "vpn_endpoint_group", 204)},
		{kind: "vpn_ike_policy", remove: restRemove("vpn/ikepolicies", "vpn_ike_policy", 204)},
		{kind: "vpn_ipsec_policy", remove: restRemove("vpn/ipsecpolicies", "vpn_ipsec_policy", 204)},
		{kind: "floating_ip", createdField: "created_at", updatedField: "updated_at", remove: restRemove("floatingips", "floating_ip", 204)},
		{kind: "security_group", createdField: "created_at", updatedField: "updated_at", remove: restRemove("security-groups", "security_group", 204)},
		{kind: "router_routes", remove: removeRouterRoutes},
		{kind: "router_interface", remove: removeRouterInterface},
		{kind: "router", createdField: "created_at", updatedField: "updated_at", remove: restRemove("routers", "router", 204)},
		{kind: "port", createdField: "created_at", updatedField: "updated_at", remove: restRemove("ports", "port", 204)},
		{kind: "subnet", createdField: "created_at", updatedField: "updated_at", remove: restRemove("subnets", "subnet", 204)},
		{kind: "network", createdField: "created_at", updatedField: "updated_at", remove: restRemove("networks", "network", 204)},
	}
}

// removeRouterRoutes sends Python's remove_extra_routes_from_router with the
// routes recorded in the plan's router row.
func removeRouterRoutes(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
	var router struct {
		Routes json.RawMessage `json:"routes"`
	}
	if err := json.Unmarshal(c.Raw, &router); err != nil || len(router.Routes) == 0 {
		return fmt.Errorf("%w: router %s has no recorded routes", resource.ErrInvalidOption, c.ID)
	}
	body := map[string]any{"router": map[string]any{"routes": router.Routes}}
	_, err := rest.DoJSON(ctx, client, "PUT", client.ServiceURL("routers", c.ID, "remove_extraroutes"), body, nil, 200)
	return notFound(err, "router", c.ID)
}

// removeRouterInterface sends Python's remove_interface_from_router with the
// interface port (Ref.ID) of the router (Ref.Parent).
func removeRouterInterface(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
	body := map[string]any{"port_id": c.ID}
	_, err := rest.DoJSON(ctx, client, "PUT", client.ServiceURL("routers", c.Parent, "remove_router_interface"), body, nil, 200)
	return notFound(err, "router interface", c.Parent+"/"+c.ID)
}

// notFound makes a 404 match resource.ErrNotFound while keeping the response.
func notFound(err error, kind, id string) error {
	if err != nil && gophercloud.ResponseCodeIs(err, 404) {
		return fmt.Errorf("%w: %s %s: %w", resource.ErrNotFound, kind, id, err)
	}
	return err
}

// networkDiscovery carries the state of one network discovery.
type networkDiscovery struct {
	d       *discovery
	project string
	// timeFilters is Python's "filters is not None".
	timeFilters bool
	routers     map[string]*row
	planned     map[Ref]bool
	touched     []string
	handled     map[string]bool
}

func (networkCleanup) discover(ctx context.Context, d *discovery) error {
	n := &networkDiscovery{d: d, project: d.Project().ID, timeFilters: d.options.createdBefore != nil || d.options.updatedBefore != nil,
		routers: map[string]*row{}, planned: map[Ref]bool{}, handled: map[string]bool{}}
	if err := n.vpn(ctx); err != nil {
		return err
	}
	if !d.Skipped("floating_ip") {
		if err := n.floatingIPs(ctx); err != nil {
			return err
		}
	}
	if !d.Skipped("security_group") {
		if err := n.securityGroups(ctx); err != nil {
			return err
		}
	}
	// Python cleans networks, routers, ports and subnets together, or not at all.
	if !d.Skipped("network") && !d.Skipped("router") && !d.Skipped("port") && !d.Skipped("subnet") {
		if err := n.networks(ctx); err != nil {
			return err
		}
	}
	if !d.Skipped("router") {
		return n.unattachedRouters(ctx)
	}
	return nil
}

func (n *networkDiscovery) query(pairs ...string) url.Values {
	query := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		query.Set(pairs[i], pairs[i+1])
	}
	return query
}

func (n *networkDiscovery) add(c Candidate, reason string, depends ...Ref) {
	c.Reason = reason
	c.DependsOn = append(c.DependsOn, depends...)
	n.planned[Ref{Service: "network", Kind: c.Kind, ID: c.ID, Parent: c.Parent}] = true
	n.d.Add(c, 0)
}

// list reads a project-scoped collection and drops rows without a usable
// ID or owned by another project, recording them as exclusions.
func (n *networkDiscovery) list(ctx context.Context, kind, path, plural string, query url.Values) ([]row, error) {
	var rows []row
	for r, err := range listRows(ctx, n.d.Client(), path, plural, query) {
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", plural, err)
		}
		id, name, ok := rowIdentity(n.d, kind, r)
		if !ok {
			continue
		}
		if reason, foreign := ownerMismatch(r, n.project, "project_id", "tenant_id"); foreign {
			n.d.Exclude(Ref{Kind: kind, ID: id}, name, reason, r.raw)
			continue
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func identity(r row) (string, string) {
	id, _ := r.str("id")
	name, _ := r.str("name")
	return id, name
}

type vpnKind struct {
	kind, path, plural string
	// fields of a site connection that reference this kind.
	refs []string
}

var vpnKinds = []vpnKind{
	{"vpn_service", "vpn/vpnservices", "vpnservices", []string{"vpnservice_id"}},
	{"vpn_endpoint_group", "vpn/endpoint-groups", "endpoint_groups", []string{"local_ep_group_id", "peer_ep_group_id"}},
	{"vpn_ike_policy", "vpn/ikepolicies", "ikepolicies", []string{"ikepolicy_id"}},
	{"vpn_ipsec_policy", "vpn/ipsecpolicies", "ipsecpolicies", []string{"ipsecpolicy_id"}},
}

// vpn cleans VPN resources only when a VPN service provider is configured.
// Site connections go first; a service, endpoint group or policy that a
// remaining site connection still uses is excluded, and one used only by
// site connection candidates depends on them.
func (n *networkDiscovery) vpn(ctx context.Context) error {
	providers := 0
	for _, err := range listRows(ctx, n.d.Client(), "service-providers", "service_providers", n.query("service_type", "VPN")) {
		if err != nil {
			return fmt.Errorf("list service providers: %w", err)
		}
		providers++
	}
	if providers == 0 {
		return nil
	}
	pending := !n.d.Skipped("vpn_ipsec_site_connection")
	for _, k := range vpnKinds {
		pending = pending || !n.d.Skipped(k.kind)
	}
	if !pending {
		return nil
	}
	// Every listed connection counts as a user, including rows that are
	// excluded here (for example rows of another owner).
	var connections []row
	for r, err := range listRows(ctx, n.d.Client(), "vpn/ipsec-site-connections", "ipsec_site_connections", n.query("project_id", n.project)) {
		if err != nil {
			return fmt.Errorf("list ipsec site connections: %w", err)
		}
		connections = append(connections, r)
	}
	candidate := map[string]bool{}
	if !n.d.Skipped("vpn_ipsec_site_connection") {
		for _, r := range connections {
			id, name, ok := rowIdentity(n.d, "vpn_ipsec_site_connection", r)
			if !ok {
				continue
			}
			if reason, foreign := ownerMismatch(r, n.project, "project_id", "tenant_id"); foreign {
				n.d.Exclude(Ref{Kind: "vpn_ipsec_site_connection", ID: id}, name, reason, r.raw)
				continue
			}
			c := n.d.Row("vpn_ipsec_site_connection", r, id, name)
			keep, err := n.d.Evaluate(c)
			if err != nil {
				return err
			}
			if keep {
				candidate[id] = true
				n.add(c, selectionReason(n.d))
			}
		}
	}
	for _, k := range vpnKinds {
		if n.d.Skipped(k.kind) {
			continue
		}
		rows, err := n.list(ctx, k.kind, k.path, k.plural, n.query("project_id", n.project))
		if err != nil {
			return err
		}
		for _, r := range rows {
			id, name := identity(r)
			var users, kept []string
			for _, conn := range connections {
				for _, field := range k.refs {
					if ref, _ := conn.str(field); ref == id {
						connID, _ := conn.str("id")
						if candidate[connID] {
							users = append(users, connID)
						} else {
							kept = append(kept, connID)
						}
						break
					}
				}
			}
			if len(kept) > 0 {
				n.d.Exclude(Ref{Kind: k.kind, ID: id}, name, "used by ipsec site connections that are not cleanup candidates: "+strings.Join(kept, ", "), r.raw)
				continue
			}
			c := n.d.Row(k.kind, r, id, name)
			keep, err := n.d.Evaluate(c)
			if err != nil {
				return err
			}
			if !keep {
				continue
			}
			var depends []Ref
			for _, user := range users {
				depends = append(depends, Ref{Service: "network", Kind: "vpn_ipsec_site_connection", ID: user})
			}
			n.add(c, selectionReason(n.d), depends...)
		}
	}
	return nil
}

// floatingIPs follows Python's fip_cleanup_evaluation, which replaces the
// time filters: without filters every floating IP is a candidate; with
// filters an attached floating IP is kept and a detached one is a candidate
// whatever its times. A selector is still consulted. A floating IP whose
// PTR record is a dns candidate depends on it, so the floating IP is not
// deleted when unsetting its PTR record did not succeed.
func (n *networkDiscovery) floatingIPs(ctx context.Context) error {
	rows, err := n.list(ctx, "floating_ip", "floatingips", "floatingips", n.query("project_id", n.project))
	if err != nil {
		return err
	}
	ptrs := dnsPTRDependencies(n.d)
	for _, r := range rows {
		id, name := identity(r)
		if name == "" {
			name, _ = r.str("floating_ip_address")
		}
		port, attached := r.str("port_id")
		attached = attached && port != ""
		if attached && n.timeFilters {
			n.d.Exclude(Ref{Kind: "floating_ip", ID: id}, name, "attached to port "+port+"; with time filters attached floating IPs are kept", r.raw)
			continue
		}
		c := n.d.Row("floating_ip", r, id, name)
		keep, err := n.selector(c)
		if err != nil {
			return err
		}
		if !keep {
			continue
		}
		reason := "all floating IPs (time filters do not apply)"
		if n.timeFilters {
			reason = "not attached to a port (time filters do not apply)"
		}
		if n.d.options.selector != nil {
			reason += ", accepted by selector"
		}
		n.add(c, reason, ptrs[id]...)
	}
	return nil
}

// selector applies only the selector, as Evaluate does after its filters.
func (n *networkDiscovery) selector(c Candidate) (bool, error) {
	if n.d.options.selector == nil {
		return true, nil
	}
	c.Service = n.d.service
	keep, err := n.d.options.selector(c.Clone())
	if err != nil {
		return false, &selectorError{fmt.Errorf("project cleanup selector for %s: %w", c.Ref, err)}
	}
	if !keep {
		n.d.Exclude(c.Ref, c.Name, "rejected by selector", c.Raw)
	}
	return keep, nil
}

func (n *networkDiscovery) securityGroups(ctx context.Context) error {
	rows, err := n.list(ctx, "security_group", "security-groups", "security_groups", n.query("project_id", n.project))
	if err != nil {
		return err
	}
	for _, r := range rows {
		id, name := identity(r)
		if name == "default" {
			n.d.Exclude(Ref{Kind: "security_group", ID: id}, name, "default security group", r.raw)
			continue
		}
		c := n.d.Row("security_group", r, id, name)
		keep, err := n.d.Evaluate(c)
		if err != nil {
			return err
		}
		if keep {
			n.add(c, selectionReason(n.d))
		}
	}
	return nil
}

func boolField(r row, key string) bool {
	var value bool
	return json.Unmarshal(r.fields[key], &value) == nil && value
}

// router reads a router once. A missing router is nil.
func (n *networkDiscovery) router(ctx context.Context, id string) (*row, error) {
	if r, ok := n.routers[id]; ok {
		return r, nil
	}
	value, err := rest.Collection(restSpec(n.d.Client(), "routers", "router", "router", "routers")).Get(ctx, id)
	if errors.Is(err, resource.ErrNotFound) {
		n.routers[id] = nil
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get router %s: %w", id, err)
	}
	raw, err := json.Marshal(value.Body)
	if err != nil {
		return nil, err
	}
	r := &row{fields: value.Body, raw: raw}
	n.routers[id] = r
	return r, nil
}

func hasRoutes(r row) bool {
	var routes []json.RawMessage
	return json.Unmarshal(r.fields["routes"], &routes) == nil && len(routes) > 0
}

// networks follows Python's network loop. A network is examined only when
// none of its listed ports is in use by something that is not being
// cleaned: DHCP and unowned ports do not count, router interfaces are
// removed, ports of compute server candidates follow their server, and the
// Octavia VIP port of a load balancer candidate goes with the load balancer.
func (n *networkDiscovery) networks(ctx context.Context) error {
	nets, err := n.list(ctx, "network", "networks", "networks", n.query("project_id", n.project))
	if err != nil {
		return err
	}
	for _, net := range nets {
		netID, netName := identity(net)
		netRef := Ref{Kind: "network", ID: netID}
		if boolField(net, "shared") {
			n.d.Exclude(netRef, netName, "network is shared", net.raw)
			continue
		}
		if boolField(net, "router:external") {
			n.d.Exclude(netRef, netName, "network is external", net.raw)
			continue
		}
		var interfaces, unowned, serverPorts []row
		var inUse []string
		// Load balancer candidates whose Octavia VIP ports are on the network.
		var loadBalancers []Ref
		for port, err := range listRows(ctx, n.d.Client(), "ports", "ports", n.query("project_id", n.project, "network_id", netID)) {
			if err != nil {
				return fmt.Errorf("list ports of network %s: %w", netID, err)
			}
			portID, ok := port.str("id")
			if !ok || resource.ID(portID).Validate() != nil {
				inUse = append(inUse, "a port without a usable id")
				continue
			}
			if reason, foreign := ownerMismatch(port, n.project, "project_id", "tenant_id"); foreign {
				inUse = append(inUse, fmt.Sprintf("%s (%s)", portID, reason))
				continue
			}
			owner, _ := port.str("device_owner")
			device, _ := port.str("device_id")
			switch {
			case slices.Contains(routerInterfaceOwners, owner):
				interfaces = append(interfaces, port)
			case owner == "network:dhcp":
			case owner == "":
				unowned = append(unowned, port)
			default:
				if _, ok := n.d.Candidate(Ref{Service: "compute", Kind: "server", ID: device}); ok && device != "" {
					serverPorts = append(serverPorts, port)
				} else if lb, ok := octaviaLoadBalancer(n.d, owner, device); ok {
					if !slices.Contains(loadBalancers, lb) {
						loadBalancers = append(loadBalancers, lb)
					}
				} else {
					inUse = append(inUse, fmt.Sprintf("%s (%s %s)", portID, owner, device))
				}
			}
		}
		if len(inUse) > 0 {
			n.d.Exclude(netRef, netName, "network has ports in use by resources that are not cleanup candidates: "+strings.Join(inUse, ", "), net.raw)
			continue
		}
		c := n.d.Row("network", net, netID, netName)
		keep, err := n.d.Evaluate(c)
		if err != nil {
			return err
		}
		if !keep {
			continue
		}
		// Read every router first, so that nothing of this network is
		// planned when one of its routers cannot be cleaned.
		var blocked []string
		for _, port := range interfaces {
			portID, _ := port.str("id")
			device, _ := port.str("device_id")
			router, err := n.router(ctx, device)
			if err != nil {
				return err
			}
			if router == nil {
				blocked = append(blocked, fmt.Sprintf("router %s of interface port %s was not found", device, portID))
			} else if reason, foreign := ownerMismatch(*router, n.project, "project_id", "tenant_id"); foreign {
				blocked = append(blocked, fmt.Sprintf("router %s of interface port %s is %s", device, portID, reason))
			}
		}
		if len(blocked) > 0 {
			n.d.Exclude(netRef, netName, strings.Join(blocked, "; "), net.raw)
			continue
		}
		subnets, err := n.list(ctx, "subnet", "subnets", "subnets", n.query("project_id", n.project, "network_id", netID))
		if err != nil {
			return err
		}
		reason := "network " + netID + " is cleaned"
		var netDepends []Ref
		for _, port := range interfaces {
			portID, portName := identity(port)
			device, _ := port.str("device_id")
			router := *n.routers[device]
			_, routerName := identity(router)
			routesRef := Ref{Service: "network", Kind: "router_routes", ID: device}
			var depends []Ref
			if hasRoutes(router) {
				if !n.planned[routesRef] {
					rc := Candidate{Ref: routesRef, Name: routerName, Raw: router.raw}
					n.add(rc, "routes block removal of router interfaces of "+reason)
				}
				depends = append(depends, routesRef)
			}
			ic := n.d.Row("router_interface", port, portID, portName)
			ic.CreatedAt, ic.UpdatedAt = nil, nil
			ic.Parent = device
			n.add(ic, "router interface on "+reason, depends...)
			if !slices.Contains(n.touched, device) {
				n.touched = append(n.touched, device)
			}
			netDepends = append(netDepends, ic.Ref)
		}
		for _, port := range unowned {
			portID, portName := identity(port)
			pc := n.d.Row("port", port, portID, portName)
			n.add(pc, "port without device owner on "+reason)
			netDepends = append(netDepends, pc.Ref)
		}
		for _, port := range serverPorts {
			portID, portName := identity(port)
			device, _ := port.str("device_id")
			pc := n.d.Row("port", port, portID, portName)
			n.add(pc, "port of server candidate "+device+" on "+reason, Ref{Service: "compute", Kind: "server", ID: device})
			netDepends = append(netDepends, pc.Ref)
		}
		// Octavia deletes its VIP port while deleting the load balancer, so
		// the port is not planned; its subnets and the network wait for the
		// load balancer instead.
		netDepends = append(netDepends, loadBalancers...)
		subnetRefs := slices.Clone(netDepends)
		for _, subnet := range subnets {
			subnetID, subnetName := identity(subnet)
			sc := n.d.Row("subnet", subnet, subnetID, subnetName)
			n.add(sc, "subnet of "+reason, subnetRefs...)
			netDepends = append(netDepends, sc.Ref)
		}
		c.Reason = selectionReason(n.d)
		n.add(c, c.Reason, netDepends...)
	}
	return n.disconnectedRouters(ctx)
}

// octaviaOwner is the device_owner Octavia sets on the VIP ports it
// creates; their device_id is "lb-" followed by the load balancer ID.
const octaviaOwner = "Octavia"

// octaviaLoadBalancer returns the load balancer candidate whose Octavia VIP
// port has this device_owner and device_id.
func octaviaLoadBalancer(d *discovery, owner, device string) (Ref, bool) {
	id, ok := strings.CutPrefix(device, "lb-")
	if owner != octaviaOwner || !ok || id == "" {
		return Ref{}, false
	}
	ref := Ref{Service: "load_balancer", Kind: "load_balancer", ID: id}
	if _, ok := d.Candidate(ref); !ok {
		return Ref{}, false
	}
	return ref, true
}

// routerPorts lists the ports of a router without a project filter, as
// Python's ports(device_id=...) does; extra visible ports only keep it.
func (n *networkDiscovery) routerPorts(ctx context.Context, id string) ([]row, error) {
	var ports []row
	for r, err := range listRows(ctx, n.d.Client(), "ports", "ports", n.query("device_id", id)) {
		if err != nil {
			return nil, fmt.Errorf("list ports of router %s: %w", id, err)
		}
		ports = append(ports, r)
	}
	return ports, nil
}

// disconnectedRouters plans the routers whose interfaces are all removed.
// Python deletes them without evaluating filters; a router that keeps other
// ports is excluded instead of sending a DELETE that Neutron would reject.
func (n *networkDiscovery) disconnectedRouters(ctx context.Context) error {
	for _, id := range n.touched {
		router := *n.routers[id]
		_, name := identity(router)
		ports, err := n.routerPorts(ctx, id)
		if err != nil {
			return err
		}
		var depends []Ref
		var kept []string
		if routes := (Ref{Service: "network", Kind: "router_routes", ID: id}); n.planned[routes] {
			depends = append(depends, routes)
		}
		for _, port := range ports {
			portID, _ := port.str("id")
			owner, _ := port.str("device_owner")
			ref := Ref{Service: "network", Kind: "router_interface", ID: portID, Parent: id}
			switch {
			case n.planned[ref]:
				depends = append(depends, ref)
			case slices.Contains(routerManagedOwners, owner):
			default:
				kept = append(kept, fmt.Sprintf("%s (%s)", portID, owner))
			}
		}
		n.handled[id] = true
		if len(kept) > 0 {
			n.d.Exclude(Ref{Kind: "router", ID: id}, name, "router keeps ports that are not removed by cleanup: "+strings.Join(kept, ", "), router.raw)
			continue
		}
		c := n.d.Row("router", router, id, name)
		n.add(c, "all router interfaces are removed (filters do not apply)", depends...)
	}
	return nil
}

// unattachedRouters follows Python's last loop: a router without any port
// is deleted without evaluating filters.
func (n *networkDiscovery) unattachedRouters(ctx context.Context) error {
	routers, err := n.list(ctx, "router", "routers", "routers", n.query("project_id", n.project))
	if err != nil {
		return err
	}
	for _, router := range routers {
		id, name := identity(router)
		if n.handled[id] {
			continue
		}
		ports, err := n.routerPorts(ctx, id)
		if err != nil {
			return err
		}
		if len(ports) > 0 {
			n.d.Exclude(Ref{Kind: "router", ID: id}, name, fmt.Sprintf("router has %d ports", len(ports)), router.raw)
			continue
		}
		c := n.d.Row("router", router, id, name)
		n.add(c, "router has no ports (filters do not apply)")
	}
	return nil
}
