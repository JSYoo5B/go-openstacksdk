# Quotas

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/get_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithGetHeader`

- Pinned _proxy.py:457 uses requires_id=False. The singleton GET quotas accepts only 200 with a non-null quotas object and uses the provider token current project; no project/name lookup or implicit token-project inference from configuration is submitted. Raw missing/null/large integer/extension/case-variant values and actual metadata are preserved.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasFourOperationsKeepFixedProjectAndLiveProvider`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`.
- Wrong/missing/null/array envelopes and malformed accepted JSON/read failures retain ResponseError body/header/status and cause without an SDK composition resend. Unexpected GET203 retains native cause; nil/invalid source, shared GET options validation and canceled/nil context fail before HTTP. Effective GET preflight preserves both context.Canceled and the caller's custom cancellation cause.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasStrictGetEnvelopesStatusAndReadEvidence`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasInvalidOptionsAndScopePreflight`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasHeadersSnapshotReuseAndCancellation`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. ResourceBase, live token rotation and configured source headers remain effective; cancellation does not perform HTTP. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or Resources CRUD/Find/Wait capabilities.
  Tests: `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.
- Actual effective GET404/403 and unexpected203 use one fixed GET quotas request without body/query, project/name lookup or default fallback under the fixture's default provider policy. Native expected200/actual status, exact URL/method, body/header and Get/quotas operation cause remain observable without false accepted-response classification. An accepted HTTP200 Close failure preserves its complete original body/header/status and custom cause, reads the body and closes it once, invokes no native retry callback and causes no SDK composition resend.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasStrictGetEnvelopesStatusAndReadEvidence`.

Python/Go differences:

- Go maps the input-free public getter to API.Get(ctx), using a concrete context and configured authenticated key-manager client. WithGetHeader is an additional owned Go option; it does not introduce project lookup, body/query injection, a quota identifier or a new List/Find/Wait surface.
- Go deliberately requires the published HTTP200 success and a non-null quotas object envelope. Pinned Resource._translate_response at resource.py:1338-1399 unwraps quotas when present but can also consume a root object and suppresses JSON ValueError; exceptions.raise_from_response at exceptions.py:214-222 permits statuses below400. Go treats unexpected success status, wrong/missing envelope and malformed accepted JSON as errors with actual evidence rather than returning an empty or differently shaped resource.
- Go returns independently owned raw response fields, Body/Data and Header/StatusCode rather than a mutable Python Quota Resource with dynamic descriptors, cached state or seeded-instance merges. Omitted json.RawMessage is nil and JSON null is retained as null bytes; unknown/case-variant data and exact large JSON numbers remain observable.
- The named Python get_quota() exposes no per-call Resource.fetch session, adapter, base_path, microversion, params or skip_cache overload. Go retains its configured shared provider, source path, transport and live authentication; optional inherited GET cache and generic Resource/session lifecycle remain separately tracked SDK scope rather than incomplete declared getter inputs. No class-specific maximum-microversion negotiation is inferred from Quota.
- Completion is restricted to python:key_manager/v1/get_quota. ProjectQuota get/update/delete, SecretStore singleton/list operations, other native declarations and full inherited Resource/session mappings retain their independent reviews and goal scope.

### `python:key_manager/v1/get_project_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.ProjectScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.ProjectScope.ProjectID`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithGetHeader`

- Pinned _proxy.py:421 extracts the project ID. InProject constructs an explicit-ID scope with zero HTTP; GET project-quotas/{escaped id} accepts only 200 with a project_quotas object. Unicode/plus IDs are safely escaped, incidental response project/id values never retarget the scope, and the original ResourceBase/current provider token remain effective.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasFourOperationsKeepFixedProjectAndLiveProvider`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`.
- Wrong configured-quota envelope retains accepted evidence. Invalid/unsupported project references, nil/invalid source, protected or unsupported request extensions and canceled context fail before HTTP; the common strict decode/read/no-resend policy remains intact.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasStrictGetEnvelopesStatusAndReadEvidence`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasInvalidOptionsAndScopePreflight`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasHeadersSnapshotReuseAndCancellation`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. ResourceBase, live token rotation and configured source headers remain effective; cancellation does not perform HTTP. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or Resources CRUD/Find/Wait capabilities.
  Tests: `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.
- Get sends one GET project-quotas/{project} and exposes secrets, orders, containers, consumers and cas from project_quotas, as Python _get(ProjectQuota) does.
  Tests: `keymanager/v1/quotas/python_rejudge_test.go:TestPythonRejudgeKeyManagerProjectQuotaGetRequest`.

Python/Go differences:

- GET fields secrets/orders/containers/consumers/cas are json.RawMessage rather than mutable untyped Python descriptors: omitted/null/large integer/unknown values and exact canonical-versus-case-variant keys remain in owned Body/Data, with actual response headers/status.
- ProjectScope accepts an explicit safe resource.ID only, without Keystone name lookup or accepting a Python Project Resource. Its fixed escaped request project is independent of response fields and the provider token project; the original ServiceClient/Provider remains shared.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Go has no client-side GET response cache or skip_cache control; every call sends its request.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:key_manager/v1/update_project_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.ProjectScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateSecrets`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateOrders`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateContainers`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateConsumers`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateCAs`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithUpdateHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.ProjectScope.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.ProjectScope.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithRecordMicroversion`

- PUT project-quotas/{fixed escaped id} sends one project_quotas envelope, accepts only 204 and replaces overrides: omitted fields use server defaults, empty typed options sends an empty object, and explicit 0/-1/-2/large int64 values remain present. There is no read-modify-write or automatic follow-up GET and no response-ID/token-project retargeting.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasFourOperationsKeepFixedProjectAndLiveProvider`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasReplacementPresenceAndOwnedOptions`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`.
- The SDK owns Optional[int64] field presence, options snapshots, last-wins helpers/bulk replacement and reusable case-insensitive header snapshots. The stock integer-only schema policy rejects null, bool, fraction, numeric string, out-of-int64 input and unknown/core body extensions rather than delegating unsafe untyped attrs; no minimum beyond the published integer schema is fabricated. Invalid source/options/context fail before HTTP and source mutations by options are rechecked.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasReplacementPresenceAndOwnedOptions`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasInvalidOptionsAndScopePreflight`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasHeadersSnapshotReuseAndCancellation`.
- UpdateResult is the actual accepted204 body/header/status acknowledgement, not a configured Quota synthesized from request inputs. Unexpected status preserves native cause; accepted read failure preserves actual response evidence, transport/context causes remain observable and no failure is resent.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasUpdateAcknowledgementAndTerminalFailures`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. ResourceBase, live token rotation and configured source headers remain effective; cancellation does not perform HTTP. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or Resources CRUD/Find/Wait capabilities.
  Tests: `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.
- Source scalar-ID UpdateRecord sends raw declared dirty values, including null/string/bool, in one project_quotas PUT; unknown attrs are ignored before encoding, empty changes no-op without fake proof, and nullable response fields overlay the independent seed. Existing strict Update reset204 behavior is unchanged.
  Tests: `keymanager/v1/quotas/python_project_records_test.go:TestPythonProjectQuotaRecordRawUpdateNullAndNoop`, `keymanager/v1/quotas/python_project_records_test.go:TestPythonProjectQuotaRecordOwnOptionsAndMerge`.
- Owned options preserve bytes/maps, constructor collisions and unsupported getter attrs fail before HTTP, source/context faults remain terminal and rejected/accepted Close failures keep their native/response receipt with one attempt.
  Tests: `keymanager/v1/quotas/python_project_records_test.go:TestPythonProjectQuotaRecordInvalidInputSourceAndCancellation`, `keymanager/v1/quotas/python_project_records_test.go:TestPythonProjectQuotaRecordFaultsKeepReceiptAndNeverReplay`.

Python/Go differences:

- Legacy strict integer-only ProjectScope.Update remains a reset/204 acknowledgement API. The new UpdateRecord is the explicit raw/null/seeded/no-dirty Source scalar-ID commit profile, including arbitrary accepted100..399 response evidence.
- The Python public update_project_quota extracts a project ID before Resource construction; Go accepts an explicit fixed safe project scope rather than arbitrary Project runtime objects. Headers/microversion are explicit concrete options, while arbitrary per-call base_path/runtime controls are omitted.
- Full mutable Resource same-object/dirty re-commit, adapter/cache/location/Requests behavior and SDK-R1/C1/S1/live-cloud/release boundaries remain whole-SDK differences; no inherited runtime parity is inferred from owned JSON snapshots.

### `python:key_manager/v1/delete_project_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.ProjectScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas.WithDeleteHeader`

- DELETE project-quotas/{fixed escaped id} accepts204, resets configured overrides and returns error-only without an extra GET or synthesized quota. Default ignoreMissing=true suppresses actual HTTP404 only; explicit false preserves native404, while403/409/503/unexpected200 remain errors. The fixed project is independent of response/token project.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasFourOperationsKeepFixedProjectAndLiveProvider`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasMissingDeleteAndNativeFailures`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`.
- A transport failure wrapping a native404 and an accepted204 body-read failure wrapping a native404 remain terminal despite default missing suppression. ResponseError keeps accepted body/header/status and native cause; no wrapped failure is resent. Nil/invalid source, unsupported query/body/protected headers and canceled context fail before HTTP.
  Tests: `api/keymanager_quotas_test.go:TestKeyManagerQuotasMissingDeleteAndNativeFailures`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasInvalidOptionsAndScopePreflight`, `api/keymanager_quotas_test.go:TestKeyManagerQuotasHeadersSnapshotReuseAndCancellation`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. ResourceBase, live token rotation and configured source headers remain effective; cancellation does not perform HTTP. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or Resources CRUD/Find/Wait capabilities.
  Tests: `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connectiontest/connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.
- Delete sends DELETE project-quotas/{project}; by default 204 and 404 both return nil like Python ignore_missing=True, and WithDeleteIgnoreMissing(false) surfaces the 404.
  Tests: `keymanager/v1/quotas/python_rejudge_test.go:TestPythonRejudgeKeyManagerProjectQuotaDeleteRequest`.

Python/Go differences:

- ProjectScope accepts an explicit safe resource.ID only, without Keystone name lookup or accepting a Python Project Resource. Its fixed escaped request project is independent of response fields and the provider token project; the original ServiceClient/Provider remains shared.
- The direct Python default ignore_missing=True and error-only success are retained; explicit Go missing/header options replace kwargs. Only actual HTTP absence is ignored, preserving transport/accepted-response errors containing a404 cause.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
quota = conn.key_manager.get_quota()
print(quota.secrets)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "net/http"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    timeout := flag.Duration("timeout", time.Minute, "authentication and quota query timeout")
    flag.Parse()
    if err := run(*cloud, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud string, timeout time.Duration) error {
    if timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return err
    }
    quota, err := service.Quotas.Get(ctx)
    if err != nil {
        return err
    }

    output := struct {
        StatusCode int                        `json:"status_code"`
        Header     http.Header                `json:"header"`
        Quotas     map[string]json.RawMessage  `json:"quotas"`
    }{
        StatusCode: quota.StatusCode,
        Header: quota.Header,
        Quotas: quota.Body,
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}
```

```go
package quotaexample

import (
    "context"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Configure(ctx context.Context, client *gophercloud.ServiceClient, projectID string) error {
    api := quotas.New(client)
    effective, err := api.Get(ctx)
    if err != nil { return err }
    fmt.Println(string(effective.Secrets))

    project, err := api.InProject(ctx, resource.ID(projectID))
    if err != nil { return err }
    configured, err := project.Get(ctx)
    if err != nil { return err }
    fmt.Println(string(configured.Orders))

    accepted, err := project.Update(ctx, quotas.UpdateOpts{
        Secrets: request.Present[int64](100),
    }, quotas.WithUpdateOrders(0), quotas.WithUpdateContainers(-1),
        quotas.WithUpdateHeader("X-Request-ID", "quota-change"))
    if err != nil { return err }
    fmt.Println(accepted.StatusCode, accepted.Header.Get("X-Request-ID"))

    return project.Delete(ctx, quotas.WithDeleteIgnoreMissing(false))
}
```

## References

- [Python Project Records](python-project-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L397)
- [Barbican quota API](https://docs.openstack.org/api-ref/key-manager/v1/#quotas)
- [ProjectQuotaValidator](https://github.com/openstack/barbican/blob/master/barbican/common/validators.py)
- [Keymanager Quotas Test](../../../api/keymanager_quotas_test.go)
- [public get_quota](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L457)
- [Quota fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/quota.py#L16)
- [Connection Keymanager SDK Owned Test](../../../connectiontest/connection_keymanager_sdk_owned_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
