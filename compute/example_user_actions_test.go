package compute_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles"
)

func Example_userActions() {
	serverID := flag.String("server-id", "", "server ID for the remote console")
	keyName := flag.String("keypair", "", "keypair name to delete")
	ownerID := flag.String("owner-user-id", "", "optional keypair owner")
	strict := flag.Bool("strict-missing", false, "report a missing keypair")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runUserActions(ctx, *serverID, *keyName, *ownerID, *strict); err != nil {
		log.Fatal(err)
	}
}

func runUserActions(ctx context.Context, serverID, keyName, ownerID string, strict bool) error {
	if serverID == "" || keyName == "" {
		return fmt.Errorf("-server-id and -keypair are required")
	}
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
	if err != nil {
		return err
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	console, err := service.RemoteConsoles.CreateConsole(ctx, serverID,
		request.Custom(func(c *request.Config[remoteconsoles.ConsoleCreateOpts]) error {
			c.Options.Type = request.Present("novnc")
			return nil
		}))
	if console != nil {
		data, printErr := json.MarshalIndent(map[string]any{
			"resource":    console.Resource,
			"wire":        console.Wire,
			"status_code": console.StatusCode,
			"header":      console.Header,
		}, "", "  ")
		if printErr != nil {
			return printErr
		}
		fmt.Println(string(data))
	}
	if err != nil {
		return fmt.Errorf("create remote console: %w", err)
	}
	options := []keypairs.KeypairDeleteOption{
		request.IgnoreMissing(!strict),
	}
	if ownerID != "" {
		options = append(options, request.Custom(func(c *request.Config[keypairs.KeypairDeleteOpts]) error { c.Options.UserID = ownerID; return nil }))
	}
	if err := service.KeyPairs.DeleteKeypair(ctx, keyName, options...); err != nil {
		return fmt.Errorf("delete keypair: %w", err)
	}
	fmt.Println("keypair deletion accepted or missing")
	return nil
}
