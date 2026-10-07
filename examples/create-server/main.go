package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func main() {
	name := flag.String("name", "", "server name")
	image := flag.String("image", "", "image name")
	flavor := flag.String("flavor", "", "flavor name")
	network := flag.String("network", "", "optional network name")
	flag.Parse()
	if *name == "" || *image == "" || *flavor == "" {
		log.Fatal("supply -name, -image and -flavor; this example creates a real server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	opts := []compute.CreateServerOption{compute.WithWait(resource.WithTimeout(5 * time.Minute))}
	if *network != "" {
		opts = append(opts, compute.WithNetworks(resource.Name(*network)))
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{Name: *name, Image: resource.Name(*image), Flavor: resource.Name(*flavor)}, opts...)
	if err != nil {
		if server != nil {
			log.Printf("server %s was created; wait failed", server.ID)
		}
		log.Fatal(err)
	}
	fmt.Println(server.ID, server.Status)
}
