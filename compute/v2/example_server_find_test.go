package v2_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_serverFind() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	identity := flag.String("server", "", "exact server ID or name")
	basic := flag.Bool("basic", false, "use basic fallback list")
	allProjects := flag.Bool("all-projects", false, "request cross-project lookup")
	strict := flag.Bool("strict", false, "return an error for missing server")
	version := flag.String("microversion", "", "optional selected service microversion")
	flag.Parse()
	if *identity == "" {
		log.Fatal("-server is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	connectionOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
	if *version != "" {
		connectionOptions = append(connectionOptions,
			sdk.WithMicroversion(sdk.Compute, *version))
	}
	conn, err := sdk.Connect(ctx, connectionOptions...)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	record, err := service.Servers.FindRecord(ctx, *identity,
		request.IgnoreMissing(!*strict),
		request.Custom(func(c *servers.RecordFindOpts) error { v := !*basic; c.Details = &v; return nil }),
		request.Custom(func(c *servers.RecordFindOpts) error { v := *allProjects; c.AllProjects = &v; return nil }))
	if errors.Is(err, resource.ErrNotFound) {
		log.Fatal("server not found")
	}
	if errors.Is(err, resource.ErrAmbiguous) {
		log.Fatal("server name is ambiguous")
	}
	if err != nil {
		log.Fatal(err)
	}
	if record == nil {
		fmt.Println("null")
		return
	}
	body, err := json.MarshalIndent(record.Resource.Body, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
	fmt.Printf("HTTP %d microversion=%s request-id=%s\n",
		record.StatusCode, record.Microversion,
		record.Header.Get("X-Openstack-Request-Id"))
}
