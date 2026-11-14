package objectstorage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func Example_cloudContainerSearch() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	matched, err := conn.SearchContainers(ctx,
		func(c *objectstorage.ContainerSearchOpts) error { c.Name = "logs-*"; return nil },
		objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"count":1}`)))
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range matched.Records {
		fmt.Printf("name=%s page_status=%d\n", row.Resource.Body["name"], row.StatusCode)
	}
	names, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("[].name"))
	if err != nil {
		log.Fatal(err)
	}
	total, err := conn.SearchContainers(ctx, objectstorage.WithContainerSearchExpression("length(@)"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
