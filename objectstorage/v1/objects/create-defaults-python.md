# Create Defaults Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }

    // Content omitted: one ordinary PUT, with the MD5 of empty bytes.
    empty, err := swift.Objects.Create(ctx, "backups", "empty.bin", objects.CreateOpts{})
    if err != nil { log.Fatal(err) }
    fmt.Printf("PUT ETag=%q transaction=%q\n", empty.ETag, empty.TransID)

    // The option snapshots these bytes. Reusing it starts a new reader.
    data := []byte("hello")
    body := objects.WithCreateBytes(data)
    data[0] = 'X'
    for _, name := range []string{"first.txt", "second.txt"} {
        result, err := swift.Objects.Create(ctx, "backups", name,
            objects.CreateOpts{ContentType: "text/plain"}, body,
            request.Header("X-Object-Meta-Category", "example"))
        if err != nil { log.Fatal(err) }
        fmt.Printf("%s PUT ETag=%q\n", name, result.ETag)
    }

    // Source-shaped bytes workflow: no native ETag calculation, and raw PUT
    // acknowledgement is retained separately from metadata observations.
    uploaded, err := swift.Objects.CreateObject(ctx, "backups", "source-empty.bin",
        objects.CreateObjectInput{Data: []byte{}})
    if err != nil { log.Fatal(err) }
    if uploaded.Ordinary != nil && uploaded.Ordinary.Acknowledgement != nil {
        fmt.Printf("source PUT status=%d\n", uploaded.Ordinary.Acknowledgement.StatusCode)
    }
}
```

```python
import openstack

conn = openstack.connect()
obj = conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "first.txt", data=b"hello",
                                content_type="text/plain")
```

## References

- [owned create workflow](create.md)
- [Create](create.md)
- [Create Defaults Test](create_defaults_test.go)
- [Native Contracts Test](native_contracts_test.go)
