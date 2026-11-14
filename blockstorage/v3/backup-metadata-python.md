# Backup Metadata Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/gophercloud/gophercloud/v2"
)

func fetchBackupMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
    profile := backups.New(client).PythonMetadata()
    result, err := profile.FetchMetadata(ctx, id,
        request.Header("X-Request-Id", "application-request"))
    if err != nil {
        return nil, err
    }
    var values map[string]string
    if result.Metadata != nil {
        if err := json.Unmarshal(result.Metadata, &values); err != nil {
            return nil, err
        }
    }
    return values, nil
}
```

## References

- [v3 router](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/router.py#L183-L187)
- [Cinder Metadata Server Contracts](../../docs/cinder-metadata-server-contracts.md)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L27)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1772-L1867)
- [metadata_python_test.go](backups/metadata_python_test.go)
- [metadata.go](backups/metadata.go)
