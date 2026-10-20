# ProjectQuota raw records and Source commit

`ProjectScope.GetRecord` and `UpdateRecord` add the pinned ProjectQuota Resource profile beside the existing strict controller-schema APIs. The scope owns one explicit project ID; neither response IDs nor hrefs retarget requests.

`UpdateRecord` sends `PUT /project-quotas/{id}` with a `project_quotas` envelope containing only supplied recognized dirty fields. The Source declarations `secrets`, `orders`, `containers`, `consumers`, `cas`, inherited `name` and `id` are untyped. Explicit null, strings, booleans and large JSON numbers remain raw. Unknown ordinary attributes are ignored before encoding; explicit id/constructor collisions fail before HTTP. No recognized attributes means no PUT and a seeded Resource without fabricated Wire/Envelope/status. The Source public method extracts a project ID before constructing ProjectQuota, so this lane does not need to fetch previous quotas or emulate comparison against an already loaded quota object.

GetRecord accepts transport options only and performs one GET, admitting HTTP100–399. Both methods unwrap project_quotas when present, otherwise use the root object; generic empty/malformed JSON keeps the seed, while valid nonobject/envelope-invalid shapes fail with actual proof. Returned Resource has seven nullable Body fields and library-owned location null; Wire contains only actual response fields. Sent indicates a response observation/attempt, not remote durability or mutation completion. UpdateRecord never automatically follows up with GET.

Existing `ProjectScope.Update(UpdateOpts)` remains the explicit integer-only reset API: empty options send an empty replacement and success is204. UpdateRecord instead follows no-dirty no-op semantics. Neither API invents server acceptance of raw values rejected by the actual Barbican controller.

## Standalone Python

```python
import openstack


def main():
    conn = openstack.connect()
    quota = conn.key_manager.update_project_quota("project-id", secrets=10, orders=20)
    print(quota.secrets, quota.orders)


if __name__ == "__main__":
    main()
```

## Standalone Go

Set `OS_KEY_MANAGER_ENDPOINT` to Barbican v1 and `OS_TOKEN` to an authorized token.

```go
package main

import (
    "context"
    "log"
    "os"

    keymanager "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
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
    scope, err := service.Quotas.InProject(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    record, err := scope.UpdateRecord(ctx,
        quotas.WithRecordAttribute("secrets", 10),
        quotas.WithRecordAttribute("orders", 20),
    )
    if err != nil { log.Fatal(err) }
    log.Printf("sent=%t status=%d secrets=%s", record.Sent, record.StatusCode, record.Resource.Body["secrets"])
}
```

`WithRecordOpts` owns attributes/header maps, bytes and version pointers; functional attribute/header/microversion helpers do not require a user builder. Selected scope/source changes during options fail before HTTP. Native HTTP errors retain original status/body/header, accepted physical errors retain ResponseError, and read/Close/context/source faults are terminal without replay.

This is a finite owned JSON/scalar-ID profile. Arbitrary Python Resource subclasses, dirty re-commit history, adapter/cache/location, per-call arbitrary base paths and full Requests/runtime behavior remain SDK-R1/C1/S1 boundaries. Live authorization, controller interoperability and release verification are separate.
