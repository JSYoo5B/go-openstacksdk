package objects_test

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func showMarkerDirectoryMarker(result *objects.CreateObjectResult, err error) {
	if result != nil {
		fmt.Printf("source=%s mode=%s size=%d md5=%q sha256=%q\n",
			result.Source, result.Mode, result.Size, result.MD5, result.SHA256)
		if result.Ordinary != nil {
			for _, attempt := range result.Ordinary.Attempts {
				fmt.Printf("logical=%d physical=%d error=%v\n",
					attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
				if attempt.Response != nil {
					fmt.Printf("HTTP%d headers=%v body=%q\n",
						attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
				}
			}
			if result.Ordinary.Acknowledgement != nil {
				fmt.Println("acknowledged HTTP", result.Ordinary.Acknowledgement.StatusCode)
			}
		}
	}
	if err != nil {
		fmt.Println(err)
		var proof *resource.ResponseError
		if errors.As(err, &proof) {
			fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
				proof.StatusCode, proof.Header, proof.Body)
		}
	}
}

func markerExampleDirectoryMarker(ctx context.Context, conn *sdk.Connection) {
	service, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		showMarkerDirectoryMarker(nil, err)
		return
	}
	options := []objects.DirectoryMarkerOption{
		objects.WithDirectoryMarkerOpts(objects.DirectoryMarkerOpts{
			Headers:  map[string]string{"X-Client": "directory-marker"},
			Metadata: map[string]string{"Owner": "sdk"},
		}),
		request.Header("X-Trace", "initial"),
		objects.WithDirectoryMarkerHeaders(map[string]string{"x-trace": "marker"}),
		objects.WithDirectoryMarkerMetadata(map[string]string{"Purpose": "web-listing"}),
		objects.WithDirectoryMarkerMetadataValue("Build", "2026-10-04"),
	}
	result, err := service.Objects.CreateDirectoryMarkerObject(ctx, "backups", "reports/", options...)
	showMarkerDirectoryMarker(result, err)
}

func Example_directoryMarker() {}
