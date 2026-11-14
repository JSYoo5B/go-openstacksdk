package users_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_recordFind() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	identity := flag.String("identity", "", "User exact name or ID")
	domain := flag.String("domain", "", "explicit domain ID")
	flag.Parse()
	if *identity == "" || *domain == "" {
		log.Fatal("identity and domain are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	row, err := service.Users.FindRecord(ctx, *identity,
		request.IgnoreMissing(false),
		request.Query("domain_id", *domain),
		request.Query("skip_cache", true),
		request.Query("error_message", "User lookup failed"))
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(row.Resource, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
