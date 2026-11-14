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
)

func Example_cloudFlavorRam() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	ram := flag.String("ram", "4096", "minimum RAM as JSON number")
	include := flag.String("include", "null", "name substring as JSON string, or null")
	extra := flag.Bool("get-extra", true, "fetch missing extra specs before selection")
	flag.Parse()
	if !json.Valid([]byte(*ram)) || !json.Valid([]byte(*include)) {
		log.Fatal("ram and include must be complete JSON")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, readErr := service.GetFlavorByRAM(ctx, json.RawMessage(*ram),
		compute.WithFlavorRAMInclude(json.RawMessage(*include)),
		compute.WithFlavorRAMGetExtra(*extra))
	var selected any
	inventoryCount := 0
	if result != nil {
		inventoryCount = len(result.Inventory)
		if row := result.Flavor; row != nil {
			var enrichment any
			if e := row.Enrichment; e != nil {
				enrichment = map[string]any{
					"resource": e.Resource, "wire": e.Wire, "extra_specs": e.ExtraSpecs,
					"envelope_bytes": []byte(e.Envelope), "header": e.Header, "status_code": e.StatusCode,
				}
			}
			selected = map[string]any{
				"resource": row.Resource, "wire": row.Wire,
				"envelope": row.Envelope, "page_envelope_bytes": []byte(row.PageEnvelope), "header": row.Header,
				"status_code": row.StatusCode, "enrichment": enrichment,
			}
		}
	}
	data, printErr := json.MarshalIndent(map[string]any{
		"selected": selected, "inventory_count": inventoryCount,
	}, "", "  ")
	if printErr == nil {
		fmt.Println(string(data))
	}
	if err := errors.Join(readErr, printErr); err != nil {
		log.Fatal(err)
	}
}
