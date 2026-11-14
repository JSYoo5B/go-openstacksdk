package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_backupFind() {
	identity := flag.String("backup", "nightly", "Backup name or ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud("source"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	backup, err := service.Backups.FindRecord(ctx, *identity,
		request.IgnoreMissing(false),
		request.Custom(func(c *backups.RecordFindOpts) error { v := false; c.Details = &v; return nil }),
	)
	if err != nil {
		log.Fatal(err)
	}
	body, err := json.MarshalIndent(backup.Resource.Body, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(body))
}
