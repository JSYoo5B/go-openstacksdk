package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func showConsistencyGroups(value *consistencygroups.ConsistencyGroupRecord) {
	if value == nil {
		fmt.Println("null")
		return
	}
	raw, err := json.MarshalIndent(value.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(raw))
}

func Example_consistencyGroups() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	version := flag.String("microversion", "", "optional selected Cinder version")
	action := flag.String("action", "list", "list/get/find/create/clone/update/delete")
	id := flag.String("group", "", "literal ID, or exact identity for find")
	name := flag.String("name", "sample", "name for create/clone/update")
	volumeTypes := flag.String("volume-types", "", "comma-separated legacy volume types")
	sourceGroup := flag.String("source-group", "", "source consistency-group ID")
	sourceSnapshot := flag.String("source-snapshot", "", "source consistency-group-snapshot ID")
	summary := flag.Bool("summary", false, "summary list/find")
	strict := flag.Bool("strict", false, "strict find missing")
	force := flag.Bool("force", false, "also request deletion of member volumes")
	sourceWire := flag.Bool("pinned-source-wire", false, "replay known Source delete spelling")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	policy := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
	if *version != "" {
		policy = append(policy, sdk.WithMicroversion(sdk.BlockStorage, *version))
	}
	conn, err := sdk.Connect(ctx, policy...)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := consistencygroups.New(service.RawClient())
	var value *consistencygroups.ConsistencyGroupRecord
	switch *action {
	case "list":
		for row, err := range api.ListRecords(ctx, request.Custom(func(c *consistencygroups.RecordListOpts) error { v := !*summary; c.Details = &v; return nil })) {
			if err != nil {
				log.Fatal(err)
			}
			showConsistencyGroups(row)
		}
		return
	case "get":
		value, err = api.GetRecord(ctx, *id)
	case "find":
		value, err = api.FindRecord(ctx, *id,
			request.IgnoreMissing(!*strict),
			request.Custom(func(c *consistencygroups.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil }))
	case "create":
		value, err = api.CreateRecord(ctx,
			request.Field("volume_types", *volumeTypes),
			request.Field("name", *name))
	case "clone":
		value, err = api.CreateFromSourceRecord(ctx,
			request.Custom(func(c *consistencygroups.RecordSourceOpts) error {
				v := *sourceGroup
				c.ConsistencyGroupID = &v
				return nil
			}),
			request.Custom(func(c *consistencygroups.RecordSourceOpts) error {
				v := *sourceSnapshot
				c.ConsistencyGroupSnapshotID = &v
				return nil
			}),
			request.Custom(func(c *consistencygroups.RecordSourceOpts) error { v := *name; c.Name = &v; return nil }))
	case "update":
		value, err = api.UpdateRecord(ctx, *id, request.Field("name", *name))
	case "delete":
		err = api.DeleteRecord(ctx, *id, request.Custom(func(c *consistencygroups.RecordDeleteOpts) error { v := *force; c.Force = &v; return nil }),
			request.Custom(func(c *consistencygroups.RecordDeleteOpts) error { c.PinnedSourceWire = *sourceWire; return nil }))
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
	showConsistencyGroups(value)
}
