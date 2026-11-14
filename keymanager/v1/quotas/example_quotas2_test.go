package quotas_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ConfigureQuotas2(ctx context.Context, client *gophercloud.ServiceClient, projectID string) error {
	api := quotas.New(client)
	effective, err := api.Get(ctx)
	if err != nil {
		return err
	}
	fmt.Println(string(effective.Secrets))

	project, err := api.InProject(ctx, resource.ID(projectID))
	if err != nil {
		return err
	}
	configured, err := project.Get(ctx)
	if err != nil {
		return err
	}
	fmt.Println(string(configured.Orders))

	accepted, err := project.Update(ctx, quotas.UpdateOpts{
		Secrets: request.Present[int64](100),
	}, request.Custom(func(c *request.Config[quotas.UpdateOpts]) error {
		c.Options.Orders = request.Present(int64(0))
		return nil
	}), request.Custom(func(c *request.Config[quotas.UpdateOpts]) error {
		c.Options.Containers = request.Present(int64(-1))
		return nil
	}),
		request.Header("X-Request-ID", "quota-change"))
	if err != nil {
		return err
	}
	fmt.Println(accepted.StatusCode, accepted.Header.Get("X-Request-ID"))

	return project.Delete(ctx, request.IgnoreMissing(false))
}
