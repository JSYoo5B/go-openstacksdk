# Read

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

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

func readObjects(ctx context.Context, api *objects.API, etag string, asOf time.Time, version string) error {
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

func main() {}
```

```python
import openstack

conn = openstack.connect()

remembered = conn.object_store.get_object(
    "report.txt", container="backups", remember_content=True
)
print("remembered text characters:", len(remembered.data))

data = conn.object_store.download_object("report.txt", container="backups")
print("downloaded bytes:", len(data))

chunks = conn.object_store.stream_object(
    "report.txt", container="backups", chunk_size=32 * 1024
)
for chunk in chunks:
    print("chunk bytes:", len(chunk))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288-L396)
- [Swift GET API reference](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L16-L151)
- [Swob](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/swob.py#L1350-L1478)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L103-L171)
- [Read Core Test](read_core_test.go)
- [Read Stream Test](read_stream_test.go)
- [concrete options](read_options_test.go)
- [Read Contracts Test](read_contracts_test.go)
- [Connection Objectstorage Object Read Test](../../../connectiontest/connection_objectstorage_object_read_test.go)
- [Swift Object Read Test](../../../internal/cmd/sdkgen/swift_object_read_test.go)
