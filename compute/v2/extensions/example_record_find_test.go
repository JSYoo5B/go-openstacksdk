package extensions_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_recordFind() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	identity := flag.String("extension", "", "literal alias or exact name")
	strict := flag.Bool("strict", false, "return missing errors")
	flag.Parse()
	if *identity == "" {
		log.Fatal("-extension is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	compute, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	value, err := compute.Extensions.FindRecord(ctx, *identity,
		request.IgnoreMissing(!*strict))
	if err != nil {
		log.Fatal(err)
	}
	var output any
	if value != nil {
		output = value.Resource
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
