package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_keypairsListFind() {
	name := flag.String("find", "", "keypair name to find instead of listing")
	owner := flag.String("owner-user-id", "", "optional owner")
	kind := flag.String("type", "", "optional local list filter, for example ssh")
	maximum := flag.Int("max-items", 0, "raw list row cap; zero is unlimited")
	strict := flag.Bool("strict", false, "return an error when find has no match")
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
	if err := runKeypairsListFind(ctx, *name, *owner, *kind, *maximum, *strict, *version, versionSet); err != nil {
		log.Fatal(err)
	}
}

func runKeypairsListFind(ctx context.Context, name, owner, kind string, maximum int, strict bool, version string, versionSet bool) error {
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	api := service.API.KeyPairs
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	printRecord := func(record *keypairs.KeypairRecord) error {
		return encoder.Encode(map[string]any{
			"resource": record.Resource, "wire": record.Wire,
			"envelope":    string(record.Envelope),
			"status_code": record.StatusCode, "header": record.Header,
		})
	}
	if name != "" {
		options := []keypairs.KeypairFindOption{
			request.Custom(func(c *request.Config[keypairs.KeypairFindOpts]) error { c.Options.UserID = owner; return nil }),
			request.IgnoreMissing(!strict),
		}
		if versionSet {
			options = append(options, request.Microversion(version))
		}
		record, err := api.FindKeypair(ctx, name, options...)
		if record != nil {
			if printErr := printRecord(record); printErr != nil {
				return errors.Join(err, printErr)
			}
		}
		if err != nil {
			return fmt.Errorf("find keypair: %w", err)
		}
		if record == nil {
			fmt.Println("keypair not found")
		}
		return nil
	}
	options := []keypairs.KeypairListOption{
		request.Custom(func(c *request.Config[keypairs.KeypairListOpts]) error { c.Options.UserID = owner; return nil }),
		request.MaxItems(maximum),
	}
	if kind != "" {
		options = append(options, keypairs.WithKeypairListFilter("type", kind))
	}
	if versionSet {
		options = append(options, request.Microversion(version))
	}
	for record, err := range api.ListRecords(ctx, options...) {
		if err != nil {
			return fmt.Errorf("list keypairs: %w", err)
		}
		if err := printRecord(record); err != nil {
			return err
		}
	}
	return nil
}
