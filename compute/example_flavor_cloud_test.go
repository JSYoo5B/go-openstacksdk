package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func Example_flavorCloud() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	pattern := flag.String("pattern", "*", "flavor name or ID glob")
	identity := flag.String("flavor", "", "optional flavor ID or name to get")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runFlavorCloud(ctx, *cloud, *pattern, *identity); err != nil {
		log.Fatal(err)
	}
}

func recordValueFlavorCloud(record *flavors.FlavorRecord) any {
	if record == nil {
		return nil
	}
	var enrichment any
	if child := record.Enrichment; child != nil {
		enrichment = map[string]any{
			"resource": child.Resource, "wire": child.Wire,
			"extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
			"header": child.Header, "status_code": child.StatusCode,
		}
	}
	return map[string]any{
		"resource": record.Resource, "wire": record.Wire,
		"envelope": string(record.Envelope), "header": record.Header,
		"status_code": record.StatusCode, "enrichment": enrichment,
	}
}

func printQueryFlavorCloud(operation string, result *compute.FlavorQueryResult) error {
	if result == nil {
		return nil
	}
	inventory := make([]any, len(result.Inventory))
	for i, record := range result.Inventory {
		inventory[i] = recordValueFlavorCloud(record)
	}
	data, err := json.MarshalIndent(map[string]any{
		"operation": operation, "value": result.Value,
		"matched_count": len(result.Flavors), "inventory_count": len(result.Inventory),
		"inventory": inventory,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runFlavorCloud(ctx context.Context, cloud, pattern, identity string) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	all, err := service.AllFlavors(ctx)
	if printErr := printQueryFlavorCloud("all", all); printErr != nil {
		return errors.Join(err, printErr)
	}
	if err != nil {
		return fmt.Errorf("all flavors: %w", err)
	}
	matched, err := service.SearchFlavors(ctx, pattern)
	if printErr := printQueryFlavorCloud("search", matched); printErr != nil {
		return errors.Join(err, printErr)
	}
	if err != nil {
		return fmt.Errorf("search flavors: %w", err)
	}
	if identity == "" {
		return nil
	}
	found, err := service.GetFlavor(ctx, identity)
	if found != nil {
		data, printErr := json.MarshalIndent(map[string]any{"operation": "get", "record": recordValueFlavorCloud(found)}, "", "  ")
		if printErr != nil {
			return errors.Join(err, printErr)
		}
		fmt.Println(string(data))
	}
	if err != nil {
		return fmt.Errorf("get flavor: %w", err)
	}
	if found == nil {
		fmt.Println("flavor not found")
	}
	return nil
}
