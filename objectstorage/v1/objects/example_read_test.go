package objects_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func readObjectsRead(ctx context.Context, api *objects.API, etag string, asOf time.Time, version string) error {
	common := []objects.ObjectReadOption{
		objects.WithObjectReadOpts(objects.ObjectReadOpts{
			Headers:           map[string]string{"X-Audit": "initial"},
			IfModifiedSince:   &asOf,
			IfUnmodifiedSince: &asOf,
		}),
		objects.WithObjectReadHeaders(map[string]string{"X-Audit": "read"}),
		request.Header("X-Audit", "read-final"),
		request.Custom(func(c *objects.ObjectReadOpts) error { v := false; c.Newest = &v; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.BufferSize = 32 * 1024; return nil }),
	}
	getOpts := append(append([]objects.ObjectReadOption(nil), common...),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.IfUnmodifiedSince = nil; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { v := asOf; c.IfModifiedSince = &v; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.IfNoneMatch = etag; return nil }),
	)
	buffered, err := api.GetObject(ctx, "backups", "report.txt", getOpts...)
	if buffered != nil {
		fmt.Printf("GET status=%d bytes=%d not-modified=%t complete=%t\n",
			buffered.StatusCode, len(buffered.Body), buffered.NotModified, buffered.Complete)
	}
	if err != nil {
		return err
	}

	copyOpts := append(append([]objects.ObjectReadOption(nil), common...),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.IfModifiedSince = nil; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { v := asOf; c.IfUnmodifiedSince = &v; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.IfMatch = etag; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.Range = "bytes=0-1023"; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.Filename = "report.part"; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.VersionID = version; return nil }),
	)
	var output bytes.Buffer
	copied, err := api.DownloadObject(ctx, "backups", "report.txt", &output, copyOpts...)
	if copied != nil {
		fmt.Printf("copied=%d complete=%t\n", copied.BytesWritten, copied.Complete)
	}
	if err != nil {
		return err
	}

	plain := append(append([]objects.ObjectReadOption(nil), common...),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.IfModifiedSince = nil; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.IfUnmodifiedSince = nil; return nil }),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.Newest = nil; return nil }),
	)
	manifestOpts := append(append([]objects.ObjectReadOption(nil), plain...),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.MultipartManifest = "get"; return nil }),
	)
	streamed, err := api.StreamObject(ctx, "backups", "manifest", manifestOpts...)
	if err != nil {
		return err
	}
	_, readErr := io.Copy(io.Discard, streamed.Body)
	closeErr := streamed.Body.Close()
	fmt.Printf("stream bytes=%d complete=%t\n", streamed.BytesRead, streamed.Complete)
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}

	linkOpts := append(append([]objects.ObjectReadOption(nil), plain...),
		request.Custom(func(c *objects.ObjectReadOpts) error { c.Symlink = "get"; return nil }),
	)
	_, err = api.GetObject(ctx, "backups", "report-link", linkOpts...)
	return err
}

func Example_read() {}
