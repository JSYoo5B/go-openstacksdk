package compute_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_userReadApis() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runUserReadApis(ctx); err != nil {
		log.Fatal(err)
	}
}

func runUserReadApis(ctx context.Context) error {
	if len(os.Args) != 4 && len(os.Args) != 5 {
		return fmt.Errorf("usage: %s SERVER_ID FLAVOR_ID KEYPAIR_NAME [OWNER_USER_ID]", os.Args[0])
	}
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
	if err != nil {
		return err
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	flavor, err := service.Flavors.FindIdentity(ctx, os.Args[2],
		resource.WithIdentityFindFallback(resource.FindFallbackNever),
		resource.WithIdentityFindIgnoreMissing(false),
		resource.WithIdentityFindExtraSpecs(true),
	)
	if err != nil {
		return fmt.Errorf("get flavor: %w", err)
	}
	metadata, err := service.Servers.Metadata(ctx, os.Args[1])
	if err != nil {
		return fmt.Errorf("get server metadata: %w", err)
	}
	var keyOptions []keypairs.GetOption
	if len(os.Args) == 5 {
		keyOptions = append(keyOptions,
			keypairs.WithGetOptions(keypairs.GetOpts{UserID: os.Args[4]}))
	}
	pair, err := service.KeyPairs.Get(ctx, os.Args[3], keyOptions...)
	if err != nil {
		return fmt.Errorf("get keypair: %w", err)
	}
	output, err := service.Servers.ConsoleOutput(ctx, os.Args[1],
		request.Custom(func(c *request.Config[servers.ConsoleOutputOpts]) error { v := 50; c.Options.Length = &v; return nil }))
	if err != nil {
		return fmt.Errorf("get console output: %w", err)
	}
	data, err := json.MarshalIndent(map[string]any{
		"flavor":         flavor,
		"metadata":       metadata,
		"keypair":        pair,
		"console_output": output,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
