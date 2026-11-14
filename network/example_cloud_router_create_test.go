package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func Example_cloudRouterCreate() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kwargs := flag.String("kwargs", "{}", "JSON create_router kwargs")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	result, err := conn.CreateRouterRecord(ctx,
		network.WithRouterCreateJSON(json.RawMessage(*kwargs)),
	)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(result.Router, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
