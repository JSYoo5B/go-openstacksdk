package tags_test

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/tags"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_tags() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx,
		sdk.WithMicroversionRange(sdk.Compute, "2.26", "2.100"),
	)
	if err != nil {
		panic(err)
	}
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		panic(err)
	}
	serverTags, err := service.Tags.InServer(ctx, resource.Name("web.prod"))
	if err != nil {
		panic(err)
	}

	if err := serverTags.Add(ctx, "managed"); err != nil {
		panic(err)
	}
	present, err := serverTags.Check(ctx, "managed")
	if err != nil {
		panic(err)
	}
	fmt.Println(serverTags.ServerID(), present)
	values, err := serverTags.List(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
	values, err = serverTags.Replace(ctx, []string{"managed", "role=db"})
	if err != nil {
		panic(err)
	}
	fmt.Println(values)
	if err := serverTags.Remove(ctx, "managed", tags.WithMissingError()); err != nil {
		panic(err)
	}
	if err := serverTags.RemoveAll(ctx); err != nil {
		panic(err)
	}
}
