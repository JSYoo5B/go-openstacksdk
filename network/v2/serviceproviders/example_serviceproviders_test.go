package serviceproviders_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_serviceproviders() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	serviceType := flag.String("service-type", "", "server service type")
	defaultMode := flag.String("default", "any", "any, yes or no")
	expression := flag.String("jmes", "", "deprecated Python Proxy expression")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []serviceproviders.RecordListOption{
		serviceproviders.WithRecordListJMESPath(*expression),
	}
	if *serviceType != "" {
		options = append(options, request.Query("service_type", *serviceType))
	}
	switch *defaultMode {
	case "any":
	case "yes", "no":
		options = append(options, request.Query("is_default", *defaultMode == "yes"))
	default:
		log.Fatal("default must be any, yes or no")
	}
	result, err := network.ServiceProviders.ProxyListRecords(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	if result.JSON != nil {
		fmt.Println(string(result.JSON))
		return
	}
	generator, ok := result.Value.(*serviceproviders.RecordListGenerator)
	if !ok {
		fmt.Printf("opaque expression result: %T\n", result.Value)
		return
	}
	for value, err := range generator.Records {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(value.Resource, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(data))
	}
}
