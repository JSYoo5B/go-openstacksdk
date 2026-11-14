package servers_test

import (
	"context"
	"flag"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_userActions() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	id := flag.String("id", "", "literal server ID")
	version := flag.String("version", "2.91", "selected Nova microversion")
	action := flag.String("action", "", "clear-password, restore, lock, unshelve-host, unshelve-unpin, add-fixed, remove-fixed, add-floating, remove-floating")
	value := flag.String("value", "", "reason, host, network ID or address")
	flag.Parse()
	if *action == "" {
		return
	}
	if *id == "" {
		log.Fatal("-id is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, *version))
	if err != nil {
		log.Fatal(err)
	}
	compute, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := compute.Servers
	switch *action {
	case "clear-password":
		err = api.ClearPassword(ctx, *id)
	case "restore":
		err = api.Restore(ctx, *id)
	case "lock":
		err = api.LockWithOptions(ctx, *id, request.Custom(func(c *request.Config[servers.LockServerOpts]) error {
			v := *value
			c.Options.LockedReason = &v
			return nil
		}))
	case "unshelve-host":
		err = api.UnshelveWithOptions(ctx, *id, request.Custom(func(c *request.Config[servers.UnshelveServerOpts]) error { c.Options.Host = *value; return nil }))
	case "unshelve-unpin":
		err = api.UnshelveWithOptions(ctx, *id, servers.WithUnshelveUnpinnedAvailabilityZone())
	case "add-fixed":
		err = api.AddFixedIP(ctx, *id, *value)
	case "remove-fixed":
		err = api.RemoveFixedIP(ctx, *id, *value)
	case "add-floating":
		err = api.AddFloatingIP(ctx, *id, *value)
	case "remove-floating":
		err = api.RemoveFloatingIP(ctx, *id, *value)
	default:
		log.Fatal("unknown -action")
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Nova accepted the request")
}
