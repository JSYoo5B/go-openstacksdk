package policies_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
)

func Example_policiesValidate() {
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
	service, err = conn.Clustering(ctx)
	if err != nil {
		log.Fatal(err)
	}
	checked, err := service.Policies.Validate(ctx, policies.ValidateOpts{},
		policies.WithValidateSpec(map[string]any{
			"type": "senlin.policy.scaling", "version": "1.0",
			"properties": map[string]any{
				"event":      "CLUSTER_SCALE_OUT",
				"adjustment": map[string]any{"type": "CHANGE_IN_CAPACITY", "number": 1},
			},
		}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(checked.Spec, checked.Body["id"])
}
