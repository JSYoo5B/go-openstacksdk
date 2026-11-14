package network_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"log"
	"os"
	"time"
)

func Example_cloudSecurityGroupReads() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	mode := flag.String("mode", "search", "search or get")
	name := flag.String("name", "", "name, ID or fnmatch pattern")
	filters := flag.String("filters", "null", "raw JSON filters; quote a JMESPath string")
	existing := flag.String("existing", "", "already fetched JSON object containing id")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var connect []sdk.ConnectionOption
	if *cloud != "" {
		connect = append(connect, sdk.WithCloud(*cloud))
	}
	conn, err := sdk.Connect(ctx, connect...)
	if err != nil {
		log.Fatal(err)
	}
	options := []network.SecurityGroupReadOption{network.WithSecurityGroupReadFiltersJSON(json.RawMessage(*filters))}
	var result *network.SecurityGroupReadResult
	switch *mode {
	case "search":
		result, err = conn.SearchSecurityGroupRecords(ctx, *name, options...)
	case "get":
		var reference any = *name
		if *existing != "" {
			reference = json.RawMessage(*existing)
		}
		result, err = conn.GetSecurityGroupRecord(ctx, reference, options...)
	default:
		log.Fatal("mode must be search or get")
	}
	if err != nil {
		if result != nil {
			fmt.Fprintf(os.Stderr, "backend=%s observed=%d\n", result.Backend, len(result.Inventory))
		}
		log.Fatal(err)
	}
	if result.Value == nil {
		fmt.Println("null")
		return
	}
	fmt.Println(string(result.Value))
}
