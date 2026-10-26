package projectcleanup

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
)

// serviceCleanup is one service's self-contained cleanup unit. An
// implementation lives in its own file (for example compute.go) and is
// registered in implementations below.
type serviceCleanup interface {
	// kinds declares every kind the service can put in a plan. Each kind
	// must also appear in stageOrder.
	kinds() []kindSpec
	// discover lists the project's resources with d.Client() and records
	// candidates and exclusions through d. It must not mutate anything.
	// Services run in serviceOrder, so d.Candidate can see candidates of
	// earlier services (for example compute servers for network ports).
	discover(ctx context.Context, d *discovery) error
}

// kindSpec describes how one kind is filtered, deleted and observed.
type kindSpec struct {
	kind string
	// createdField and updatedField are the wire fields behind Python's
	// created_at and updated_at attributes; empty when the Python resource
	// has no such attribute, so a time filter excludes the kind.
	createdField, updatedField string
	// wait makes Execute observe absence before the next stage.
	wait bool
	// remove sends exactly one mutation. A 404 must surface as an error
	// for which errors.Is(err, resource.ErrNotFound) is true.
	remove func(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error
	// absent polls until the resource is gone, within timeout and ctx.
	// Required when wait is set.
	absent func(ctx context.Context, client *gophercloud.ServiceClient, c Candidate, timeout, interval time.Duration) error
}

// implementations maps a Python service name to its registered cleanup.
// A core service without an entry is reported as NotYetSupported.
var implementations = map[string]serviceCleanup{
	"block_storage": blockStorageCleanup{},
	"compute":       computeCleanup{},
	"image":         imageCleanup{},
	"key_manager":   keyManagerCleanup{},
	"network":       networkCleanup{},
	"object_store":  objectStoreCleanup{},
}

// serviceInfo is a service that Python's project_cleanup visits.
type serviceInfo struct {
	name string
	// types are catalog types: the official type first, then aliases.
	types []string
	// core services are phase 1; others are reported only when present.
	core bool
}

// serviceOrder is the discovery order, following Python's dependency graph
// (compute before block storage, network and image).
var serviceOrder = []serviceInfo{
	{"compute", []string{"compute"}, true},
	{"block_storage", []string{"block-storage", "volumev3", "volumev2", "volume", "block-store"}, true},
	{"network", []string{"network"}, true},
	{"image", []string{"image"}, true},
	{"object_store", []string{"object-store"}, true},
	{"key_manager", []string{"key-manager"}, true},
	{"orchestration", []string{"orchestration"}, false},
	{"load_balancer", []string{"load-balancer"}, false},
	{"dns", []string{"dns"}, false},
}

// stageOrder fixes the execution order of all phase-1 kinds. Kinds of
// services that are not implemented yet are listed so that the order is
// decided in one place.
var stageOrder = []string{
	"compute.server",
	"compute.server_group",
	"block_storage.backup",
	"block_storage.snapshot",
	"block_storage.volume",
	"network.vpn_ipsec_site_connection",
	"network.vpn_service",
	"network.vpn_endpoint_group",
	"network.vpn_ike_policy",
	"network.vpn_ipsec_policy",
	"network.floating_ip",
	"network.security_group",
	"network.router_routes",
	"network.router_interface",
	"network.router",
	"network.port",
	"network.subnet",
	"network.network",
	"image.image",
	"object_store.object",
	"object_store.container",
	"key_manager.container",
	"key_manager.secret",
}

// skipNames are all names that Python's skip_resources understands.
var skipNames = []string{
	"compute.server",
	"block_storage.backup", "block_storage.snapshot", "block_storage.volume",
	"network.vpn_ipsec_site_connection", "network.vpn_service", "network.vpn_endpoint_group",
	"network.vpn_ike_policy", "network.vpn_ipsec_policy", "network.floating_ip",
	"network.security_group", "network.network", "network.router", "network.port", "network.subnet",
	"image.image",
	"object_store.container", "object_store.object",
	"key_manager.container", "key_manager.secret",
	"orchestration.stack", "load_balancer.load_balancer", "dns.zone", "dns.floating_ip",
}
