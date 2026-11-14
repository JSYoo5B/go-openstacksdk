# Member Record Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

owner = openstack.connect(cloud="owner")
recipient = openstack.connect(cloud="recipient")
image_id = "image-id"
member_id = "recipient-project-id"

created = owner.image.add_member(image_id, member_id=member_id)
fetched = owner.image.get_member(member_id, image_id)
updated = recipient.image.update_member(fetched, image_id, status="accepted")
owner.image.remove_member(member_id, image_id, ignore_missing=False)
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ownerCloud := flag.String("owner-cloud", os.Getenv("OPENSTACKSDK_OWNER_CLOUD"), "owner clouds.yaml cloud")
    memberCloud := flag.String("member-cloud", os.Getenv("OPENSTACKSDK_MEMBER_CLOUD"), "shared recipient clouds.yaml cloud")
    imageID := flag.String("image-id", os.Getenv("OPENSTACKSDK_IMAGE_ID"), "literal parent image ID")
    memberID := flag.String("member-id", os.Getenv("OPENSTACKSDK_MEMBER_ID"), "literal recipient project ID")
    status := flag.String("status", "accepted", "requested member status")
    flag.Parse()
    if *ownerCloud == "" || *memberCloud == "" || *imageID == "" || *memberID == "" {
        log.Fatal("owner-cloud, member-cloud, image-id and member-id are required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *ownerCloud, *memberCloud, *imageID, *memberID, *status); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func showRecord(operation string, record *image.ImageMemberRecord, callErr error) error {
    value := map[string]any{"operation": operation}
    if record != nil {
        value["resource"] = record.Resource
        value["wire"] = record.Wire
        value["image_id"] = record.ImageID
        value["status_code"] = record.StatusCode
        value["envelope_bytes"] = len(record.Envelope)
    }
    if callErr != nil { value["error"] = callErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return callErr
}

func showAcknowledgement(ack *image.ImageMemberAcknowledgement, callErr error) error {
    value := map[string]any{"operation": "remove"}
    if ack != nil {
        value["image_id"] = ack.ImageID
        value["member_id"] = ack.MemberID
        value["status_code"] = ack.StatusCode
        value["response_bytes"] = len(ack.Body)
    }
    if callErr != nil { value["error"] = callErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return callErr
}

func run(ctx context.Context, ownerCloud, memberCloud, imageID, memberID, status string) error {
    owner, err := openstack.Connect(ctx, openstack.WithCloud(ownerCloud))
    if err != nil { return err }
    recipient, err := openstack.Connect(ctx, openstack.WithCloud(memberCloud))
    if err != nil { return err }
    parent := resource.ID(imageID)
    input := image.ImageMemberRecordRequest{ID: memberID}
    created, err := owner.AddImageMemberRecord(ctx, parent,
        image.WithImageMemberRecordWriteAttribute("member_id", memberID),
        request.Header("X-Request-Source", "member-record-example"))
    if err := showRecord("add", created, err); err != nil { return err }
    fetched, err := owner.GetImageMemberRecord(ctx, parent, input,
        request.Header("X-Request-Source", "member-record-example"))
    if err := showRecord("get", fetched, err); err != nil { return err }
    updated, err := recipient.UpdateImageMemberRecord(ctx, parent,
        image.ImageMemberRecordRequest{Record: fetched},
        image.WithImageMemberRecordWriteAttribute("status", status))
    if err := showRecord("update", updated, err); err != nil { return err }
    service, err := recipient.Image(ctx)
    if err != nil { return err }
    repeated, err := service.UpdateImageMemberRecord(ctx, parent,
        image.ImageMemberRecordRequest{Record: updated},
        image.WithImageMemberRecordWriteAttribute("status", status))
    if err := showRecord("update-again", repeated, err); err != nil { return err }
    ack, err := owner.RemoveImageMemberRecord(ctx, parent, input,
        request.IgnoreMissing(false),
        request.Header("X-Request-Source", "member-record-example"))
    return showAcknowledgement(ack, err)
}
```

## References

- [Member Records](member-records.md)
- [Members](members.md)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L223-L292)
- [Install](../docs/install.md)
- [add_member 1222–1241](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1241)
- [get_member 1316–1336](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1316-L1336)
- [update_member 1356–1386](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1356-L1386)
- [remove_member 1243–1271](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1243-L1271)
- [Member descriptor 16–43](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)
- [generic proxy create/update/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669-L799)
- [Resource response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [microversion negotiation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1439)
- [create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519-L1638)
- [commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1937)
- [delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2152)
