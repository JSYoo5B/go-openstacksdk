package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_consistencyGroupSnapshots() {
	operation := flag.String("operation", "list", "get, find, list, create or delete")
	identity := flag.String("identity", "", "literal ID, or exact ID/name for find")
	group := flag.String("group", "", "source consistency group ID for create")
	name := flag.String("name", "", "create name, including an explicit empty string")
	attrsJSON := flag.String("attrs", "{}", "create JSON object")
	queryJSON := flag.String("query", "{}", "list JSON query object")
	strict := flag.Bool("strict", false, "report missing find/delete as an error")
	summary := flag.Bool("summary", false, "list summary, also on find fallback")
	flag.Parse()
	supplied := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
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
	api := consistencygroupsnapshots.New(cinder.RawClient())
	show := func(value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, err error) {
		if value != nil {
			fmt.Printf("actual_http=%d actual_body=%q\n", value.StatusCode, value.Envelope)
		}
		if err != nil {
			log.Fatal(err)
		}
		if value == nil {
			fmt.Println("no matching consistency group snapshot")
			return
		}
		raw, err := json.MarshalIndent(value.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(raw))
	}
	switch *operation {
	case "get":
		show(api.GetRecord(ctx, *identity))
	case "find":
		show(api.FindRecord(ctx, *identity,
			request.IgnoreMissing(!*strict),
			request.Custom(func(c *consistencygroupsnapshots.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil })))
	case "list":
		var query map[string]json.RawMessage
		if err := json.Unmarshal([]byte(*queryJSON), &query); err != nil {
			log.Fatal(err)
		}
		for value, err := range api.ListRecords(ctx,
			consistencygroupsnapshots.WithRecordListOpts(consistencygroupsnapshots.RecordListOpts{Query: query}),
			request.Custom(func(c *consistencygroupsnapshots.RecordListOpts) error { v := !*summary; c.Details = &v; return nil })) {
			show(value, err)
		}
	case "create":
		var attrs map[string]any
		decoder := json.NewDecoder(strings.NewReader(*attrsJSON))
		decoder.UseNumber()
		if err := decoder.Decode(&attrs); err != nil {
			log.Fatal(err)
		}
		if attrs == nil {
			attrs = make(map[string]any)
		}
		if supplied["group"] {
			attrs["consistencygroup_id"] = *group
		}
		if supplied["name"] {
			attrs["name"] = *name
		}
		show(api.CreateRecord(ctx, attrs))
	case "delete":
		receipt, err := api.DeleteRecord(ctx, *identity,
			request.IgnoreMissing(!*strict))
		if receipt != nil {
			fmt.Printf("actual_http=%d actual_body=%q ignored_missing=%t\n",
				receipt.StatusCode, receipt.Envelope, receipt.IgnoredMissing)
		}
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unknown operation: ", *operation)
	}
}
