package network_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	network "github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"time"
)

func networkService(ctx context.Context) *network.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_waitAndDelete() {
	ctx := context.Background()
	service := networkService(ctx)
	ready, err := service.Networks.Wait(ctx, resource.Name("private"), "ACTIVE")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ready.Status)
	if err := service.Networks.Delete(ctx, resource.Name("private"), resource.WithMissingError()); err != nil {
		log.Fatal(err)
	}
}

func Example_createFloatingIP() {
	ctx := context.Background()
	service := networkService(ctx)
	created, err := service.FloatingIPs.Create(ctx,
		network.CreateFloatingIPRequest{Network: resource.Name("public")},
		network.WithServer(resource.Name("web")),
		network.WithNATDestination(resource.Name("private")),
		network.WithWait(resource.WithTimeout(time.Minute)))
	if err != nil {
		if created != nil {
			fmt.Println("created floating IP remains:", created.ID)
		}
		log.Fatal(err)
	}
	fmt.Println(created.FloatingIP, created.PortID, created.FixedIP)
}

func Example_createFloatingIPOnPort() {
	ctx := context.Background()
	service := networkService(ctx)
	created, err := service.FloatingIPs.Create(ctx,
		network.CreateFloatingIPRequest{Network: resource.ID("external-network-id")},
		network.WithServer(resource.ID("server-id")),
		network.WithPort(resource.ID("port-id")),
		network.WithFixedAddress("10.0.0.10"),
		network.WithDescription("web ingress"),
		request.Field("vendor_enabled", false))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(created.ID)
}

func Example_clearNetworkDNSDomain() {
	ctx := context.Background()
	updated, err := networkService(ctx).UpdateNetwork(ctx, resource.ID("network-id"),
		network.WithNetworkDNSDomainValue(request.Null[string]()))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(updated)
}

func Example_createProviderNetwork() {
	ctx := context.Background()
	provider := network.ProviderNetwork{
		NetworkType:     request.Present("vlan"),
		PhysicalNetwork: request.Present("physnet1"),
		SegmentationID:  request.Present[any](100),
	}
	created, err := networkService(ctx).CreateNetwork(ctx,
		network.CreateNetworkRequest{Name: "provider-net"},
		network.WithNetworkProvider(provider))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(created)
}

func Example_createExternalNetwork() {
	ctx := context.Background()
	detail, err := networkService(ctx).CreateNetworkDetail(ctx,
		network.CreateNetworkRequest{Name: "public"},
		network.WithNetworkExternal(true),
		network.WithNetworkMTU(1450))
	if err != nil {
		log.Fatal(err)
	}
	if detail.RouterExternal != nil && *detail.RouterExternal && detail.MTU != nil {
		fmt.Println(detail.ID, "external, MTU", *detail.MTU)
	}
	fmt.Println(string(detail.Metadata.Body["provider:network_type"]))
}
