# Create Import

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
)

func createAndWait(ctx context.Context, client *gophercloud.ServiceClient, data io.Reader, size int64) (*image.CreateImportResult, error) {
    return image.New(client).CreateAndImport(ctx,
        image.CreateAndImportRequest{Name: "worker", Data: data},
        image.WithCreateImportStage(imagedata.StageOpts{Size: &size}),
        image.WithCreateImportWait(),
    )
}
```

```python
image = conn.image.create_image(
    "worker", data=data, disk_format="qcow2", container_format="bare",
    visibility="private", use_import=True, import_method="glance-direct",
    allow_duplicates=True, disable_vendor_agent=False, size=size)
```

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
)

func createFromWeb(ctx context.Context, client *gophercloud.ServiceClient, uri string) (*image.CreateImportResult, error) {
    requireAll := false
    return image.New(client).CreateAndImport(ctx,
        image.CreateAndImportRequest{Name: "worker-from-web"},
        image.WithCreateImportImport(imageimport.ImportOpts{
            Method: "web-download", URI: uri,
            Stores: []string{"fast", "reliable"}, AllStoresMustSucceed: &requireAll,
        }),
    )
}
```

```python
image = conn.image.create_image(
    "worker-from-web", use_import=True, import_method="web-download", uri=uri,
    disk_format="qcow2", container_format="bare", visibility="private",
    stores=["fast", "reliable"], all_stores_must_succeed=False,
    allow_duplicates=True, disable_vendor_agent=False)
```

## References

- [Proxy.create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L438)
- [_upload_image_put](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L727-L822)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html)
- [Stage](v2/imagedata/README.md)
- [Import](v2/imageimport/README.md)
- [Service.Upload](README.md)
- [Create Import Contracts Test](create_import_contracts_test.go)
- [Create Import Core Test](create_import_core_test.go)
- [Create Import Options Test](create_import_options_test.go)
- [Stage Prepare Test](v2/imagedata/stage_prepare_test.go)
- [Import Prepare Test](v2/imageimport/import_prepare_test.go)
