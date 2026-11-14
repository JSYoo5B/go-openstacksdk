package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
)

func Example_routerActions1() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	id := flag.String("router", "", "literal router ID")
	action := flag.String("action", "", "add-routes, remove-routes, add-gateways, update-gateways, remove-gateways")
	body := flag.String("body", "", "complete JSON action body, including router envelope")
	flag.Parse()
	if *id == "" || !json.Valid([]byte(*body)) {
		log.Fatal("-router and a valid JSON -body are required")
	}
	switch *action {
	case "add-routes", "remove-routes", "add-gateways", "update-gateways", "remove-gateways":
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
	api := service.Routers
	raw := json.RawMessage(*body)
	var result *routers.RouterActionRecord
	switch *action {
	case "add-routes":
		result, err = api.AddExtraRoutesRecord(ctx, *id, raw)
	case "remove-routes":
		result, err = api.RemoveExtraRoutesRecord(ctx, *id, raw)
	case "add-gateways":
		result, err = api.AddExternalGatewaysRecord(ctx, *id, raw)
	case "update-gateways":
		result, err = api.UpdateExternalGatewaysRecord(ctx, *id, raw)
	case "remove-gateways":
		result, err = api.RemoveExternalGatewaysRecord(ctx, *id, raw)
	}
	if result != nil {
		fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
		fmt.Printf("actual envelope=%s\n", result.Envelope)
		if result.Resource != nil {
			view, encodeErr := json.MarshalIndent(result.Resource, "", "  ")
			if encodeErr != nil {
				log.Fatal(encodeErr)
			}
			fmt.Printf("Source descriptor view=%s\n", view)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
