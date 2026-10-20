# Swift image upload object cleanup

`swift.Cloud.DeleteAutocreatedImageObjects` maps pinned openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` cloud `delete_autocreated_image_objects`. It owns task policy, lazy listing, fresh metadata selection and deletion order. Callers provide concrete options rather than composing builders or stitching requests together.

| Python | Go |
|---|---|
| connection `image_api_use_tasks` | `WithAutocreatedImageTasks(bool)`; default false |
| container omitted/None | nil `Container`; default `images` |
| `container="custom"` | `WithAutocreatedImageContainer("custom")` |
| `segment_prefix="upload/"` | `WithAutocreatedImagePrefix("upload/")` |
| returned bool | result `Deleted`; per-object receipts retained separately |

When tasks are disabled the workflow makes no Swift requests, even if the selected client is absent. When enabled it lazily lists each page with `format=json` and optional server prefix. Each yielded object gets a fresh metadata HEAD before processing the next object. Only exact custom metadata `x-sdk-autocreated == "true"` qualifies. `True`, `1`, an empty value and the older `x-shade-autocreated` marker do not qualify. A later page error preserves earlier deletions and stops the workflow.

The normal HEAD updates the object's static-large-object flag. If it is absent, a listed flag is retained; if both are absent another HEAD runs inside the delete phase. A nonempty header string, including `"false"` or `"0"`, is truthy in the pinned Python bool descriptor. A true flag adds `multipart-manifest=delete`. The DELETE forwards the current custom metadata, using the second HEAD values when that HEAD occurs.

A clean 404 from the first metadata HEAD is an error. A clean 404 from optional deletion discovery or DELETE is ignored and counts as `Deleted=true`, matching Proxy `ignore_missing=True`; discovery 404 may therefore have no DELETE receipt. Other errors propagate immediately. A successful DELETE's non-JSON or bulk report body is retained without applying the separate Go `DeleteObject` bulk-report validation policy. The Source helper also does not inspect that report.

The default examples below do no Swift cleanup requests because tasks are disabled. Setting `--tasks` enables actual deletion of marked upload objects. Configure the selected cloud and Swift permissions before running.

```python
import argparse
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--tasks", action="store_true")
    parser.add_argument("--container")
    parser.add_argument("--prefix")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud, image_api_use_tasks=args.tasks) as conn:
        print(conn.delete_autocreated_image_objects(
            container=args.container, segment_prefix=args.prefix))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func run(ctx context.Context, cloud, container, prefix string, tasks bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    swift, err := conn.ObjectStorage(ctx)
    if err != nil { return err }
    result, err := swift.Cloud.DeleteAutocreatedImageObjects(ctx,
        objectstorage.WithAutocreatedImageTasks(tasks),
        objectstorage.WithAutocreatedImageContainer(container),
        objectstorage.WithAutocreatedImagePrefix(prefix))
    if err != nil { return err }
    fmt.Printf("deleted=%t observed=%d\n", result.Deleted, len(result.Observations))
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    container := flag.String("container", "images", "Swift container")
    prefix := flag.String("prefix", "", "server object prefix")
    tasks := flag.Bool("tasks", false, "enable image task upload cleanup")
    timeout := flag.Duration("timeout", time.Minute, "whole workflow timeout")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *container, *prefix, *tasks); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Existing clients can call `objectstorage.New(client)` directly. `WithAutocreatedImageObjectsOpts` copies pointer inputs when created and applied. Replacement options restore nil/default fields; a nil prefix omits the query and an explicit empty prefix is preserved. Options default to the pinned connection setting's false value; this helper deliberately does not read hidden Python configuration or automatically enable tasks.

`AutocreatedImageObjectsResult` preserves the resolved container, Source bool and ordered observations. Each observation separates the listed object, first metadata HEAD, optional discovery HEAD and actual DELETE acknowledgement. `IgnoredMissing` identifies a clean missing delete phase; `Deleted` means the Source helper's successful-or-missing operation, not proof that an object or every SLO segment existed and was removed. Failed responses and previously completed phases remain available with the returned error. Never blindly retry the entire cleanup after uncertain writes.

The workflow reuses owned `objects.ListRecords`, `HeadMetadata`, shared REST guards and the existing Gophercloud public testhelper fixture. Go input is typed literal strings and finite JSON listing records. Existing listing pagination safety bounds, name validation, metadata projection and safe header policies remain in force. Response metadata HEAD accepts the existing Python metadata helper's 200..399 profile; deletion accepts final 100..399 and clean 404. Intermediate responses and redirects are still governed by net/http. Mutable Python Resource subclasses, configuration mutation during iteration and arbitrary Python descriptors are outside this Go profile. Live Swift task uploads, policy and authorization acceptance remain separate release validation.

Focused tests cover disabled defaults and owned options; exact marker selection and interleaved page/HEAD/DELETE order; listed/HEAD SLO flag behavior and second-HEAD missing; first-HEAD, list and mutation errors; cancellation and route changes before HTTP. Root runs focused/full Go checks and standalone example builds. No live cloud execution is claimed.
