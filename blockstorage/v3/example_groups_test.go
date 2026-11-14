package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func printGroupGroups(value *resource.RawResource) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}

func Example_groups() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	version := flag.String("microversion", "3.38", "Cinder selected version")
	action := flag.String("action", "list", "list/get/create/clone/update/delete")
	id := flag.String("group", "", "group ID")
	name := flag.String("name", "sample", "name for create/clone/update")
	groupType := flag.String("group-type", "", "existing group-type name or ID")
	volumeTypes := flag.String("volume-types", "", "comma-separated volume-type IDs")
	sourceGroup := flag.String("source-group", "", "source group ID for clone")
	sourceSnapshot := flag.String("source-snapshot", "", "source group-snapshot ID for clone")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.BlockStorage, *version))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := groups.New(service.RawClient())
	var value *groups.GroupRecord
	switch *action {
	case "list":
		for row, err := range api.ListRecords(ctx) {
			if err != nil {
				log.Fatal(err)
			}
			printGroupGroups(row.Resource)
		}
		return
	case "get":
		value, err = api.GetRecord(ctx, *id, request.Custom(func(c *groups.RecordGetOpts) error { v := true; c.ListVolume = &v; return nil }))
	case "create":
		value, err = api.CreateRecord(ctx,
			request.Field("group_type", *groupType),
			request.Field("volume_types", strings.Split(*volumeTypes, ",")),
			request.Field("name", *name))
	case "clone":
		var snapshot, source any
		if *sourceSnapshot != "" {
			snapshot = *sourceSnapshot
		}
		if *sourceGroup != "" {
			source = *sourceGroup
		}
		value, err = api.CreateFromSourceRecord(ctx,
			groups.WithRecordSourceGroupSnapshotID(snapshot),
			groups.WithRecordSourceSourceGroupID(source),
			groups.WithRecordSourceName(*name))
	case "update":
		value, err = api.UpdateRecord(ctx, *id, request.Field("name", *name))
	case "delete":
		err = api.DeleteRecord(ctx, *id, request.IgnoreMissing(false))
		if err != nil {
			log.Fatal(err)
		}
		return
	default:
		log.Fatal("unknown action")
	}
	if err != nil {
		log.Fatal(err)
	}
	printGroupGroups(value.Resource)
}
