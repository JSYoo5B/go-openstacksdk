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
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_keypairsCloud() {
	operation := flag.String("operation", "list", "list, search, get, create, delete")
	name := flag.String("name", "", "keypair name or search pattern")
	filters := flag.String("filters", "", "optional filters as JSON")
	owner := flag.String("owner-user-id", "", "owner for unfiltered get")
	publicKey := flag.String("public-key", "", "optional public key for create")
	flag.Parse()
	filtersSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "filters" {
			filtersSet = true
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runKeypairsCloud(ctx, *operation, *name, *filters, filtersSet, *owner, *publicKey); err != nil {
		log.Fatal(err)
	}
}

func runKeypairsCloud(ctx context.Context, operation, name, filters string, filtersSet bool, owner, publicKey string) error {
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	options := []compute.KeypairQueryOption{request.Custom(func(c *request.Config[compute.KeypairQueryOpts]) error { c.Options.UserID = owner; return nil })}
	if filtersSet {
		options = append(options, compute.WithKeypairQueryFilters(json.RawMessage(filters)))
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	printResult := func(value any, callErr error) error {
		return errors.Join(callErr, encoder.Encode(value))
	}
	switch operation {
	case "list":
		result, err := service.ListKeypairs(ctx, options...)
		return printResult(result, err)
	case "search":
		result, err := service.SearchKeypairs(ctx, name, options...)
		return printResult(result, err)
	case "get":
		result, err := service.GetKeypair(ctx, name, options...)
		return printResult(result, err)
	case "create":
		result, err := service.CreateKeypair(ctx, name, request.Custom(func(c *request.Config[compute.CloudKeypairCreateOpts]) error {
			c.Options.PublicKey = publicKey
			return nil
		}))
		return printResult(result, err)
	case "delete":
		deleted, err := service.DeleteKeypair(ctx, name)
		return printResult(map[string]bool{"deleted": deleted}, err)
	default:
		return fmt.Errorf("unknown operation %q", operation)
	}
}
