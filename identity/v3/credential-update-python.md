# Credential Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/update_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithRecordUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithRecordUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.WithRecordUpdateHeaders`

- PATCH on the literal generic /credentials member sends only declared name/blob/project_id/type/user_id under credential, preserves omitted/null/false/zero/empty JSON values, ignores unknown and id attrs, and overlays actual response fields on owned request/id seed while preserving unknown raw response fields and exact numbers.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdateRawBodyAndResponseOverlay`.
- Empty, unknown-only, id-only and request-policy-header-only updates issue no HTTP and return ID-only declared view without fabricated response evidence; explicit empty array/object values still cause PATCH.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdateNoOpAndLiteralID`.
- Source accepts 200..399, singular or root response objects and malformed/empty JSON seed retention; valid nonobject responses preserve accepted status/body proof, and 4xx/5xx/600 retain original native HTTP evidence without missing suppression.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdateTranslationAndStatuses`.
- Options own copied maps/raw JSON/headers and isolate retained callback pointers; invalid options, JSON, UTF-8, protected headers, context/client/ID fail before HTTP and callback errors join simultaneous cancellation or Source mutation.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdateOptionOwnershipAndPreflight`.
- Physical accepted and rejected Read/Close failures, context cancellation/cause and Source changes restored by Close keep status/body/header and original causes, close each body once and never replay the update automatically.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdatePhysicalResponseFailures`.
- Native retry policy retains the fixed PATCH body/route and response overlay; retry callback, body/Source mutation, cancellation and transport errors remain discoverable and invalid retries stop before a second physical request.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdateNativeRetryPolicyAndOwnership`.
- Literal UTF-8 member IDs containing slash/percent escape as exactly one URL segment.
  Tests: `identity/v3/credentials/python_update_test.go:TestPythonCredentialUpdateEscapesMemberExactlyOnce`.

Python/Go differences:

- The explicit Go profile accepts a literal string ID, owns copied concrete options and returns a fresh CredentialRecord; mutable Python Resource/Munch input, same-instance dirty comparison, Connection location and dynamic adapter/subclass/session hooks remain global Resource parity work.
- Unknown Credential attrs are ignored because the pinned Resource does not allow unknown Body attrs; native Update with WithUpdateField remains available for explicit Gophercloud extension bodies.
- ID options are ignored to keep the explicit literal target immutable; duplicate Python literal-ID/id-kwarg TypeError is not reproduced. Go always validates ID/context/client/headers even for an otherwise empty commit.
- WithRecordUpdateHeaders is an explicit Go request-policy extension rather than a Source dirty Header descriptor and does not alone trigger PATCH.
- The Go fixed request policy escapes one literal member segment and retains physical rejected Read/Close/cancel/Source causes that native Gophercloud otherwise loses.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.identity.update_credential(
    credential_id,
    blob='{"access":"example","secret":"replace-me"}',
)
print(updated.id, updated.blob)
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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "generic Credential ID")
    blob := flag.String("blob", "", "replacement credential blob string")
    clearProject := flag.Bool("clear-project", false, "send project_id=null")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    options := []credentials.RecordUpdateOption{
        credentials.WithRecordUpdateField("blob", *blob),
    }
    if *clearProject {
        options = append(options, credentials.WithRecordUpdateField("project_id", nil))
    }
    updated, err := service.Credentials.UpdateRecord(ctx, *id, options...)
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(updated.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Credentials Python](credentials-python.md)
- [Install](../../docs/install.md)
- [Python Update Test](credentials/python_update_test.go)
