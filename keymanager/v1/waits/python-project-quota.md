# Barbican admin ProjectQuota waits

`ProjectQuota(projectID)` and `ForProjectQuota(record)` add the missing concrete admin target to `WaitForStatus` and `WaitForDelete`. They reuse the existing guarded reader and prepared wait policy, polling only `GET /project-quotas/{originalProjectID}`. Response id/name/href are passive. Literal targets seed Resource.id with the requested project, all other declared fields with null, and no fake receipt. Cached records snapshot actual Resource/Wire/Envelope/header/status independently.

ProjectQuota declares **no status field**. A default status wait fails unsupported before HTTP; select a declared string/null field using `WithAttribute`. Numeric quota values are not converted into status text. Cached matching string/null states return without polling. Status timeout is unlimited, deletion timeout120s, interval2s; explicit options preserve the existing Source-shaped timer/failure/callback policy.

WaitForDelete sends no DELETE. It ignores undeclared wire status=deleted and completes only when GET returns a clean404. A physical read/Close/context/source failure carrying404 is not absence. The result owns the original Resource snapshot plus actual absence receipt. Resetting a project override normally leaves effective/configuration GET available; observing deletion is not proof that quota settings were reset.

## Standalone Python

```python
import os
import openstack


def main():
    conn = openstack.connect()
    quota = conn.key_manager.get_project_quota("project-id")
    current = conn.key_manager.wait_for_status(
        quota, "project-id", attribute="id", wait=120
    )
    print(current.id)
    if os.environ.get("WAIT_ABSENCE") == "1":
        conn.key_manager.wait_for_delete(quota, wait=120)


if __name__ == "__main__":
    main()
```

## Standalone Go

Set `OS_KEY_MANAGER_ENDPOINT` and `OS_TOKEN`. The optional `WAIT_ABSENCE=1` demonstrates absence observation rather than issuing a configuration DELETE.

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    keymanager "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    endpoint := os.Getenv("OS_KEY_MANAGER_ENDPOINT")
    if endpoint == "" { log.Fatal("OS_KEY_MANAGER_ENDPOINT is required") }
    provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
    client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "key-manager"}
    service := keymanager.New(client)
    ctx := context.Background()
    projectID := "project-id"
    scope, err := service.Quotas.InProject(ctx, resource.ID(projectID))
    if err != nil { log.Fatal(err) }
    record, err := scope.GetRecord(ctx)
    if err != nil { log.Fatal(err) }
    target := waits.ForProjectQuota(record)
    current, err := service.Waits.WaitForStatus(ctx, target, &projectID,
        waits.WithAttribute("id"), waits.WithTimeout(120*time.Second))
    if err != nil { log.Fatal(err) }
    log.Printf("id=%s status=%d", current.Resource.Body["id"], current.StatusCode)
    if os.Getenv("WAIT_ABSENCE") == "1" {
        if _, err := service.Waits.WaitForDelete(ctx, target, waits.WithTimeout(120*time.Second)); err != nil { log.Fatal(err) }
    }
}
```

These finite constructors now cover the known ProjectQuota gap alongside Container, Order, Secret, SecretStore, SecretACL and effective Quota. SecretConsumer has allow_fetch=False in pinned Source. This does not implement arbitrary Python Resource/runtime subclasses, mutable identity, session/location/cache, untyped callbacks or every dynamic getattr behavior. Immutable routes, typed callback errors, selected nullable strings, finite UTF-8 JSON and owned-null location remain explicit Go differences under SDK-R1/C1/S1. Local HTTP tests do not establish live cloud or release parity.
