package users_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_groupRecords() {
	groupID := flag.String("group-id", "", "required group UUID")
	flag.Parse()
	if *groupID == "" {
		log.Fatal("-group-id is required")
	}
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for row, err := range service.Users.ListUsersInGroupRecords(ctx, *groupID,
		users.WithGroupRecordListQuery("email", "member@example.com"),
		request.Limit(100),
		request.MaxItems(100),
	) {
		if err != nil {
			log.Fatal(err)
		}
		data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", data)
	}
}
