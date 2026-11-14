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
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_availabilityZones() {
	cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
	unavailable := flag.Bool("unavailable", false, "include unavailable zone names")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runAvailabilityZones(ctx, *cloudName, *unavailable); err != nil {
		log.Fatal(err)
	}
}

func printJSONAvailabilityZones(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runAvailabilityZones(ctx context.Context, cloudName string, unavailable bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}

	records := make([]map[string]any, 0)
	for record, readErr := range service.ListAvailabilityZones(ctx) {
		if record != nil {
			records = append(records, map[string]any{
				"resource": record.Resource, "wire": record.Wire,
				"envelope": string(record.Envelope),
				"header":   record.Header, "status_code": record.StatusCode,
			})
		}
		if readErr != nil {
			return errors.Join(readErr, printJSONAvailabilityZones(map[string]any{
				"operation": "zones", "partial_records": records,
			}))
		}
	}
	if err := printJSONAvailabilityZones(map[string]any{"operation": "zones", "records": records}); err != nil {
		return err
	}

	options := make([]compute.AvailabilityZoneNamesOption, 0)
	if unavailable {
		options = append(options, request.Custom(func(c *request.Config[compute.AvailabilityZoneNamesOpts]) error {
			c.Options.Unavailable = true
			return nil
		}))
	}
	result, readErr := service.ListAvailabilityZoneNames(ctx, options...)
	if result != nil {
		suppressed := ""
		if result.SuppressedError != nil {
			suppressed = result.SuppressedError.Error()
		}
		printErr := printJSONAvailabilityZones(map[string]any{
			"operation": "names", "names": result.Names, "value": result.Value,
			"inventory_count": len(result.Inventory), "suppressed_error": suppressed,
		})
		if printErr != nil {
			return errors.Join(readErr, printErr)
		}
	}
	return readErr
}
