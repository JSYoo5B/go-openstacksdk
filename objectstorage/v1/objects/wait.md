# Wait

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showWait(result *objects.ObjectWaitResult, err error) {
    if result != nil {
        status := "<absent>"
        if result.Status != nil {
            status = *result.Status
        }
        fmt.Printf("polls=%d complete=%v deleted=%v status=%q\n",
            result.Polls, result.Complete, result.Deleted, status)
        if result.Last != nil {
            for _, attempt := range result.Last.Attempts {
                fmt.Printf("physical=%d error=%v\n", attempt.PhysicalAttempt, attempt.Error)
                if attempt.Response != nil {
                    fmt.Printf("HTTP%d headers=%v body=%q\n",
                        attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
                }
            }
            if result.Last.Acknowledgement != nil {
                fmt.Printf("last acknowledgement HTTP%d\n", result.Last.Acknowledgement.StatusCode)
            }
        }
    }
    if err != nil {
        fmt.Println(err)
        if errors.Is(err, resource.ErrUnsupported) {
            fmt.Println("unsupported status source")
        }
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
                proof.StatusCode, proof.Header, proof.Body)
        }
    }
}

func waitExamples(ctx context.Context, conn *sdk.Connection) {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        showWait(nil, err)
        return
    }
    policy := []objects.ObjectWaitOption{
        objects.WithObjectWaitOpts(objects.ObjectWaitOpts{
            Headers: map[string]string{"X-Client": "object-wait"},
            Wait:    []resource.WaitOption{resource.WithPollInterval(2 * time.Second)},
        }),
        request.Header("X-Trace", "initial"),
        objects.WithObjectWaitHeaders(map[string]string{"x-trace": "last"}),
        objects.WithObjectWait(
            resource.WithTimeout(time.Minute),
            resource.WithProgressCallback(func(progress int) error {
                fmt.Printf("nonterminal observation, progress=%d\n", progress)
                return nil
            }),
        ),
    }
    statusOptions := append([]objects.ObjectWaitOption{}, policy...)
    statusOptions = append(statusOptions,
        objects.WithObjectWaitStatusHeader("X-Object-Meta-State"),
        objects.WithObjectWait(resource.WithFailureStates("ERROR", "FAILED")),
    )
    result, err := service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "READY", statusOptions...)
    showWait(result, err)

    // A known string attribute can also be observed, such as the MIME type.
    attributeOptions := []objects.ObjectWaitOption{
        objects.WithObjectWait(resource.WithStatusAttribute("content_type"), resource.WithFailureStates()),
    }
    result, err = service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "application/octet-stream", attributeOptions...)
    showWait(result, err)

    // UnlimitedWait removes the SDK deadline; this caller still has 30 seconds.
    deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    deleteOptions := []objects.ObjectWaitOption{objects.WithObjectWait(resource.WithUnlimitedWait())}
    result, err = service.Objects.WaitForDelete(deleteCtx, "backups", "already-deleted.bin", deleteOptions...)
    showWait(result, err)
}

func main() {}
```

## References

- [pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L1247-L1306)
- [Resource waits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [Object descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [Collection.WaitDeleted](../../../resource/README.md)
