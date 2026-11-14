package v1_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func FindSenlinResourcesFinding1(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.Clustering(ctx)
	if err != nil {
		return err
	}
	profile, err := service.Profiles.FindIdentity(ctx, "worker_template")
	if err != nil {
		return err
	}
	policy, err := service.Policies.FindIdentity(ctx, "placement")
	if err != nil {
		return err
	}
	cluster, err := service.Clusters.FindIdentity(ctx, "workers")
	if err != nil {
		return err
	}
	node, err := service.Nodes.FindIdentity(ctx, "worker_1")
	if err != nil {
		return err
	}
	receiver, err := service.Receivers.FindIdentity(ctx, "scale_out")
	if err != nil {
		return err
	}
	if profile != nil {
		fmt.Println("profile", profile.ID)
	}
	if policy != nil {
		fmt.Println("policy", policy.ID)
	}
	if cluster != nil {
		fmt.Println("cluster", cluster.ID)
	}
	if node != nil {
		fmt.Println("node", node.ID)
	}
	if receiver != nil {
		fmt.Println("receiver", receiver.ID)
	}
	return nil
}
