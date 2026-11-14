package clusters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_metadataFragment2() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, err := service.Clusters.SetMetadata(ctx, resource.ID("CLUSTER_ID"),
		map[string]any{
			"team":     "infra",
			"nullable": nil,
			"big":      json.Number("9007199254740993"),
		}, request.Header("X-Request-Id", "metadata-change"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Changed, string(result.Previous["team"]), string(result.Requested["team"]))
	if result.Operation != nil {
		fmt.Println(result.Operation.ActionID)
	}
}
