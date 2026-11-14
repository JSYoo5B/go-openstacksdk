package v3_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_groupReplication() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	id := flag.String("group", "", "literal group ID")
	action := flag.String("action", "enable", "enable/disable/failover")
	version := flag.String("microversion", "3.38", "selected Cinder version")
	allow := flag.Bool("allow-attached-volume", false, "allow attached volumes")
	backend := flag.String("secondary-backend-id", "", "optional backend ID")
	sourceWire := flag.Bool("pinned-source-wire", false, "replay the fixed Source body")
	flag.Parse()
	backendPresent := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "secondary-backend-id" {
			backendPresent = true
		}
	})
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
	var value *groups.ReplicationRecordResult
	switch *action {
	case "enable":
		value, err = service.Groups.EnableReplicationRecord(ctx, *id,
			request.Custom(func(c *groups.RecordReplicationOpts) error { c.PinnedSourceWire = *sourceWire; return nil }))
	case "disable":
		value, err = service.Groups.DisableReplicationRecord(ctx, *id,
			request.Custom(func(c *groups.RecordReplicationOpts) error { c.PinnedSourceWire = *sourceWire; return nil }))
	case "failover":
		options := []groups.RecordFailoverOption{
			request.Custom(func(c *groups.RecordFailoverOpts) error { v := *allow; c.AllowedAttachedVolume = &v; return nil }),
			request.Custom(func(c *groups.RecordFailoverOpts) error { c.PinnedSourceWire = *sourceWire; return nil }),
		}
		if backendPresent {
			options = append(options, request.Custom(func(c *groups.RecordFailoverOpts) error { v := *backend; c.SecondaryBackendID = &v; return nil }))
		}
		value, err = service.Groups.FailoverReplicationRecord(ctx, *id, options...)
	default:
		log.Fatal("unknown action")
	}
	if value != nil {
		fmt.Printf("reference=%q actual_http=%d actual_body=%q discovery=%d\n",
			value.Reference, value.StatusCode, value.Envelope, len(value.Discovery))
	}
	if err != nil {
		log.Fatal(err)
	}
}
