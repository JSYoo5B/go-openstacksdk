package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding"
)

func printForwardingPortForwarding(value *portforwarding.PortForwardingRecord) {
	if value == nil {
		fmt.Println("null")
		return
	}
	body, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("parent=%s status=%d\n%s\n", value.ParentID, value.StatusCode, body)
}

func Example_portForwarding() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	floatingIP := flag.String("floating-ip", "", "floating IP ID")
	id := flag.String("id", "", "port forwarding ID to get and find")
	port := flag.String("port", "", "internal Neutron port ID")
	address := flag.String("address", "10.0.0.5", "internal IP address")
	create := flag.Bool("create", false, "create a port forwarding")
	description := flag.String("description", "", "replace description, including empty")
	remove := flag.String("delete", "", "port forwarding ID to delete")
	flag.Parse()
	if *floatingIP == "" {
		log.Fatal("-floating-ip is required")
	}
	update := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "description" {
			update = true
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.PortForwarding
	for row, err := range api.ListRecords(ctx, *floatingIP,
		portforwarding.WithRecordListQuery("protocol", "tcp"),
		request.MaxItems(100)) {
		if err != nil {
			log.Fatal(err)
		}
		printForwardingPortForwarding(row)
	}
	if *id != "" {
		row, err := api.GetRecord(ctx, *floatingIP, *id,
			portforwarding.WithRecordFields("id", "external_port"))
		if err != nil {
			log.Fatal(err)
		}
		printForwardingPortForwarding(row)
		found, err := api.FindRecord(ctx, *floatingIP, *id)
		if err != nil {
			log.Fatal(err)
		}
		printForwardingPortForwarding(found)
		if update {
			row, err := api.UpdateRecord(ctx, *floatingIP, *id,
				request.Field("description", *description))
			if err != nil {
				log.Fatal(err)
			}
			printForwardingPortForwarding(row)
		}
	}
	if *create {
		row, err := api.CreateRecord(ctx, *floatingIP,
			request.Field("internal_port_id", *port),
			request.Field("internal_ip_address", *address),
			request.Field("internal_port", 22),
			request.Field("external_port", 2222),
			request.Field("protocol", "tcp"))
		if err != nil {
			log.Fatal(err)
		}
		printForwardingPortForwarding(row)
	}
	if *remove != "" {
		if err := api.DeleteRecord(ctx, *floatingIP, *remove,
			request.IgnoreMissing(false)); err != nil {
			log.Fatal(err)
		}
	}
}
