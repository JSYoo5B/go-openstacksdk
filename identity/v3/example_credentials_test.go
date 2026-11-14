package v3_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func printRecordCredentials(v *credentials.CredentialRecord) {
	if v == nil {
		fmt.Println("null")
		return
	}
	data, err := json.MarshalIndent(v.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}

func Example_credentials() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	user := flag.String("user", "", "credential owner user ID")
	project := flag.String("project", "", "credential project ID")
	kind := flag.String("type", "ec2", "credential blob type")
	id := flag.String("id", "", "credential ID to GET and find")
	blob := flag.String("blob", "{}", "credential blob string")
	create := flag.Bool("create", false, "create a generic Credential")
	remove := flag.String("delete", "", "credential ID to delete")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	api := service.Credentials
	for v, err := range api.ListRecords(ctx,
		credentials.WithRecordListType(*kind),
		credentials.WithRecordListUserID(*user),
		request.MaxItems(100)) {
		if err != nil {
			log.Fatal(err)
		}
		printRecordCredentials(v)
	}
	if *id != "" {
		v, err := api.GetRecord(ctx, *id)
		if err != nil {
			log.Fatal(err)
		}
		printRecordCredentials(v)
		found, err := api.FindRecord(ctx, *id)
		if err != nil {
			log.Fatal(err)
		}
		printRecordCredentials(found)
	}
	if *create {
		v, err := api.CreateRecord(ctx,
			request.Field("user_id", *user),
			request.Field("project_id", *project),
			request.Field("type", *kind),
			request.Field("blob", *blob))
		if err != nil {
			log.Fatal(err)
		}
		printRecordCredentials(v)
	}
	if *remove != "" {
		if err := api.DeleteRecord(ctx, *remove,
			request.IgnoreMissing(false)); err != nil {
			log.Fatal(err)
		}
	}
}
