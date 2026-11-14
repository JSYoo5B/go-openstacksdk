package objectstorage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func Example_cloudObjectSearch() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	matched, err := conn.SearchObjects(ctx, "assets",
		func(c *objectstorage.ObjectSearchOpts) error { c.Name = "logs-*"; return nil },
		objectstorage.WithObjectSearchFiltersJSON(json.RawMessage(`{"content_length":5}`)))
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range matched.Records {
		fmt.Printf("name=%s bytes=%s page_status=%d\n",
			row.Resource.Body["name"], row.Resource.Body["content_length"], row.StatusCode)
	}
	names, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("[].name"))
	if err != nil {
		log.Fatal(err)
	}
	total, err := conn.SearchObjects(ctx, "assets", objectstorage.WithObjectSearchExpression("length(@)"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("names=%s total=%s\n", names.Value, total.Value)
}
