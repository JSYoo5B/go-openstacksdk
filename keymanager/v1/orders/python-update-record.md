# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/update_order`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithUpdateRecordBasePath`

- String-ID Order commit removes id from dirty state; unknown-only/no attributes makes no HTTP or receipt, while known null/false attributes send one unwrapped PUT. Alias option and response member order use the last canonical value; meta request JSON stays raw and descriptor view is projected separately.
  Tests: `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordNoop`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordDirtyAliasesAndOwnedReceipt`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordAliasPriority`.
- Final 100..399 statuses commit the actual response; invalid/empty JSON retains seed, valid nonobject or invalid HREF reports a response error with physical body/status. Owned request/seed/resource/wire/header state is independent; configured literal route/header/microversion remain scoped to the selected client.
  Tests: `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordResponseDomain`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordAcceptedStatusAndFault`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordOptionsOwnInputsAndLiteralPath`.
- Preflight rejects unsafe dirty IDs, scalar-constructor collisions and selected-route mutation before HTTP. Bounded shared charset normalization decodes Latin1 into Resource/Wire while preserving original receipt/error bytes.
  Tests: `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordPreflightAndClientGuard`, `keymanager/v1/orders/python_update_record_test.go:TestPythonOrderUpdateRecordCharsetKeepsPhysicalProof`.

Python/Go differences:

- The Go mapping deliberately accepts a newly seeded string ID plus closed finite JSON attributes, rather than mutable Python Resource/dictionary seeds, Python keyword binding or existing dirty/equality/recommit history. ID is not included in the PUT body; the previous remaining text incorrectly claimed it was.
- Dirty PUT IDs must be nonblank UTF-8 literal segments without controls/dot segments; custom BasePath is a literal service-relative path rather than a Python URI printf template. Source dynamic constructor/binding collisions are represented by concrete Go options and preflight errors.
- The explicit Go JSON profile preserves finite raw JSON number literals and supports UTF-8/Latin1 plus UTF-16/32 BOM or explicit endianness. Requests implicit charset guessing, other codec namespaces, BOM-less bare UTF-16/32 native-endian and permissive NaN/Infinity are outside this profile. Unknown attributes remain discarded; no caller builder is required.
- Go returns independent input/resource/wire/raw HTTP observations rather than mutating and returning the same Python object. This operation mapping does not claim live Barbican PUT support or full SDK-R1/C1/S1 parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--order", default="o1")
    parser.add_argument("--attrs-json", default="{}")
    args = parser.parse_args()
    attrs = json.loads(args.attrs_json)
    with openstack.connect(cloud=args.cloud) as conn:
        result = conn.key_manager.update_order(args.order, **attrs)
        print(json.dumps(result.to_dict(), ensure_ascii=False))


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
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
)

func run(ctx context.Context, cloud, id, attributes string) error {
    var attrs map[string]json.RawMessage
    if err := json.Unmarshal([]byte(attributes), &attrs); err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { return err }
    result, err := service.Orders.UpdateRecord(ctx, id,
        orders.WithUpdateRecordOpts(orders.UpdateRecordOpts{Attributes: attrs}))
    if err != nil { return err }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Committed bool `json:"committed"`
        StatusCode int `json:"status_code"`
        Resource map[string]json.RawMessage `json:"resource"`
    }{result.Committed, result.StatusCode, result.Resource.Body})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    id := flag.String("order", "o1", "literal order ID")
    attributes := flag.String("attrs-json", "{}", "closed JSON attributes; empty object makes no PUT")
    timeout := flag.Duration("timeout", time.Minute, "whole request deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *id, *attributes); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```
