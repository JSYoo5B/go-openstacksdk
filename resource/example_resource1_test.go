package resource_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func PrintFirstPage[T any](ctx context.Context, collection *resource.Collection[T]) error {
	for value, err := range collection.List(ctx,
		resource.WithMaxItems(5), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(value)
	}
	return nil
}
