package bgpvpns_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
)

func Example_portAssociationFind() {
	vpn := flag.String("vpn", "", "parent BGPVPN ID")
	association := flag.String("association", "", "association ID or name")
	port := flag.String("port", "", "port ID filter")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	row, err := service.API.BGPVPNs.FindPortAssociationRecord(ctx, *vpn, *association,
		request.IgnoreMissing(false),
		request.Query("port_id", *port))
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(row.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
