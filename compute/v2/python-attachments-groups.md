# Python Attachments Groups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
for attachment in conn.compute.volume_attachments(
    server_id, limit=20, offset=0, max_items=100,
):
    print(attachment.to_dict())

conn.compute.update_volume_attachment(
    server_id, current_volume_id, delete_on_termination=False,
)
conn.compute.update_volume_attachment(
    server_id, current_volume_id, volume_id=replacement_volume_id,
)
group = conn.compute.find_server_group(group_name, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    serverID := flag.String("server", "", "server ID to list attachments")
    groupName := flag.String("group", "", "server group exact ID or name")
    allProjects := flag.Bool("all-projects", false, "search server groups across projects")
    updateVolume := flag.String("update-volume", "", "current volume ID to update")
    replacement := flag.String("swap-to", "", "replacement volume ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.Compute, "2.89"))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    if *serverID != "" {
        for value, err := range service.VolumeAttachments.ListRecords(ctx, *serverID,
            request.Limit(20),
            request.Custom(func(c *volumeattach.AttachmentListOpts) error { v := 0; c.Offset = &v; return nil }),
            request.MaxItems(100)) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    }
    if *groupName != "" {
        value, err := service.ServerGroups.FindRecord(ctx, *groupName,
            request.IgnoreMissing(false),
            request.Custom(func(c *servergroups.GroupFindOpts) error { c.AllProjects = *allProjects; return nil }))
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
    }
    if *updateVolume != "" {
        if *serverID == "" { log.Fatal("-server is required for -update-volume") }
        options := []volumeattach.AttachmentUpdateOption{
            volumeattach.WithAttachmentAttribute("delete_on_termination", false),
        }
        if *replacement != "" {
            options = append(options, volumeattach.WithAttachmentAttribute("volume_id", *replacement))
        }
        value, err := service.VolumeAttachments.UpdateRecord(ctx,
            *serverID, *updateVolume, options...)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
        fmt.Printf("HTTP %d request-id=%s\n", value.StatusCode, value.Header.Get("X-Openstack-Request-Id"))
    }
}
```

## References

- [Install](../../docs/install.md)
