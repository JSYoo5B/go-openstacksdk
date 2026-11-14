package servers_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSONServerMigrations(value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_serverMigrations() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	server := flag.String("server", "", "parent server ID")
	migration := flag.String("migration", "", "migration ID")
	action := flag.String("action", "", "list, get, abort, force or crash")
	version := flag.String("microversion", "2.80", "explicit Nova version")
	strict := flag.Bool("strict-abort", false, "abort missing is an error")
	flag.Parse()
	if *server == "" {
		log.Fatal("server is required")
	}
	if (*action == "get" || *action == "abort" || *action == "force") && *migration == "" {
		log.Fatal("migration is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, *version))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.Servers
	options := []servers.ServerMigrationOption{request.Microversion(*version)}
	switch *action {
	case "list":
		for row, err := range api.ListServerMigrationRecords(ctx, *server, options...) {
			if err != nil {
				log.Fatal(err)
			}
			printJSONServerMigrations(row.Resource)
		}
	case "get":
		row, err := api.GetServerMigrationRecord(ctx, *server, *migration, options...)
		if err != nil {
			log.Fatal(err)
		}
		printJSONServerMigrations(row.Resource)
	case "abort":
		options = append(options, request.IgnoreMissing(!*strict))
		result, err := api.AbortServerMigration(ctx, *server, *migration, options...)
		if err != nil {
			log.Fatal(err)
		}
		printJSONServerMigrations(result.Completed)
	case "force":
		result, err := api.ForceCompleteServerMigration(ctx, *server, *migration, options...)
		if err != nil {
			log.Fatal(err)
		}
		printJSONServerMigrations(result.Completed)
	case "crash":
		if err := api.TriggerServerCrashDump(ctx, *server); err != nil {
			log.Fatal(err)
		}
		fmt.Println("crash-dump request accepted")
	default:
		log.Fatal("action must be list, get, abort, force or crash")
	}
}
