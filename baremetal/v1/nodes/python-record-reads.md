# Python Record Reads

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import sys
import openstack

def main():
    conn = openstack.connect(cloud='dev', baremetal_api_version='1.109')
    action = sys.argv[1] if len(sys.argv) > 1 else 'list'
    identity = sys.argv[2] if len(sys.argv) > 2 else 'node-a'
    if action == 'list':
        values = conn.baremetal.nodes(details=True, fields=['id', 'name'])
        print(json.dumps([value.to_dict() for value in values], indent=2))
    elif action == 'get':
        value = conn.baremetal.get_node(identity, fields=['id', 'name'])
        print(json.dumps(value.to_dict(), indent=2))
    elif action == 'find':
        value = conn.baremetal.find_node(identity, ignore_missing=True, details=True)
        print(json.dumps(value.to_dict() if value is not None else None, indent=2))
    else:
        raise ValueError('expected list, get or find')

if __name__ == '__main__':
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BareMetal(ctx)
    if err != nil { log.Fatal(err) }
    action, identity := "list", "node-a"
    if len(os.Args) > 1 { action = os.Args[1] }
    if len(os.Args) > 2 { identity = os.Args[2] }
    var output any
    switch action {
    case "list":
        values, callErr := service.Nodes.AllRecords(ctx,
            request.Microversion("1.109"),
            nodes.WithRecordListDetails(true),
            request.Query("fields", []string{"id", "name"}))
        err = callErr
        views := make([]any, 0, len(values))
        for _, value := range values { views = append(views, value.Resource.Body) }
        output = views
    case "get":
        value, callErr := service.Nodes.GetRecord(ctx, identity,
            request.Microversion("1.109"), nodes.WithRecordGetFields("id", "name"))
        err = callErr
        if value != nil && value.Resource != nil { output = value.Resource.Body }
    case "find":
        value, callErr := service.Nodes.FindRecord(ctx, identity,
            request.Microversion("1.109"), request.Custom(func(c *nodes.RecordFindOpts) error { v := true; c.Details = &v; return nil }),
            request.IgnoreMissing(true))
        err = callErr
        if value != nil && value.Resource != nil { output = value.Resource.Body }
    default:
        log.Fatal("expected list, get or find")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```
