# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/update_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithUpdateRecordBasePath`

- Scalar ID/no recognized attrs performs no PUT/GET and keeps seed without HTTP receipt; id itself is not dirty.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordNoopSeedHasNoReceiptOrPUT`.
- Container dirty attributes are flat unwrapped JSON PUT without id; nullable/raw values and Source list/HREF projection remain distinct from received fields and owned request/seed.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordFlatDirtyBodyNullableAndRawValues`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordAliasesOrderResponseMergeAndReferenceProjection`.
- Malformed JSON/empty body keeps seed; valid nonobject and malformed HREF errors preserve accepted physical status/header/body.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordEmptyInvalidJSONAndShapeErrorsKeepActualProof`.
- Concrete options snapshot raw attrs, headers, version/basepath and literal escaped IDs; unsafe constructor/source/context/header inputs fail before HTTP.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordOptionsSnapshotBasePathAndSelectedVersion`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordPreflightAndConstructorCollisions`.
- Synthetic final100..399 accepts,400..999 preserves native proof; physical read/Close/cancel/source faults suppress callbacks; clean retry preserves body while tampering blocks replay and failedreauth retains original401.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordAcceptedAndRejectedStatusPolicy`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordPhysicalFailuresNeverReplayAndRetainNativeProof`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordCleanRetryAndFailedReauthPreserveBody`.
- Shared charset subset decodes Latin1, BOM UTF16/32 or explicitLE/BE projection while actual envelope/headers/error proof remain physical; unsupported codecs remain explicit.
  Tests: `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordRequestsCharsetGapIsExplicit`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordExplicitCharsetsDecodeViewsKeepPhysicalEnvelope`, `keymanager/v1/containers/python_update_record_test.go:TestPythonContainerUpdateRecordDecodedShapeFailureKeepsOriginalEncodedProof`.

Python/Go differences:

- Pinned Source commit discards dirty id before checking requires_commit, correcting the older review that said id is in the body. New API maps only scalar-ID newly seeded closed JSON commits; no-attrs is no-op.
- Source unknown ordinary attrs are ignored; Go raw attributes must be valid UTF8 JSON before selection. Go map bulk alias conflicts use sorted keys, while sequential factories preserve order.
- Accepted HTTP evidence is separated from projected Resource/received Wire; Committed false never proves server execution did not occur. Live Barbican support is not established by Source emulation.
- Existing Container/Resource or dictionary seed synchronized/dirty/equal noPUT state, pending dirty/recommit and same-object behavior are not exposed by this scalar-ID API. Full Python kwargs duplicate binding/constructor and printf URI templates are not reproduced; empty/control/dot-segment ID commits are restricted.
- Requests response charset sniffing/full codecs, BOM-less bare utf16/utf32 native-endian choice and permissive nonfinite JSON remain outside the shared normalized subset; cp1006/unsupported charset retain ErrUnsupported with actual response proof.
- Independent Source audit executed 11 AST-extracted assertions for component dirty tracking, attribute consumption, commit/body preparation and list conversion; this was not Proxy/decoder/full-runtime or live-cloud execution. Full-runtime import was blocked by unavailable cached keystoneauth1.adapter, without dependency installation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--container", default="c1")
    parser.add_argument("--attrs-json", default="{}")
    args = parser.parse_args()
    attrs = json.loads(args.attrs_json)
    with openstack.connect(cloud=args.cloud) as conn:
        result = conn.key_manager.update_container(args.container, **attrs)
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
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
)

func run(ctx context.Context, cloud, id, attributes string) error {
    var attrs map[string]json.RawMessage
    if err := json.Unmarshal([]byte(attributes), &attrs); err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { return err }
    result, err := service.Containers.UpdateRecord(ctx, id,
        containers.WithUpdateRecordOpts(containers.UpdateRecordOpts{Attributes: attrs}))
    if err != nil { return err }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Committed bool `json:"committed"`
        StatusCode int `json:"status_code"`
        Resource map[string]json.RawMessage `json:"resource"`
    }{result.Committed, result.StatusCode, result.Resource.Body})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    id := flag.String("container", "c1", "literal container ID")
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

```sh
python main.py --cloud devstack --container c1 --attrs-json '{"name":"new","updated_at":null}'
go run main.go --cloud devstack --container c1 --attrs-json '{"name":"new","updated_at":null}'
```
