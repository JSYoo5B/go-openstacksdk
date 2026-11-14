package compute_test

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverNetworkInterfaces() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runServerNetworkInterfaces(ctx); err != nil {
		log.Fatal(err)
	}
}

func runServerNetworkInterfaces(ctx context.Context) error {
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.42"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name: "web-01", Image: resource.Name("ubuntu"), Flavor: resource.Name("small"),
	}, compute.WithNetworkInterfaces(
		compute.ServerNetworkInterface{Network: resource.Name("private"), FixedIP: "10.0.0.10", Tag: request.Present("app")},
		compute.ServerNetworkInterface{Port: resource.Name("app-port"), Tag: request.Present("existing-port")},
	))
	if err != nil {
		return err
	}
	fmt.Println(server.ID)
	return nil
}
