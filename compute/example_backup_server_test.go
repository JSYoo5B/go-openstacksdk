package compute_test

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_backupServer() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	id := flag.String("id", "", "literal server ID")
	name := flag.String("name", "backup", "backup image name")
	kind := flag.String("type", "daily", "backup type")
	rotation := flag.Int("rotation", 3, "backups retained by Nova")
	wait := flag.Bool("wait", false, "wait for Glance active status")
	timeout := flag.Duration("timeout", 120*time.Second, "image wait timeout")
	flag.Parse()
	if *id == "" {
		log.Fatal("-id is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.96"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.BackupServer(ctx, *id,
		compute.BackupServerRequest{Name: *name, BackupType: *kind, Rotation: *rotation},
		compute.WithBackupServerWait(*wait), compute.WithBackupServerWaitOptions(resource.WithTimeout(*timeout)))
	if result != nil && result.Action != nil {
		log.Printf("Nova action status=%d image_id=%q operation_version=%q",
			result.Action.StatusCode, result.Action.ImageID, result.Action.Microversion)
	}
	if err != nil {
		log.Fatal(err)
	}
	if result.Image == nil {
		log.Print("Glance image not found")
		return
	}
	data, err := json.MarshalIndent(result.Image.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(data))
}
