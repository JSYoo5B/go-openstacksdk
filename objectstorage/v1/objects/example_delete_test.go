package objects_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func showDeleteDelete(result *objects.DeleteObjectResult) {
	if result == nil {
		return
	}
	fmt.Println("ignored missing:", result.IgnoredMissing)
	if result.StaticLargeObject == nil {
		fmt.Println("SLO flag unknown")
	} else {
		fmt.Println("SLO:", *result.StaticLargeObject)
	}
	for _, phase := range []struct {
		name     string
		response *objects.DeleteObjectResponse
	}{{"HEAD", result.Discovery}, {"DELETE", result.Deletion}} {
		if phase.response != nil {
			fmt.Printf("%s status=%d transaction=%q observed-bytes=%d\n",
				phase.name, phase.response.StatusCode,
				phase.response.Header.Get("X-Trans-Id"), len(phase.response.Body))
		}
	}
	if result.Bulk != nil {
		fmt.Printf("bulk status=%d deleted=%d not-found=%d errors=%d\n",
			result.Bulk.ResponseCode, result.Bulk.NumberDeleted,
			result.Bulk.NumberNotFound, len(result.Bulk.Errors))
	}
}

func showDeleteErrorDelete(err error) {
	var response *resource.ResponseError
	if errors.As(err, &response) {
		fmt.Printf("failed physical status=%d observed-bytes=%d\n",
			response.StatusCode, len(response.Body))
	}
	var bulk *objects.ObjectDeleteBulkError
	if errors.As(err, &bulk) {
		fmt.Printf("embedded bulk failure=%q errors=%d\n",
			bulk.ResponseStatus, len(bulk.Errors))
	}
}

func deleteObjectsDelete(ctx context.Context, api *objects.API, versionID string) error {
	knownSLO, allowMissing, newest := false, false, true
	auto := []objects.DeleteObjectOption{
		objects.WithDeleteObjectOpts(objects.DeleteObjectOpts{
			Headers:           map[string]string{"X-Request-Label": "initial"},
			StaticLargeObject: &knownSLO, IgnoreMissing: &allowMissing, Newest: &newest,
		}),
		objects.WithDeleteObjectHeaders(map[string]string{"X-Trace-Label": "object-delete"}),
		request.Header("X-Request-Label", "auto"),
		request.Custom(func(c *objects.DeleteObjectOpts) error { c.StaticLargeObject = nil; return nil }),
		request.IgnoreMissing(false),
		request.Custom(func(c *objects.DeleteObjectOpts) error { c.IgnoreMissing = nil; return nil }),
		request.Custom(func(c *objects.DeleteObjectOpts) error { c.VersionID = versionID; return nil }),
		objects.WithDeleteObjectNewest(true),
		request.Custom(func(c *objects.DeleteObjectOpts) error { c.Newest = nil; return nil }),
		objects.WithDeleteObjectNewest(false),
	}
	result, err := api.DeleteObject(ctx, "backups", "report.txt", auto...)
	showDeleteDelete(result)
	if err != nil {
		showDeleteErrorDelete(err)
		return err
	}

	result, err = api.DeleteObject(ctx, "backups", "manifest",
		objects.WithDeleteObjectStaticLargeObject(true),
		request.IgnoreMissing(false),
	)
	showDeleteDelete(result)
	if err != nil {
		showDeleteErrorDelete(err)
	}
	return err
}

func Example_delete() {}
