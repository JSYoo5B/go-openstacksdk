package v1_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_waits() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "order", "container/order/secret")
	id := flag.String("id", "", "literal executable ID")
	operation := flag.String("operation", "status", "status/delete")
	status := flag.String("status", "ACTIVE", "target state")
	nullStatus := flag.Bool("null-status", false, "target is Source None")
	interval := flag.Duration("interval", 2*time.Second, "polling interval")
	timeout := flag.Duration("timeout", 0, "positive override; zero keeps Source default")
	disableFailures := flag.Bool("disable-failures", false, "disable status failure states")
	flag.Parse()
	ctx := context.Background()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := waits.New(service.RawClient())
	var target waits.Target
	switch *kind {
	case "container":
		target = waits.Container(*id)
	case "order":
		target = waits.Order(*id)
	case "secret":
		target = waits.Secret(*id)
	default:
		log.Fatal("unknown kind")
	}
	options := []waits.Option{
		waits.WithInterval(*interval),
		waits.WithWait(resource.WithProgressCallback(func(progress int) error {
			fmt.Println("progress", progress)
			return nil
		})),
	}
	if *timeout != 0 {
		options = append(options, waits.WithTimeout(*timeout))
	}
	var result *waits.Record
	switch *operation {
	case "delete":
		result, err = api.WaitForDelete(ctx, target, options...)
	case "status":
		if *disableFailures {
			options = append(options, waits.WithWait(resource.WithFailureStates()))
		}
		selected := status
		if *nullStatus {
			selected = nil
		}
		result, err = api.WaitForStatus(ctx, target, selected, options...)
	default:
		log.Fatal("unknown operation")
	}
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
