package compute_test

import (
	"context"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func connect(ctx context.Context) (*openstack.Connection, *compute.Service) {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return conn, service
}

func reportCreate(server *compute.Server, err error) {
	if err != nil {
		if server != nil {
			fmt.Println("created:", server.ID)
		}
		log.Fatal(err)
	}
}

func ExampleServers_Create() {
	ctx := context.Background()
	_, service := connect(ctx)
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name:   "web-01",
		Image:  resource.Name("ubuntu"),
		Flavor: resource.Name("small"),
	}, compute.WithNetworks(resource.Name("private")),
		compute.WithKeyName("my-key"),
		compute.WithConfigDrive(false),
		compute.WithWait(resource.WithTimeout(5*time.Minute)))
	reportCreate(server, err)
}

func ExampleWithBootVolume() {
	ctx := context.Background()
	_, service := connect(ctx)
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name:   "volume-server",
		Flavor: resource.Name("small"),
	}, compute.WithBootVolume(resource.Name("root-volume")),
		compute.WithNetworks(resource.Name("private")),
		compute.WithWait(resource.WithTimeout(5*time.Minute)))
	reportCreate(server, err)
}

func ExampleWithBootVolumeSize() {
	ctx := context.Background()
	_, service := connect(ctx)
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name:   "new-volume-server",
		Image:  resource.Name("ubuntu"),
		Flavor: resource.Name("small"),
	}, compute.WithBootVolumeSize(50),
		compute.WithWait(resource.WithTimeout(5*time.Minute)))
	reportCreate(server, err)
}

func ExampleWithServerGroup() {
	ctx := context.Background()
	_, service := connect(ctx)
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name:   "grouped",
		Image:  resource.Name("ubuntu"),
		Flavor: resource.Name("small"),
	}, compute.WithServerGroup(resource.Name("anti-affinity")),
		compute.WithSchedulerHints(map[string]any{"different_host": []string{"other-id"}}),
		compute.WithDataVolumes(resource.Name("data")),
		compute.WithUserData([]byte("#cloud-config\n")),
		compute.WithMetadata(map[string]string{"role": "web"}))
	reportCreate(server, err)
}

func ExampleServers_Wait() {
	ctx := context.Background()
	_, service := connect(ctx)
	ready, err := service.Servers.Wait(ctx, resource.Name("web-01"), "ACTIVE",
		resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ready.Status)
	if err := service.Servers.Delete(ctx, resource.ID(ready.ID)); err != nil {
		log.Fatal(err)
	}
}

func ExampleService_GetServerConsole() {
	ctx := context.Background()
	_, service := connect(ctx)
	result, err := service.GetServerConsole(ctx, "vm", compute.WithServerConsoleLength(50))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}

func ExampleCreateFloatingIPRequest() {
	ctx := context.Background()
	conn, _ := connect(ctx)
	networkName := "public"
	result, err := conn.CreateFloatingIP(ctx,
		compute.CreateFloatingIPRequest{Network: &networkName},
		func(c *compute.FloatingIPCreateOpts) error { c.Server = resource.Name("web-01"); return nil },
		compute.WithFloatingIPCreateWait(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}

func ExampleDeleteFloatingIPRequest() {
	ctx := context.Background()
	conn, _ := connect(ctx)
	result, err := conn.DeleteFloatingIP(ctx,
		compute.DeleteFloatingIPRequest{ID: "floating-ip-id"},
		func(c *compute.FloatingIPDeleteOpts) error { c.Retries = 1; return nil },
		compute.WithFloatingIPDeleteTimeout(time.Minute),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	unattached, err := conn.DeleteUnattachedFloatingIPs(ctx,
		func(c *compute.FloatingIPDeleteOpts) error { c.Retries = 1; return nil },
		compute.WithFloatingIPDeleteTimeout(time.Minute),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(unattached)
}
