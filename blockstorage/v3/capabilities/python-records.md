# Backend capability lookup

```python
import openstack

conn = openstack.connect()
capabilities = conn.block_storage.get_capabilities("host@backend")
print(capabilities.properties)
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    blockstorage "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    provider := &gophercloud.ProviderClient{}
    provider.SetToken(os.Getenv("OS_TOKEN"))
    client := &gophercloud.ServiceClient{
        ProviderClient: provider, Type: "volumev3",
        Endpoint: os.Getenv("OS_VOLUME_ENDPOINT"),
    }
    sdk := blockstorage.New(client)
    result, err := sdk.Capabilities.GetRecord(context.Background(), "host@backend")
    if err != nil { panic(err) }
    fmt.Println(string(result.Resource.Body["properties"]))
}
```

The host is a literal escaped URI segment. Only declared fields enter Resource;
Wire retains unknown extensions. Properties uses the shared Source dictionary
projection and replication_targets uses its list projection, preserving literal
JSON in Wire. Missing values and explicit null remain visible. Headers can be
supplied with `capabilities.WithHeaders`; mutable Python Resource inputs and
Requests decoding outside the shared finite JSON profile remain outside this
scalar-host API. A 404 is never converted into an empty capability set. These
examples do not establish live-cloud validation.
