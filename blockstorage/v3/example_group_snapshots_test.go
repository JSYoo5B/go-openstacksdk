package v3_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func showGroupSnapshots(value *groupsnapshots.GroupSnapshotRecord) {
	if value == nil {
		return
	}
	if value.Resource != nil {
		fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
	}
	fmt.Printf("actual_http=%d discovery=%d actual_body=%q\n", value.StatusCode, len(value.Discovery), value.Envelope)
}

func Example_groupSnapshots() {
	action := flag.String("action", "get", "get, list, create, or delete")
	id := flag.String("id", "group-snapshot-id", "existing group snapshot ID")
	group := flag.String("group", "group-id", "existing source group ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	cinder, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := groupsnapshots.New(cinder.RawClient())
	switch *action {
	case "get":
		value, err := api.GetRecord(ctx, *id)
		showGroupSnapshots(value)
		if err != nil {
			log.Fatal(err)
		}
	case "list":
		for value, err := range api.ListRecords(ctx,
			request.Custom(func(c *groupsnapshots.RecordListOpts) error { v := false; c.Details = &v; return nil }),
			request.Query("status", "available"),
			request.Limit(10)) {
			if err != nil {
				log.Fatal(err)
			}
			showGroupSnapshots(value)
		}
	case "create":
		value, err := api.CreateRecord(ctx, map[string]any{"group_id": *group, "name": "sdk-example"})
		showGroupSnapshots(value)
		if err != nil {
			log.Fatal(err)
		}
	case "delete":
		receipt, err := api.DeleteRecord(ctx, *id, request.IgnoreMissing(false))
		if receipt != nil {
			fmt.Printf("actual_http=%d ignored404=%t actual_body=%q\n", receipt.StatusCode, receipt.IgnoredMissing, receipt.Envelope)
		}
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unknown action")
	}
}
