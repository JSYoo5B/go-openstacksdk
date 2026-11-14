# Messages Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.37")
    parser.add_argument("--message", default="")
    parser.add_argument("--project", default="")
    parser.add_argument("--delete", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.message:
        if args.delete:
            conn.block_storage.delete_message(args.message, ignore_missing=False)
        else:
            value = conn.block_storage.get_message(args.message)
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
    else:
        query = {"project_id": args.project} if args.project else {}
        for value in conn.block_storage.messages(**query):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printMessage(value *resource.RawResource) {
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.37", "Cinder selected version")
    id := flag.String("message", "", "optional message ID")
    project := flag.String("project", "", "optional server project filter")
    remove := flag.Bool("delete", false, "strictly delete the selected message")
    flag.Parse()
    if *remove && *id == "" { log.Fatal("-delete requires -message") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Messages
    if *id != "" {
        if *remove {
            if err := api.DeleteRecord(ctx, *id,
                request.IgnoreMissing(false)); err != nil { log.Fatal(err) }
            return
        }
        value, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printMessage(value.Resource)
        return
    }
    options := []messages.RecordListOption{}
    if *project != "" { options = append(options, request.Query("project_id", *project)) }
    for value, err := range api.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        printMessage(value.Resource)
    }
}
```

## References

- [Message](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/message.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3029-L3064)
- [Install](../../docs/install.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#show-message-details)
- [Policy Personas](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy-personas.html#user-messages-microversion-3-3)
- [Policy](https://docs.openstack.org/cinder/2026.1/configuration/block-storage/policy.html)
- [Python Records Contracts Test](messages/python_records_contracts_test.go)
