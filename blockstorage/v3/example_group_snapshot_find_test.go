package v3_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_groupSnapshotFind() {
	identity := flag.String("identity", "group-snapshot-id", "group snapshot ID or exact name")
	strict := flag.Bool("strict", false, "report completed search absence as an error")
	summary := flag.Bool("summary", false, "use summary list only on fallback")
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
	value, err := api.FindRecord(ctx, *identity,
		request.IgnoreMissing(!*strict),
		request.Custom(func(c *groupsnapshots.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil }))
	if err != nil {
		if value != nil {
			fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
		}
		if errors.Is(err, resource.ErrNotFound) {
			log.Fatal("completed search found no group snapshot")
		}
		if errors.Is(err, resource.ErrAmbiguous) {
			log.Fatal("more than one ID/name match")
		}
		log.Fatal(err)
	}
	if value == nil {
		fmt.Println("no matching group snapshot")
		return
	}
	fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
	fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
}
