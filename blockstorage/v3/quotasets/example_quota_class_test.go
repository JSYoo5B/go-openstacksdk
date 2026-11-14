package quotasets_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_quotaClass() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	action := flag.String("action", "", "get or update")
	name := flag.String("name", "default", "quota class name")
	attrs := flag.String("attrs", `{"volumes":0}`, "update JSON attributes")
	version := flag.String("microversion", "", "explicit service version, empty uses configured/default policy")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	options := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
	if *version != "" {
		options = append(options, sdk.WithMicroversion(sdk.BlockStorage, *version))
	}
	conn, err := sdk.Connect(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.BlockStorageV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := quotasets.New(service.RawClient())
	var row *quotasets.QuotaClassRecord
	switch *action {
	case "get":
		row, err = api.GetQuotaClassRecord(ctx, request.Custom(func(c *quotasets.QuotaClassGetOpts) error { v := *name; c.Name = &v; return nil }))
	case "update":
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(*attrs), &fields); err != nil {
			log.Fatal(err)
		}
		if fields == nil {
			log.Fatal("attrs must be a JSON object")
		}
		row, err = api.UpdateQuotaClassRecord(ctx, *name,
			quotasets.WithQuotaClassUpdateOpts(quotasets.QuotaClassUpdateOpts{Attributes: fields}))
	default:
		log.Fatal("action must be get or update")
	}
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(row.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
