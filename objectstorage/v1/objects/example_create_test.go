package objects_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func showPhaseCreate(name string, phase *objects.ObjectCreatePhaseResult) {
	if phase == nil {
		return
	}
	for _, attempt := range phase.Attempts {
		if attempt.Response != nil {
			fmt.Printf("%s logical=%d physical=%d status=%d response-bytes=%d error=%v\n",
				name, attempt.LogicalAttempt, attempt.PhysicalAttempt,
				attempt.Response.StatusCode, len(attempt.Response.Body), attempt.Error)
		} else {
			fmt.Printf("%s logical=%d physical=%d transport-error=%v\n",
				name, attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
		}
	}
}

func showCreateCreate(result *objects.CreateObjectResult) {
	if result == nil {
		return
	}
	fmt.Printf("source=%s mode=%s size=%d skipped=%t prefix=%q manifest-ambiguous=%t\n",
		result.Source, result.Mode, result.Size, result.Skipped,
		result.SegmentPrefix, result.ManifestAmbiguous)
	if result.Capabilities != nil {
		fmt.Printf("segment-size=%d fallback=%t\n",
			result.Capabilities.Size, result.Capabilities.UsedFallback)
	}
	if result.Discovery != nil {
		fmt.Println("stale HEAD", result.Discovery.StatusCode)
	}
	showPhaseCreate("ordinary PUT", result.Ordinary)
	for _, segment := range result.Segments {
		fmt.Printf("segment=%q offset=%d size=%d created=%t ambiguous=%t\n",
			segment.Name, segment.Offset, segment.Size, segment.Created, segment.Ambiguous)
		showPhaseCreate("segment PUT", segment.Upload)
	}
	showPhaseCreate("manifest PUT", result.Manifest)
	for _, cleanup := range result.Cleanup {
		fmt.Println("cleanup target", cleanup.Name)
		showPhaseCreate("cleanup DELETE", cleanup.Deletion)
	}
}

func showErrorCreate(err error) {
	var proof *resource.ResponseError
	if errors.As(err, &proof) {
		fmt.Println("response error", proof.StatusCode, "bytes", len(proof.Body))
	}
	var unconfirmed *objects.ObjectCreateUnconfirmedSegmentError
	if errors.As(err, &unconfirmed) {
		fmt.Println("unconfirmed segment", unconfirmed.Name)
	}
}

func uploadExamplesCreate(ctx context.Context, conn *sdk.Connection,
	filename, knownMD5, knownSHA256 string) error {
	service, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		return err
	}
	options := []objects.CreateObjectOption{
		objects.WithCreateObjectOpts(objects.CreateObjectOpts{
			Headers: map[string]string{"Content-Type": "application/octet-stream"},
		}),
		request.Header("X-Trans-Id-Extra", "sdk-upload"),
		objects.WithCreateObjectHeaders(map[string]string{"Cache-Control": "private"}),
		objects.WithCreateObjectMetadata(map[string]string{"owner": "sdk"}),
		objects.WithCreateObjectMetadataValue("empty", ""),
		request.Custom(func(c *objects.CreateObjectOpts) error { c.MD5 = knownMD5; return nil }),
		request.Custom(func(c *objects.CreateObjectOpts) error { c.SHA256 = knownSHA256; return nil }),
		request.Custom(func(c *objects.CreateObjectOpts) error { c.SegmentSize = nil; return nil }),
		request.Custom(func(c *objects.CreateObjectOpts) error { c.UseSLO = nil; return nil }),
		request.Custom(func(c *objects.CreateObjectOpts) error { c.GenerateChecksums = nil; return nil }),
	}
	result, err := service.Objects.CreateObject(ctx, "backups", "empty ?.bin",
		objects.CreateObjectInput{Data: []byte{}}, options...)
	showCreateCreate(result)
	if err != nil {
		showErrorCreate(err)
		return err
	}
	stale, err := service.Objects.IsObjectStale(ctx, "backups", "archive ?.tar", filename,
		objects.WithIsObjectStaleOpts(objects.IsObjectStaleOpts{}),
		request.Header("X-Trace", "stale"),
		objects.WithIsObjectStaleHeaders(map[string]string{"X-Trans-Id-Extra": "sdk-stale"}),
		request.Custom(func(c *objects.IsObjectStaleOpts) error { c.MD5 = knownMD5; return nil }),
		request.Custom(func(c *objects.IsObjectStaleOpts) error { c.SHA256 = knownSHA256; return nil }))
	if stale != nil {
		if stale.Discovery != nil {
			fmt.Println("standalone stale HEAD", stale.Discovery.StatusCode)
		}
		if stale.Stale != nil {
			fmt.Println("observed stale", *stale.Stale)
		}
	}
	if err != nil {
		showErrorCreate(err)
		return err
	}
	result, err = service.Objects.CreateObject(ctx, "backups", "archive ?.tar",
		objects.CreateObjectInput{Filename: filename}, options...)
	showCreateCreate(result)
	if err != nil {
		showErrorCreate(err)
		return err
	}
	readerOptions := append([]objects.CreateObjectOption{}, options...)
	readerOptions = append(readerOptions,
		request.Custom(func(c *objects.CreateObjectOpts) error { c.MD5 = ""; return nil }),
		request.Custom(func(c *objects.CreateObjectOpts) error { c.SHA256 = ""; return nil }),
		request.Custom(func(c *objects.CreateObjectOpts) error { v := int64(1 << 30); c.SegmentSize = &v; return nil }),
		objects.WithCreateObjectUseSLO(false),
		objects.WithCreateObjectGenerateChecksums(false))
	result, err = service.Objects.CreateObject(ctx, "backups", "stream ?.bin",
		objects.CreateObjectInput{Reader: bytes.NewReader([]byte("reader contents"))},
		readerOptions...)
	showCreateCreate(result)
	if err != nil {
		showErrorCreate(err)
	}
	return err
}

func Example_create() {}
