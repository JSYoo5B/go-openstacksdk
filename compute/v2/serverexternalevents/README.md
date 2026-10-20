# Server external events

`compute.Service.ServerExternalEvents.CreateRecords` maps pinned openstacksdk `create_server_external_events`. This Nova API is intended for inter-service event notifications. The SDK posts one nonempty batch to `POST /os-server-external-events` with an `events` array; it does not split, reorder or automatically retry a batch after a physical failure.

`Event` is an SDK-owned concrete JSON object, without a user builder interface. Declared request fields are `name`, `server_uuid`, `status`, `tag`, `code`, plus inherited `id`. Unknown fields are discarded before JSON serialization. Known values remain raw: for example `code:"-1"` is transmitted as a string even though the Resource descriptor view would be0. Null remains null. Empty/nil batches, nil event objects, constructor collisions (`self`, `cls`, `connection`, `_synchronized`) and invalid declared JSON/getter values fail before HTTP. Nova validates event names/statuses and service authorization.

The POST and envelope decoding happen when CreateRecords is called; returned row projection happens during iteration. Each row has:

- `Resource`: declared fields with missing values null, integer `code` descriptor conversion, and a null library-owned location view;
- `Wire`: all actual row fields, including unknown extensions;
- `Envelope`, `Header`, `StatusCode`: the whole physical response evidence.

Response rows are constructed independently and never merge request attributes. Unknown response properties remain in Wire but not Resource. A malformed later row yields an error after earlier rows; `resource.ResponseError` preserves the whole receipt. Breaking iteration stops before later projection. HTTP errors retain native status/header/body evidence and joined physical causes. Accepted read/Close/context/source failures are terminal too.

## Standalone Python

```python
import openstack


def main():
    conn = openstack.connect()
    rows = conn.compute.create_server_external_events([
        {"name": "network-changed", "server_uuid": "server-id", "status": "completed"}
    ])
    for row in rows:
        print(row.name, row.code)


if __name__ == "__main__":
    main()
```

## Standalone Go

Set `OS_COMPUTE_ENDPOINT` to the project-scoped Nova endpoint and `OS_TOKEN` to an authorized service token.

```go
package main

import (
    "context"
    "log"
    "os"

    compute "github.com/JSYoo5B/go-openstacksdk/compute/v2"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/serverexternalevents"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    endpoint := os.Getenv("OS_COMPUTE_ENDPOINT")
    if endpoint == "" { log.Fatal("OS_COMPUTE_ENDPOINT is required") }
    provider := &gophercloud.ProviderClient{TokenID: os.Getenv("OS_TOKEN")}
    client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: gophercloud.NormalizeURL(endpoint), Type: "compute"}
    service := compute.New(client)
    rows, err := service.ServerExternalEvents.CreateRecords(context.Background(), []serverexternalevents.Event{
        {"name": "network-changed", "server_uuid": "server-id", "status": "completed"},
    })
    if err != nil { log.Fatal(err) }
    for row, err := range rows {
        if err != nil { log.Fatal(err) }
        log.Printf("name=%s code=%s status=%d", row.Resource.Body["name"], row.Resource.Body["code"], row.StatusCode)
    }
}
```

## Deliberate Go profile boundaries

The Go operation uses a concrete slice/object and Go error iteration instead of arbitrary Python Sequence/Mapping objects and exceptions. Successful HTTP100–399 still requires a JSON `events` envelope; malformed non-array events shapes are reported upon iteration in this Go profile, while Python can reject some shapes during generator creation. Duplicate JSON keys, nonfinite values, arbitrary Python runtime objects and the full Requests codec/session behavior are outside the finite owned JSON profile. Response metadata is exposed directly instead of hidden in a mutable Resource.

`WithCreateHeader` and `WithCreateMicroversion` are explicit transport options. Omission preserves the selected client version without discovery: the Source class declares no maximum. Arbitrary mutable Connection/location/cache state is not synthesized. SDK-R1/C1/S1, real-cloud interoperability and release validation remain separate; this mapping is not complete SDK parity.

Focused tests in `python_create_records_test.go` reuse the Gophercloud testhelper-backed `internal/testcloud` fixture and cover batch wire data, declared versus raw views, lazy partial failure, rejected/accepted physical errors, ownership and source guards.
