package compute_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
)

func Example_keypairsConsole() {
	name := flag.String("keypair", "", "keypair name to create")
	publicKeyPath := flag.String("public-key-file", "", "SSH public key to import")
	owner := flag.String("owner-user-id", "", "optional owner")
	serverID := flag.String("server-id", "", "optional server ID for a legacy console")
	consoleType := flag.String("console-type", "novnc", "legacy console type")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runKeypairsConsole(ctx, *name, *publicKeyPath, *owner, *serverID, *consoleType); err != nil {
		log.Fatal(err)
	}
}

func runKeypairsConsole(ctx context.Context, name, publicKeyPath, owner, serverID, consoleType string) error {
	if name == "" || publicKeyPath == "" {
		return fmt.Errorf("-keypair and -public-key-file are required")
	}
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
	if err != nil {
		return err
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	options := []keypairs.KeypairCreateOption{
		keypairs.WithKeypairCreateAttribute("name", name),
		keypairs.WithKeypairCreateAttribute("public_key", string(publicKey)),
	}
	if owner != "" {
		options = append(options, keypairs.WithKeypairCreateAttribute("user_id", owner))
	}
	created, err := service.KeyPairs.CreateKeypair(ctx, options...)
	if err != nil {
		return fmt.Errorf("create keypair: %w", err)
	}
	data, err := json.MarshalIndent(map[string]any{
		"resource": created.Resource, "wire": created.Wire,
		"status_code": created.StatusCode, "header": created.Header,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	if serverID != "" {
		console, err := service.Servers.ConsoleURL(ctx, serverID, consoleType)
		if err != nil {
			return fmt.Errorf("get legacy console: %w", err)
		}
		fmt.Println(string(console))
	}
	return nil
}
