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
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_flavorRecords() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	identity := flag.String("flavor", "", "optional flavor ID or name to find after listing")
	extra := flag.Bool("extra", false, "conditionally fetch missing or empty extra specs")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runFlavorRecords(ctx, *cloud, *identity, *extra); err != nil {
		log.Fatal(err)
	}
}

func runFlavorRecords(ctx context.Context, cloud, identity string, extra bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	printRecord := func(operation string, record *flavors.FlavorRecord) error {
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
		data, err := json.MarshalIndent(map[string]any{
			"operation": operation, "resource": record.Resource, "wire": record.Wire,
			"envelope": string(record.Envelope), "header": record.Header,
			"status_code": record.StatusCode, "enrichment": enrichment,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for record, err := range service.ListFlavors(ctx,
		request.MaxItems(20),
		request.Custom(func(c *request.Config[flavors.FlavorListOpts]) error { c.Options.GetExtraSpecs = extra; return nil })) {
		if printErr := printRecord("list", record); printErr != nil {
			return errors.Join(err, printErr)
		}
		if err != nil {
			return fmt.Errorf("list flavors: %w", err)
		}
	}
	if identity == "" {
		return nil
	}
	record, err := service.FindFlavor(ctx, identity, request.Custom(func(c *request.Config[flavors.FlavorFindOpts]) error { c.Options.GetExtraSpecs = extra; return nil }))
	if printErr := printRecord("find", record); printErr != nil {
		return errors.Join(err, printErr)
	}
	if err != nil {
		return fmt.Errorf("find flavor: %w", err)
	}
	if record == nil {
		fmt.Println("flavor not found")
	}
	return nil
}
