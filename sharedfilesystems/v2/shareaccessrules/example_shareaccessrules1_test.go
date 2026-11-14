package shareaccessrules_test

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
)

func Example_shareaccessrules1() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx,
		sdk.WithMicroversionRange(sdk.SharedFileSystem, "2.45", "2.82"),
	)
	if err != nil {
		panic(err)
	}
	service, err := conn.SharedFileSystemV2(ctx)
	if err != nil {
		panic(err)
	}
	rules, err := service.ShareAccessRules.InShare(ctx, resource.Name("team-data"))
	if err != nil {
		panic(err)
	}

	created, err := rules.Allow(ctx, shareaccessrules.AllowOpts{
		AccessType: "ip",
		AccessTo:   "192.0.2.0/24",
	}, request.Custom(func(c *request.Config[shareaccessrules.AllowOpts]) error { c.Options.AccessLevel = "ro"; return nil }),
		shareaccessrules.WithMetadata(map[string]string{"owner": "ops"}))
	if err != nil {
		// The action may have succeeded before response decoding failed.
		// A non-nil created value retains any known ID and raw response.
		fmt.Println(created)
		panic(err)
	}
	ready, err := rules.Wait(ctx, resource.ID(created.ID), "active",
		resource.WithTimeout(2*time.Minute),
		resource.WithPollInterval(time.Second))
	if err != nil {
		panic(err)
	}
	fmt.Println(ready.ID, ready.State)

	for rule, err := range rules.List(ctx) {
		if err != nil {
			panic(err)
		}
		fmt.Println(rule.ID, rule.AccessTo, rule.State)
	}
	if err := rules.Delete(ctx, resource.ID(created.ID)); err != nil {
		panic(err)
	}
	if err := rules.WaitDeleted(ctx, resource.ID(created.ID)); err != nil {
		panic(err)
	}
}
