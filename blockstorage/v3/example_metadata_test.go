package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func showMetadata(kind string, declared *resource.RawResource, envelope json.RawMessage, status int) {
	metadata := "<no declared view>"
	if declared != nil {
		metadata = string(declared.Body["metadata"])
	}
	fmt.Printf("%s metadata=%s status=%d actual_body=%q\n", kind, metadata, status, envelope)
}

func Example_metadata() {
	volumeID := flag.String("volume", "", "existing volume ID")
	snapshotID := flag.String("snapshot", "", "existing snapshot ID")
	action := flag.String("action", "fetch", "fetch, get, set, delete, clear, or noop")
	metadataJSON := flag.String("metadata", `{"sdk-example":"go"}`, "metadata JSON object")
	key := flag.String("key", "sdk-example", "one metadata key to delete")
	flag.Parse()
	if *volumeID == "" && *snapshotID == "" {
		log.Fatal("provide -volume or -snapshot")
	}
	switch *action {
	case "fetch", "get", "set", "delete", "clear", "noop":
	default:
		log.Fatal("unknown action")
	}
	// RawMessage values preserve JSON null/false/zero and large integers.
	var members map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*metadataJSON), &members); err != nil {
		log.Fatal(err)
	}
	values := make(map[string]any, len(members))
	for name, raw := range members {
		values[name] = raw
	}
	var keys []string
	if *action == "noop" {
		keys = []string{}
	}
	if *action == "delete" {
		keys = []string{*key}
	}
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if *volumeID != "" {
		var record *volumes.VolumeResourceRecord
		switch *action {
		case "fetch":
			record, err = service.Volumes.FetchMetadataRecord(ctx, *volumeID)
		case "get":
			record, err = service.Volumes.GetMetadataRecord(ctx, *volumeID)
		case "set":
			record, err = service.Volumes.SetMetadataRecord(ctx, *volumeID, values,
				request.Header("X-Client-Request-Id", "metadata-volume"))
		default:
			var deleted *volumes.MetadataRecordDeletion
			deleted, err = service.Volumes.DeleteMetadataRecord(ctx, *volumeID, keys)
			if deleted != nil {
				record = deleted.Record
				fmt.Printf("volume clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
			}
		}
		if record != nil {
			showMetadata("volume", record.Resource, record.Envelope, record.StatusCode)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	if *snapshotID != "" {
		var record *snapshots.SnapshotRecord
		switch *action {
		case "fetch":
			record, err = service.Snapshots.FetchMetadataRecord(ctx, *snapshotID)
		case "get":
			record, err = service.Snapshots.GetMetadataRecord(ctx, *snapshotID)
		case "set":
			record, err = service.Snapshots.SetMetadataRecord(ctx, *snapshotID, values,
				request.Header("X-Client-Request-Id", "metadata-snapshot"))
		default:
			var deleted *snapshots.MetadataRecordDeletion
			deleted, err = service.Snapshots.DeleteMetadataRecord(ctx, *snapshotID, keys)
			if deleted != nil {
				record = deleted.Record
				fmt.Printf("snapshot clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
			}
		}
		if record != nil {
			showMetadata("snapshot", record.Resource, record.Envelope, record.StatusCode)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}
