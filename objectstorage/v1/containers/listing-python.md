# Listing Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prefix", default="assets/")
    parser.add_argument("--page-size", type=int)
    parser.add_argument("--max-items", type=int, default=0)
    args = parser.parse_args()
    query = {"prefix": args.prefix, "max_items": args.max_items}
    if args.page_size is not None:
        query["limit"] = args.page_size
    conn = openstack.connect()
    for container in conn.object_store.containers(**query):
        print(container.name, container.count, container.bytes)

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    prefix := flag.String("prefix", "assets/", "container name prefix")
    pageSize := flag.Int("page-size", 0, "server page size; zero disables limit fallback")
    maxItems := flag.Int("max-items", 0, "raw rows to consume; zero means no row cap")
    flag.Parse()
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }
    options := []containers.RecordListOption{
        request.Custom(func(c *containers.RecordListOpts) error { v := *prefix; c.Prefix = &v; return nil }),
        request.Limit(*pageSize),
        request.MaxItems(*maxItems),
    }
    for record, err := range swift.Containers.ListRecords(ctx, options...) {
        if err != nil {
            if record != nil {
                log.Printf("actual page status=%d body=%s", record.StatusCode, record.Envelope)
            }
            log.Fatal(err)
        }
        fmt.Printf("name=%s count=%s bytes=%s HTTP=%d\n",
            record.Resource.Body["name"], record.Resource.Body["count"],
            record.Resource.Body["bytes"], record.StatusCode)
    }
}
```

## References

- [Python List Contracts Test](python_list_contracts_test.go)
