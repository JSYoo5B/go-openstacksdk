# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/update_secret`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.WithUpdateRecordBasePath`

- secrets.API.Update sends PUT secrets/{id} with the raw payload as body and UpdateOpts.ContentType as the Content-Type header, accepts only 204 and turns 200 into an error.
  Tests: `keymanager/v1/secrets/python_parity_test.go:TestPythonSecretUpdateSendsRawPayload`.
- No-op ID-only and unknown/location/microversion/if_match-only inputs make no PUT or GET and retain only a seed Resource, no status/headers/Wire; ID itself is not dirty.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordNoopSeedHasNoReceiptOrPUT`.
- Known attrs are flat unwrapped JSON PUT, nullable values remain distinct, payload is JSON rather than raw upload, raw content_types maps to Source descriptor view, and seed/request/response do not alias.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordFlatDirtyBodyNullableAndRawValues`.
- Request aliases obey option order; response aliases obey JSON insertion order, null/id presence and passive secret_ref/secret_id projection.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordAliasesOrderResponseMergeAndReferenceProjection`.
- Empty or valid-UTF8 malformed JSON responses keep request seed like Source ValueError; valid nonobject or malformed HREF projection errors retain actual accepted receipt.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordEmptyInvalidJSONAndShapeErrorsKeepActualProof`.
- Options snapshot attrs/headers/basepath/version; literal URL escaping and private selectedversion retain sharedclient. Nil/context/source/constructor/unsafe controls fail before commit.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordOptionsSnapshotBasePathAndSelectedVersion`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordPreflightAndConstructorCollisions`.
- Original100..399 status policy, native400..999 proof, read/Close/customcancel/restoredsource terminal errors block more PUT/retry/auth/backoff. Clean native retry keeps bytes, alteredbody terminates before replay and failedreauth retains401original.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordAcceptedAndRejectedStatusPolicy`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordPhysicalFailuresNeverReplayAndRetainNativeProof`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordCleanRetryAndFailedReauthPreserveBody`.
- UTF16/UTF32 with a BOM or explicit LE/BE and ISO8859-1/Latin1 decode through shared responsejson.Normalize only for JSON Resource/Wire projection. Actual Envelope/Header/Status and ResponseError.Body retain original physical bytes, including normalized-shape errors. Unsupported cp1006/remaining namespace keeps ErrUnsupported+actual proof; no-op/dirty PUT semantics remain unchanged.
  Tests: `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordExplicitCharsetsDecodeViewsKeepPhysicalEnvelope`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordDecodedShapeFailureKeepsOriginalEncodedProof`, `keymanager/v1/secrets/python_update_record_test.go:TestPythonSecretUpdateRecordRequestsCharsetGapIsExplicit`.

Python/Go differences:

- secrets.API.Update is Barbican's payload upload; it returns only an error rather than a Secret.
- Pinned openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/Gophercloudv2.15.0/Go1.25.0. New UpdateRecord maps a string-ID seeded JSON commit. Existing API.Update remains rawpayload upload; no generated/native transport alone establishes named parity.
- Actual Source commit discards dirty id before requires_commit: scalar ID does not appear in the PUT body, correcting the earlier review. No dirtyattrs means no PUT/GET and no HTTP receipt.
- Concrete option factories own attrs/rawbytes/header maps/basepath/version pointers. Sequential attribute factories and response JSON preserve alias order; unordered Go input maps use sorted key order. Explicit Headers/Microversion are Go transport controls; Python attrs microversion does not override commit sessionversion.
- SDK-owned Seed/Request/Resource/Wire/Envelope/Header/StatusCode distinguish input from actual response. Committed means accepted body/Close observed; false never proves no remote execution, and a nonnil Resource on error can be a request seed.
- Literal service-relative BasePath and bounded nonblank escaped commit IDs replace arbitrary URI templates/falsey Source ID routes. Captured source/headers/body stay guarded through native hooks; returned HREFs never trigger metadata/payload reads.
- SDK-R1/C1/S1 Resource/class/session/cache foundations remain separate gaps. Actual Source61 observations use Proxy/Secret/Resource/Requests2.34.2/urllib32.8.0/CPython3.14.8 with adapterput returning memoryHTTPResponse. No liveBarbican/auth/retry/ACL proof. Worker no Go execution; root owns final checks.
- Cycle11 closes the declared BOM-or-explicit-LE/BE UTF16/UTF32 plus Latin1 response charset subset through the shared responsejson normalizer; actual wire byte receipts remain separate from normalized views. This does not complete the two named remaining areas or the full Requests loader/codec namespace. Existing standalone mains and Source61 oracle are unchanged; root owns Go validation.

Unresolved boundaries:

- Full existing Python Secret/Resource/dictionary synchronized and dirty-state equality, pending changes, same-instance mutation/recommit, complete MRO/constructor/kwargs binding and URI formatting remain unsupported. Raw self/secret/resource_type/value attrs are ignored but Source duplicate binding raises TypeError beforeHTTP; id/connection/_synchronized/cls are separately rejected. Empty/falsey commit ID routes, arbitrary printf base_path/URI attrs and deferred formal-validation order also remain outside this closed string-ID API.
- Full Requests Response.json guess_json_utf/unmarked encoding inference, codec namespace beyond UTF16/UTF32 with BOM or explicit LE/BE plus ISO8859-1/Latin1, text fallback and permissive Python float/NaN/Infinity loading remain unmatched. Supported explicit charsets now decode Resource/Wire while raw physical Envelope/Header/Status/Error.Body remain unchanged. Unsupported cp1006 or malformed charset declarations return ErrUnsupported+actualproof; nocharset/UTF8 bytes are cloned without unmarked nonUTF8 inference. Finite Go rawJSON retains number spelling/large integers rather than all Python loaded-value semantics. BOM-less bare utf-16/utf-32 native-endian selection remains unsupported.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--secret", default="s1")
    parser.add_argument("--attrs-json", default="{}")
    args = parser.parse_args()
    attrs = json.loads(args.attrs_json)
    with openstack.connect(cloud=args.cloud) as conn:
        result = conn.key_manager.update_secret(args.secret, **attrs)
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
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func run(ctx context.Context, cloud, id, attributes string) error {
    var attrs map[string]json.RawMessage
    if err := json.Unmarshal([]byte(attributes), &attrs); err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { return err }
    result, err := service.Secrets.UpdateRecord(ctx, id,
        secrets.WithUpdateRecordOpts(secrets.UpdateRecordOpts{Attributes: attrs}))
    if err != nil { return err }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Committed bool `json:"committed"`
        StatusCode int `json:"status_code"`
        Resource map[string]json.RawMessage `json:"resource"`
    }{result.Committed, result.StatusCode, result.Resource.Body})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    id := flag.String("secret", "s1", "literal secret ID")
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
python main.py --cloud devstack --secret s1 --attrs-json '{}'
go run main.go --cloud devstack --secret s1 --attrs-json '{}'
python main.py --cloud devstack --secret s1 --attrs-json '{"name":"new","expiration":null}'
go run main.go --cloud devstack --secret s1 --attrs-json '{"name":"new","expiration":null}'
```
