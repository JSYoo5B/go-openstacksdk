package defaultstatefulness_test

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func networkV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	api := defaultstatefulness.New(networkV2(ctx).RawClient())
	projectID := "project-id"
	setting, err := api.Create(ctx, request.Field("project_id", projectID), request.Field("stateful", false))
	if err != nil {
		log.Fatal(err)
	}
	id := setting.Text("id")
	var revision int64
	if err := json.Unmarshal(setting.Resource.Body["revision_number"], &revision); err != nil {
		log.Fatal(err)
	}
	if _, err := api.Update(ctx, id, request.Field("stateful", true), defaultstatefulness.WithUpdateRevision(revision)); err != nil {
		log.Fatal(err)
	}
	for item, err := range api.List(ctx, request.Query("project_id", projectID)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Text("id"), string(item.Resource.Body["stateful"]))
	}
	if _, err := api.Delete(ctx, id); err != nil {
		log.Fatal(err)
	}
}
