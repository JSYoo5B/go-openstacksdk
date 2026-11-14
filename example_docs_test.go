package openstack_test

import (
	"context"
	"fmt"
	networkcloud "github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
	"log"
	"net/url"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	blockstoragev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	storagev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	dnsv2 "github.com/JSYoo5B/go-openstacksdk/dns/v2"
	identityv3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	imagev2 "github.com/JSYoo5B/go-openstacksdk/image/v2"
	lbv2 "github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func FindApplicationResourcesFindingIdentities1(ctx context.Context, compute *computev2.Service,
	storage *storagev3.Service, network *networkv2.Service) error {
	strict := resource.WithIdentityFindIgnoreMissing(false)
	server, err := compute.Servers.FindIdentity(ctx, "web-01", strict)
	if err != nil {
		return err
	}
	volume, err := storage.Volumes.FindIdentity(ctx, "data-01", strict)
	if err != nil {
		return err
	}
	port, err := network.Ports.FindIdentity(ctx, "web-port", strict)
	if err != nil {
		return err
	}
	_, _, _ = server, volume, port
	return nil
}

func FindScopedResourcesFindingIdentities2(ctx context.Context, dns *dnsv2.Service,
	loadbalancer *lbv2.Service) error {
	records, err := dns.RecordSets.InZone(ctx, resource.Name("example.org."))
	if err != nil {
		return err
	}
	record, err := records.FindIdentity(ctx, "www.example.org.",
		resource.WithIdentityFindIgnoreMissing(false))
	if err != nil {
		return err
	}
	members, err := loadbalancer.Pools.Members(ctx, resource.ID("pool-id"))
	if err != nil {
		return err
	}
	member, err := members.FindIdentity(ctx, "backend-01",
		resource.WithIdentityFindIgnoreMissing(false))
	if err != nil {
		return err
	}
	_, _ = record, member
	return nil
}

func FindAcrossProjectsFindingIdentities3(ctx context.Context, compute *computev2.Service,
	storage *blockstoragev3.Service) error {
	summary := resource.WithIdentityFindDetails(false)
	allProjects := resource.WithIdentityFindAllProjects(true)
	strict := resource.WithIdentityFindIgnoreMissing(false)
	server, err := compute.Servers.FindIdentity(ctx, "web-01", summary, allProjects, strict)
	if err != nil {
		return err
	}
	volume, err := storage.Volumes.FindIdentity(ctx, "data-01", summary, allProjects, strict)
	if err != nil {
		return err
	}
	_, _ = server, volume
	return nil
}

func FindFlavorFindingIdentities4(ctx context.Context, compute *computev2.Service) error {
	flavor, err := compute.Flavors.FindIdentity(ctx, "small",
		resource.WithIdentityFindExtraSpecs(true),
		resource.WithIdentityFindIgnoreMissing(false),
		resource.WithIdentityFindQuery("minRam", "256"))
	if err != nil {
		return err
	}
	_, _ = flavor.ID, flavor.ExtraSpecs
	return nil
}

func FindImageFindingIdentities5(ctx context.Context, images *imagev2.Service) error {
	image, err := images.Images.FindIdentity(ctx, "ubuntu",
		resource.WithIdentityFindIgnoreMissing(false))
	if err != nil {
		return err
	}
	_, _ = image.ID, image.Hidden
	return nil
}

func FindTenantResourcesFindingIdentities6(ctx context.Context, identity *identityv3.Service,
	network *networkv2.Service) error {
	strict := resource.WithIdentityFindIgnoreMissing(false)
	domain, err := identity.Domains.FindIdentity(ctx, "Default", strict)
	if err != nil {
		return err
	}
	domainQuery := resource.WithIdentityFindQuery("domain_id", domain.ID)
	project, err := identity.Projects.FindIdentity(ctx, "app", strict, domainQuery)
	if err != nil {
		return err
	}
	user, err := identity.Users.FindIdentity(ctx, "app-user", strict, domainQuery)
	if err != nil {
		return err
	}
	group, err := identity.Groups.FindIdentity(ctx, "app-group", strict, domainQuery)
	if err != nil {
		return err
	}
	role, err := identity.Roles.FindIdentity(ctx, "reader", strict, domainQuery)
	if err != nil {
		return err
	}
	net, err := network.Networks.FindIdentity(ctx, "web-net", strict)
	if err != nil {
		return err
	}
	subnet, err := network.Subnets.FindIdentity(ctx, "web-subnet", strict,
		resource.WithIdentityFindQuery("network_id", net.ID))
	if err != nil {
		return err
	}
	_, _, _, _, _ = project, user, group, role, subnet
	return nil
}

func FindNetworkExtensionsFindingIdentities7(ctx context.Context, network *networkv2.Service,
	projectID string) error {
	filter := resource.WithIdentityFindOptions(resource.IdentityFindOpts{
		Query: url.Values{"project_id": {projectID}, "fields": {"id", "name"}},
	})
	strict := resource.WithIdentityFindIgnoreMissing(false)
	router, err := network.Routers.FindIdentity(ctx, "app-router", filter, strict)
	if err != nil {
		return err
	}
	group, err := network.SecurityGroups.FindIdentity(ctx, "app-sg", filter, strict)
	if err != nil {
		return err
	}
	_, _ = router, group
	return nil
}

func FindSubnetPoolAndTrunkFindingIdentities8(ctx context.Context, network *networkv2.Service,
	projectID string) error {
	filter := resource.WithIdentityFindQuery("project_id", projectID)
	strict := resource.WithIdentityFindIgnoreMissing(false)
	pool, err := network.SubnetPools.FindIdentity(ctx, "app-pool", filter, strict)
	if err != nil {
		return err
	}
	trunk, err := network.Trunks.FindIdentity(ctx, "app-trunk", filter, strict)
	if err != nil {
		return err
	}
	_, _ = pool.Prefixes, trunk.Subports
	return nil
}

func FindQoSAndAddressesFindingIdentities9(ctx context.Context, network *networkv2.Service,
	projectID string) error {
	strict := resource.WithIdentityFindIgnoreMissing(false)
	policy, err := network.QoSPolicies.FindIdentity(ctx, "app-qos",
		resource.WithIdentityFindQuery("shared", "true"), strict)
	if err != nil {
		return err
	}
	group, err := network.SecurityAddressGroups.FindIdentity(ctx, "app-addresses",
		resource.WithIdentityFindQuery("project_id", projectID), strict)
	if err != nil {
		return err
	}
	_, _ = policy.Rules, group.Addresses
	return nil
}

func Example_install() {
	_ = sdk.Connect
	_ = compute.CreateServerRequest{}
	_ = resource.ID("example-id")
	_ = request.Present(false)
}

func ListServersListing1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	for server, err := range service.Servers.Resources.List(ctx,
		resource.WithName("worker.*"),
		resource.WithStatus("ACTIVE"),
		resource.WithPageSize(50),
		resource.WithMaxItems(100),
	) {
		if err != nil {
			return err
		}
		fmt.Println(server.ID, server.Name, server.Status)
	}
	return nil
}

func ListRecordSetsListing2(ctx context.Context, conn *sdk.Connection, zoneID string) error {
	service, err := conn.DNS(ctx)
	if err != nil {
		return err
	}
	scope, err := service.RecordSets.InZone(ctx, resource.ID(zoneID))
	if err != nil {
		return err
	}
	records, err := scope.All(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false))
	if err != nil {
		return err
	}
	for _, record := range records {
		fmt.Println(record.ID, record.Name, record.Type, record.Records)
	}
	return nil
}

func ListMembersListing3(ctx context.Context, conn *sdk.Connection, poolID string) error {
	service, err := conn.LoadBalancer(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Pools.Members(ctx, resource.ID(poolID))
	if err != nil {
		return err
	}
	for member, err := range scope.List(ctx,
		resource.WithMaxItems(10),
		resource.WithPaginated(false),
		resource.WithStatus("ACTIVE"),
	) {
		if err != nil {
			return err
		}
		fmt.Println(member.ID, member.Name, member.ProvisioningStatus)
	}
	return nil
}

func ListLocalMatchesListing4(ctx context.Context, service *networkv2.Service) error {
	policies, err := service.QoSPolicies.Resources.All(ctx,
		resource.WithBodyFilter("rules", []map[string]any{}),
		resource.WithMaxItems(100))
	if err != nil {
		return err
	}
	groups, err := service.SecurityAddressGroups.Resources.All(ctx,
		resource.WithBodyFilters(map[string]any{
			"addresses": []string{"192.0.2.0/24"},
		}), resource.WithPaginated(false))
	if err != nil {
		return err
	}
	_, _ = policies, groups
	return nil
}

func ListSubnetMatchesListing4(ctx context.Context, service *networkv2.Service) error {
	for value, err := range service.Networks.Resources.List(ctx,
		resource.WithBodyFilter("subnet_ids", []string{"subnet-a", "subnet-b"}),
		resource.WithStatus("ACTIVE")) {
		if err != nil {
			return err
		}
		_ = value
	}
	_, err := service.SubnetPools.Resources.All(ctx,
		resource.WithBodyFilter("prefixes", []string{"192.0.2.0/24"}),
		resource.WithMaxItems(100))
	return err
}

func Example_microversions1() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx,
		sdk.WithMicroversionRange(sdk.Compute, "2.52", "2.90"),
		sdk.WithLatestMicroversion(sdk.BlockStorage),
	)
	if err != nil {
		panic(err)
	}

	compute, err := conn.ComputeV2(ctx)
	if err != nil {
		panic(err)
	}
	selection, err := conn.Microversion(ctx, sdk.Compute)
	if err != nil {
		panic(err)
	}
	fmt.Println(selection.RequestedMinimum, selection.RequestedMaximum)
	fmt.Println(selection.SupportedMinimum, selection.SupportedMaximum)
	fmt.Println(selection.Selected, compute.RawClient().Microversion)
}

func waitForServerServiceWaits1(ctx context.Context, client *gophercloud.ServiceClient, id string) (*servers.Server, error) {
	return servers.New(client).WaitForServer(ctx, resource.ID(id),
		resource.WithTimeout(time.Minute))
}

func waitForVolumeServiceWaits2(ctx context.Context, client *gophercloud.ServiceClient, id string) (*volumes.Volume, error) {
	bounded, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return volumes.New(client).WaitForAvailable(bounded, resource.ID(id))
}

func waitForVolumeDeletionServiceWaits2(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
	return volumes.New(client).WaitForDelete(ctx, resource.ID(id),
		resource.WithUnlimitedWait())
}

func Example_findFlavorStrictly() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	compute, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	flavor, err := compute.Flavors.FindIdentity(ctx, "flavor-id",
		resource.WithIdentityFindFallback(resource.FindFallbackNever),
		resource.WithIdentityFindIgnoreMissing(false),
		resource.WithIdentityFindExtraSpecs(true))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(flavor)
}

func Example_microversionPolicy() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx,
		sdk.WithLatestMicroversion(sdk.Compute),
		sdk.WithMicroversion(sdk.Compute, "2.52"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(conn != nil)
}

func Example_waitForDefaultRuleAttribute() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	rules := defaultrules.New(service.RawClient()).Resources()
	rule, err := networkcloud.WaitForStatus(ctx, rules, resource.ID("rule-id"), "IPv6", resource.WithStatusAttribute("ether_type"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rule)
}

func Example_listWithEarlyStop() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	seen := 0
	for server, err := range service.Servers.Resources.List(ctx, resource.WithPageSize(100)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(server.ID)
		if seen++; seen == 10 {
			break
		}
	}
}

func Example_waitOnAttribute() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BareMetalV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	node, err := service.Nodes.Resources.Wait(ctx, resource.ID("node-uuid"), "power on",
		resource.WithStatusAttribute("power_state"),
		resource.WithFailureStates("error"),
		resource.WithUnlimitedWait(),
		resource.WithProgressCallback(func(progress int) error { fmt.Println(progress); return nil }))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(node)
}
