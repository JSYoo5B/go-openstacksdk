package compute_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_createWithAutomaticFloatingIp() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	name := flag.String("name", "web-01", "new server name")
	image := flag.String("image", "ubuntu", "exact image name")
	flavor := flag.String("flavor", "small", "exact flavor name")
	nic := flag.String("network", "private", "exact NIC network name")
	external := flag.String("floating-network", "", "exact external network name; empty uses roles/router")
	volume := flag.String("boot-volume", "", "existing boot volume ID")
	size := flag.Int("boot-size", 0, "new image-backed boot volume GiB")
	enabled := flag.Bool("auto-ip", true, "enable conditional floating IPv4 assignment")
	reuse := flag.Bool("reuse", true, "reuse an attached/free floating IP")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := runCreateWithAutomaticFloatingIp(ctx, *cloud, *name, *image, *flavor, *nic, *external, *volume, *size, *enabled, *reuse); err != nil {
		log.Fatal(err)
	}
}

func runCreateWithAutomaticFloatingIp(ctx context.Context, cloud, name, image, flavor, nic, external, volume string, size int, enabled, reuse bool) error {
	if volume != "" && size != 0 {
		return fmt.Errorf("boot-volume and boot-size are mutually exclusive")
	}
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.37"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	request := compute.CreateServerRequest{
		Name: name, Image: resource.Name(image), Flavor: resource.Name(flavor),
	}
	serverOptions := []compute.CreateServerOption{
		compute.WithNetworks(resource.Name(nic)),
		compute.WithWait(resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Second)),
	}
	if volume != "" {
		request.Image = resource.Ref{}
		serverOptions = append(serverOptions, compute.WithBootVolume(resource.ID(volume)))
	}
	if size != 0 {
		serverOptions = append(serverOptions, compute.WithBootVolumeSize(size))
	}
	options := compute.AutomaticServerCreateOptions{
		Server: serverOptions,
		AutomaticIP: []compute.AutomaticFloatingIPOption{
			compute.WithAutomaticIPEnabled(enabled),
			compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
			compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(reuse)),
			compute.WithAutomaticIPWait(resource.WithTimeout(3*time.Minute), resource.WithPollInterval(time.Second)),
		},
	}
	if external != "" {
		options.FloatingIPNetwork = resource.Name(external)
	}
	result, err := service.CreateWithAutomaticFloatingIP(ctx, request, options)
	if result != nil {
		if result.Creation != nil {
			fmt.Printf("creation-id=%s\n", result.Creation.ID)
		}
		if result.Server != nil {
			fmt.Printf("server=%s status=%s\n", result.Server.ID, result.Server.Status)
		}
		if automatic := result.Automatic; automatic != nil {
			fmt.Printf("observed=%t\n", automatic.Observed)
			if decision := automatic.Decision; decision != nil {
				fmt.Printf("needed=%t reason=%s backend=%s\n", decision.Needed, decision.Reason, decision.Backend)
			}
			if assignment := automatic.Assignment; assignment != nil {
				fmt.Printf("reused=%t allocated=%t\n", assignment.Reused, assignment.Allocated)
				if ip := assignment.FloatingIP; ip != nil {
					fmt.Printf("ip-id=%s floating4=%s status=%s\n", ip.ID, ip.FloatingIP, ip.Status)
				}
			}
		}
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("the shared deadline expired; inspect the returned resources")
		}
		var response *resource.ResponseError
		if errors.As(err, &response) {
			log.Printf("response evidence: status=%d", response.StatusCode)
		}
		return fmt.Errorf("create with automatic floating IPv4: %w", err)
	}
	return nil
}
