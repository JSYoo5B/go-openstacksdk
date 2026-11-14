package v2_test

import (
	"context"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
)

func Example_routerInterfaces() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	id := flag.String("router", "", "literal router ID")
	action := flag.String("action", "", "add-interface, remove-interface, add-gateway, remove-gateway")
	port := flag.String("port", "", "interface port ID; wins subnet")
	subnet := flag.String("subnet", "", "interface subnet ID")
	network := flag.String("network", "", "single external gateway network ID")
	advertise := flag.Bool("advertise-host", false, "add-interface only; requires supporting backend")
	legacy := flag.Bool("legacy-source-route", false, "gateway only; requires backend legacy action support")
	flag.Parse()
	if *id == "" {
		log.Fatal("-router is required")
	}
	switch *action {
	case "add-interface", "remove-interface":
		if *port == "" && *subnet == "" {
			log.Fatal("provide -port or -subnet")
		}
		if *legacy {
			log.Fatal("-legacy-source-route is a gateway option")
		}
		if *action == "remove-interface" && *advertise {
			log.Fatal("-advertise-host is add-interface only")
		}
	case "add-gateway":
		if *network == "" {
			log.Fatal("-network is required")
		}
	case "remove-gateway":
		if *network != "" && !*legacy {
			log.Fatal("stock remove-gateway accepts no network attribute")
		}
	default:
		log.Fatal("select a supported -action")
	}
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
	var result *routers.RouterJSONResult
	switch *action {
	case "add-interface", "remove-interface":
		options := []routers.RouterInterfaceOption{}
		if *port != "" {
			options = append(options, routers.WithRouterInterfacePort(*port))
		}
		if *subnet != "" {
			options = append(options, routers.WithRouterInterfaceSubnet(*subnet))
		}
		if *action == "add-interface" {
			options = append(options, routers.WithRouterInterfaceAdvertiseHost(*advertise))
			result, err = service.Routers.AddInterfaceRecord(ctx, *id, options...)
		} else {
			result, err = service.Routers.RemoveInterfaceRecord(ctx, *id, options...)
		}
	case "add-gateway", "remove-gateway":
		options := []routers.RouterGatewayOption{request.Custom(func(c *routers.RouterGatewayOpts) error { c.LegacySourceRoutes = *legacy; return nil })}
		if *network != "" {
			options = append(options, request.Field("network_id", *network))
		}
		if *action == "add-gateway" {
			result, err = service.Routers.AddGatewayRecord(ctx, *id, options...)
		} else {
			result, err = service.Routers.RemoveGatewayRecord(ctx, *id, options...)
		}
	}
	if result != nil {
		fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
		fmt.Printf("actual envelope=%s\n", result.Envelope)
		if result.Value != nil {
			fmt.Printf("complete JSON value=%s\n", result.Value)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
