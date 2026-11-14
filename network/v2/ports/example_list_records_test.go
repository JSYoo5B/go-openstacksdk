package ports_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func Example_listRecords() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	networkID := flag.String("network", "", "server network_id filter")
	host := flag.String("host", "", "local binding_host_id filter")
	mode := flag.String("mode", "list", "list, proxy, eager or search")
	expression := flag.String("expression", "[*].id", "JMESPath expression")
	name := flag.String("name", "", "search name, ID or glob")
	filters := flag.String("filter", "", "search raw JSON filters")
	flag.Parse()
	if err := runListRecords(*cloud, *networkID, *host, *mode, *expression, *name, *filters); err != nil {
		log.Fatal(err)
	}
}

func runListRecords(cloud, networkID, host, mode, expression, name, filters string) error {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		return err
	}
	api := network.Ports
	var options []ports.RecordListOption
	if networkID != "" {
		options = append(options, ports.WithRecordListQuery("network_id", networkID))
	}
	if host != "" {
		options = append(options, ports.WithRecordListQuery("binding_host_id", host))
	}
	switch mode {
	case "list":
		for row, err := range api.ListRecords(ctx, options...) {
			if err != nil {
				return err
			}
			if err := printPortListRecords(row); err != nil {
				return err
			}
		}
	case "proxy":
		options = append(options, ports.WithRecordListQuery("jmespath_filters", expression))
		result, err := api.ProxyListRecords(ctx, options...)
		if err != nil {
			return err
		}
		if generator, ok := result.Value.(*ports.RecordListGenerator); ok {
			for row, err := range generator.Records {
				if err != nil {
					return err
				}
				if err := printPortListRecords(row); err != nil {
					return err
				}
			}
		} else if result.JSON != nil {
			fmt.Println(string(result.JSON))
		} else {
			fmt.Printf("opaque result: %T\n", result.Value)
		}
	case "eager":
		result, err := api.ListRecordValues(ctx, expression, options...)
		if err != nil {
			return err
		}
		fmt.Println(string(result.Value))
	case "search":
		var search []ports.RecordSearchOption
		if filters != "" {
			search = append(search, ports.WithRecordSearchFilters(json.RawMessage(filters)))
		}
		result, err := api.SearchRecords(ctx, name, search...)
		if err != nil {
			return err
		}
		fmt.Println(string(result.Value))
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
	return nil
}

func printPortListRecords(row *ports.PortRecord) error {
	data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
