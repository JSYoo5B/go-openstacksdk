package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func main() {
	name := flag.String("name", "", "server name")
	imageName := flag.String("image", "", "image name")
	flavor := flag.String("flavor", "", "flavor name")
	privateNetwork := flag.String("network", "", "server network name")
	externalNetwork := flag.String("external-network", "", "floating IP allocation network name")
	fixedAddress := flag.String("fixed-address", "", "optional fixed IPv4 address when selection is ambiguous")
	flag.Parse()
	if *name == "" || *imageName == "" || *flavor == "" || *privateNetwork == "" || *externalNetwork == "" {
		log.Fatal("supply -name, -image, -flavor, -network and -external-network")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	computeService, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	server, err := computeService.Servers.Create(ctx, compute.CreateServerRequest{
		Name: *name, Image: resource.Name(*imageName), Flavor: resource.Name(*flavor),
	}, compute.WithNetworks(resource.Name(*privateNetwork)), compute.WithWait())
	if err != nil {
		if server != nil {
			log.Printf("created server: %s", server.ID)
		}
		log.Fatal(err)
	}
	fmt.Println("server", server.ID, server.Status)
	networkService, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []network.CreateFloatingIPOption{
		network.WithServer(resource.ID(server.ID)),
		network.WithNATDestination(resource.Name(*privateNetwork)),
		network.WithWait(),
	}
	if *fixedAddress != "" {
		options = append(options, network.WithFixedAddress(*fixedAddress))
	}
	floatingIP, err := networkService.FloatingIPs.Create(ctx, network.CreateFloatingIPRequest{
		Network: resource.Name(*externalNetwork),
	}, options...)
	if err != nil {
		if floatingIP != nil {
			log.Printf("allocated floating IP: %s", floatingIP.ID)
		}
		log.Fatal(err)
	}
	fmt.Println("floating IP", floatingIP.ID, floatingIP.FloatingIP, floatingIP.Status)
}
