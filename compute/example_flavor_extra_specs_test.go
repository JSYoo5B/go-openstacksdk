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

func Example_flavorExtraSpecs() {
	id := flag.String("flavor-id", "", "flavor ID to fetch extra specs for")
	version := flag.String("microversion", "", "optional operation version; explicit empty is versionless")
	flag.Parse()
	versionSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "microversion" {
			versionSet = true
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runFlavorExtraSpecs(ctx, *id, *version, versionSet); err != nil {
		log.Fatal(err)
	}
}

func runFlavorExtraSpecs(ctx context.Context, id, version string, versionSet bool) error {
	if id == "" {
		return fmt.Errorf("-flavor-id is required")
	}
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	options := []flavors.FlavorExtraSpecsOption{}
	if versionSet {
		options = append(options, request.Microversion(version))
	}
	record, fetchErr := service.API.Flavors.FetchExtraSpecs(ctx,
		flavors.FlavorExtraSpecsRequest{ID: id}, options...)
	if record != nil {
		data, printErr := json.MarshalIndent(map[string]any{
			"resource":    record.Resource,
			"wire":        record.Wire,
			"extra_specs": record.ExtraSpecs,
			"status_code": record.StatusCode,
			"header":      record.Header,
		}, "", "  ")
		if printErr != nil {
			return errors.Join(fetchErr, printErr)
		}
		fmt.Println(string(data))
		fmt.Printf("response body: %q\n", record.Envelope)
	}
	return fetchErr
}
