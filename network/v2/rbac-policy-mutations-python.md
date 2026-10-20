# Rbac Policy Mutations Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_rbac_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordCreateMicroversion`

- Concrete owned attrs/header/version options snapshot bytes/maps/pointers and callback output; nil/malformed/constructor/route/Source preflight avoids HTTP while original marshal/callback causes survive.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationOptionsSnapshotAndMarshalCause`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationPreflightAndCallbackSourceGuard`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationResetOptionsAndDeterministicBulkAliases`.
- Actual every400..599 native rejection retains exact status/body/header; accepted200..399 malformed shape/UTF8 fails with received proof.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationAllRejectedStatusAndMalformedProof`.
- Combined read/Close/sticky Source/cancellation failures preserve every cause and receipt, stop retry/reauth callbacks, one physical request/Close; native401 pointer ErrUnableToReauthenticate ErrOriginal/ErrReauth fields retained.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationMixedPhysicalFaultsNeverRetryOrIgnore`.
- Clean native retry/live-token reauthentication allowed with fixed route/body/query/selected header/version/protocol; injected body/auth/foreign/version/conditions or omitted owned header abort before replay retaining original native receipt.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationCleanRetriesKeepBodyRouteAndLiveToken`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationRetryCannotChangeOwnedProtocol`.
- Source attrs microversion is consumed constructor metadata and does not override Adapter default; explicit Go version/default/empty override policy stays private with matching selected casefold aliases.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationVersionDefaultConstructorAndEmptyOverride`.
- Provided dirty attributes only, empty create still wrapped{}, untyped/null/false/zero/nested unknown attrs pass server-owned schema without native required fields; public-to-wire and sparse response overlay/defaults/actual wire/forged location receipt remain distinct.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationCreateProvidedAttrsAndEmptyEnvelope`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationAcceptedSparseOverlayAndRawNull`.
- Ordered canonical target_project_id/wire target_tenant aliases last supplied key wins; bulk maps sorted, tenant_id read fallback does not invent duplicate request field; invalid JSON/bodyless success retains seed/defaults.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationOrderedAliasesAndTenantFallback`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationResetOptionsAndDeterministicBulkAliases`.
- if_none_match/if_match Body kwargs differ from explicit actual safe HTTP Headers; descriptor schema has no Header coercion, selected response microversion projects actual version while raw Wire retains forged value.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationBodyHeaderAndVersionOwnership`.

Python/Go differences:

- Go takes immutable literal IDs and concrete JSON-domain option values. Mutable Python Resource dirty/cache/session/Connection location lifecycles and deprecated __conflicting_attrs/class shadowing remain separately tracked.
- Bulk maps have deterministic lexical key order; ordered With attrs retain insertion order and last canonical/wire alias wins. Nil RawMessage means explicit JSON null.
- SDK-owned constructor/route controls and unsafe literal IDs are preflight errors; Source arbitrary Resource/Adapter mutable controls are not emulated by this fixed immutable record profile.
- Explicit safe Header/Microversion options are Go conveniences. Source constructor microversion attrs do not become request overrides; selected client and casefold MoreHeaders remain unchanged.
- Raw exact-decimal JSON Body values are preserved; full CPython binary64 rounding/underflow/nonfinite Body normalization is not claimed.
- Completion200..399 is supported. Arbitrary adapter final informational100..199 remains outside completion profile. Actual400..599 and every observed physical/cancel/Source cause retain proof.
- Base stock Neutron27 mutation permissions are project ADMIN_OR_PROJECT_MEMBER; wildcard target * create/update is separate admin-only attribute branch. Operator/deprecated/default/extension/backend role grants require actual cloud evidence.

### `python:network/v2/delete_rbac_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordDeleteMicroversion`

- Concrete owned attrs/header/version options snapshot bytes/maps/pointers and callback output; nil/malformed/constructor/route/Source preflight avoids HTTP while original marshal/callback causes survive.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationOptionsSnapshotAndMarshalCause`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationPreflightAndCallbackSourceGuard`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationResetOptionsAndDeterministicBulkAliases`.
- Actual every400..599 native rejection retains exact status/body/header; accepted200..399 malformed shape/UTF8 fails with received proof.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationAllRejectedStatusAndMalformedProof`.
- Combined read/Close/sticky Source/cancellation failures preserve every cause and receipt, stop retry/reauth callbacks, one physical request/Close; native401 pointer ErrUnableToReauthenticate ErrOriginal/ErrReauth fields retained.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationMixedPhysicalFaultsNeverRetryOrIgnore`.
- Clean native retry/live-token reauthentication allowed with fixed route/body/query/selected header/version/protocol; injected body/auth/foreign/version/conditions or omitted owned header abort before replay retaining original native receipt.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationCleanRetriesKeepBodyRouteAndLiveToken`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationRetryCannotChangeOwnedProtocol`.
- Source attrs microversion is consumed constructor metadata and does not override Adapter default; explicit Go version/default/empty override policy stays private with matching selected casefold aliases.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationVersionDefaultConstructorAndEmptyOverride`.
- Bodyless literal member DELETE never performs hidden Get/Find or response JSON decode; exposes actual successful receipt and default clean404 IgnoreMissing=true with IgnoredMissing. False returns original native404 proof.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationDeleteMissingDefaultsAndReceipt`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationLiteralIdentityAndServerOwnedSchema`.
- Mixed physical/Source/cancel404, custom retry callback cause, or retry-expanded OkCodes404 is terminal and cannot establish logical missing. Every cause plus actual receipt remains observable.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationMixedPhysicalFaultsNeverRetryOrIgnore`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationDeleteCallbackAndExpandedStatusRemainTerminal`.

Python/Go differences:

- Go takes immutable literal IDs and concrete JSON-domain option values. Mutable Python Resource dirty/cache/session/Connection location lifecycles and deprecated __conflicting_attrs/class shadowing remain separately tracked.
- Bulk maps have deterministic lexical key order; ordered With attrs retain insertion order and last canonical/wire alias wins. Nil RawMessage means explicit JSON null.
- SDK-owned constructor/route controls and unsafe literal IDs are preflight errors; Source arbitrary Resource/Adapter mutable controls are not emulated by this fixed immutable record profile.
- Explicit safe Header/Microversion options are Go conveniences. Source constructor microversion attrs do not become request overrides; selected client and casefold MoreHeaders remain unchanged.
- Raw exact-decimal JSON Body values are preserved; full CPython binary64 rounding/underflow/nonfinite Body normalization is not claimed.
- Completion200..399 is supported. Arbitrary adapter final informational100..199 remains outside completion profile. Actual400..599 and every observed physical/cancel/Source cause retain proof.
- Base stock Neutron27 mutation permissions are project ADMIN_OR_PROJECT_MEMBER; wildcard target * create/update is separate admin-only attribute branch. Operator/deprecated/default/extension/backend role grants require actual cloud evidence.
- Source returns None. Go exposes actual deletion receipt and IgnoredMissing as explicit convenience; no decoded Resource is invented for DELETE.

### `python:network/v2/update_rbac_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordUpdateMicroversion`

- Concrete owned attrs/header/version options snapshot bytes/maps/pointers and callback output; nil/malformed/constructor/route/Source preflight avoids HTTP while original marshal/callback causes survive.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationOptionsSnapshotAndMarshalCause`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationPreflightAndCallbackSourceGuard`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationResetOptionsAndDeterministicBulkAliases`.
- Actual every400..599 native rejection retains exact status/body/header; accepted200..399 malformed shape/UTF8 fails with received proof.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationAllRejectedStatusAndMalformedProof`.
- Combined read/Close/sticky Source/cancellation failures preserve every cause and receipt, stop retry/reauth callbacks, one physical request/Close; native401 pointer ErrUnableToReauthenticate ErrOriginal/ErrReauth fields retained.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationMixedPhysicalFaultsNeverRetryOrIgnore`.
- Clean native retry/live-token reauthentication allowed with fixed route/body/query/selected header/version/protocol; injected body/auth/foreign/version/conditions or omitted owned header abort before replay retaining original native receipt.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationCleanRetriesKeepBodyRouteAndLiveToken`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationRetryCannotChangeOwnedProtocol`.
- Source attrs microversion is consumed constructor metadata and does not override Adapter default; explicit Go version/default/empty override policy stays private with matching selected casefold aliases.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationVersionDefaultConstructorAndEmptyOverride`.
- Literal escaped ID PUTs only provided dirty fields with no lookup/query/defaults/id body; empty/constructor-only update ProjectSeed no-op returns status0/Wire nil/no receipt, safe transport options do not force HTTP; explicit null is a dirty change.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationUpdateNoopAndExplicitNull`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationLiteralIdentityAndServerOwnedSchema`.
- Pinned Source unknown/stock-immutable attrs are transmitted and real400 schema rejection preserved rather than silently filtered or native required-target constrained; invalid or colliding identity/route controls reject preflight.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationLiteralIdentityAndServerOwnedSchema`, `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationPreflightAndCallbackSourceGuard`.
- Explicit actual HTTP If-None-Match/If-Match headers remain independent of Body attributes/null and selected response microversion; captures matching header aliases without Source mutation.
  Tests: `network/v2/extensions/rbacpolicies/python_mutations_contracts_test.go:TestPythonRBACMutationBodyHeaderAndVersionOwnership`.

Python/Go differences:

- Go takes immutable literal IDs and concrete JSON-domain option values. Mutable Python Resource dirty/cache/session/Connection location lifecycles and deprecated __conflicting_attrs/class shadowing remain separately tracked.
- Bulk maps have deterministic lexical key order; ordered With attrs retain insertion order and last canonical/wire alias wins. Nil RawMessage means explicit JSON null.
- SDK-owned constructor/route controls and unsafe literal IDs are preflight errors; Source arbitrary Resource/Adapter mutable controls are not emulated by this fixed immutable record profile.
- Explicit safe Header/Microversion options are Go conveniences. Source constructor microversion attrs do not become request overrides; selected client and casefold MoreHeaders remain unchanged.
- Raw exact-decimal JSON Body values are preserved; full CPython binary64 rounding/underflow/nonfinite Body normalization is not claimed.
- Completion200..399 is supported. Arbitrary adapter final informational100..199 remains outside completion profile. Actual400..599 and every observed physical/cancel/Source cause retain proof.
- Base stock Neutron27 mutation permissions are project ADMIN_OR_PROJECT_MEMBER; wildcard target * create/update is separate admin-only attribute branch. Operator/deprecated/default/extension/backend role grants require actual cloud evidence.
- This leaf updates an immutable literal-ID seed. Source updating an existing mutable Resource with previously clean/dirty attrs is a separate remaining lifecycle contract. Empty dirty Body is a no-op even with transport-only header/version options.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--mode", choices=("create", "update", "delete"), required=True)
parser.add_argument("--policy", default="")
parser.add_argument("--object", default="")
parser.add_argument("--target-project", default="")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.network.create_rbac_policy(
            object_type="network", object_id=args.object,
            action="access_as_shared", target_project_id=args.target_project,
        )
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "update":
        value = conn.network.update_rbac_policy(
            args.policy, target_project_id=args.target_project
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.network.delete_rbac_policy(args.policy)
        print(json.dumps({"deleted": args.policy}, indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
)

func run(ctx context.Context, cloud, mode, policyID, objectID, target string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "create":
        value, err := network.RBACPolicies.CreateRecord(ctx,
            rbacpolicies.WithRecordCreateField("object_type", "network"),
            rbacpolicies.WithRecordCreateField("object_id", objectID),
            rbacpolicies.WithRecordCreateField("action", "access_as_shared"),
            rbacpolicies.WithRecordCreateField("target_project_id", target),
        )
        if err != nil { return nil, err }
        return value.Resource, nil
    case "update":
        value, err := network.RBACPolicies.UpdateRecord(ctx, policyID,
            rbacpolicies.WithRecordUpdateField("target_project_id", target),
        )
        if err != nil { return nil, err }
        return value.Resource, nil
    case "delete":
        return network.RBACPolicies.DeleteRecord(ctx, policyID)
    default:
        return nil, fmt.Errorf("mode must be create, update or delete")
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create, update or delete")
    policyID := flag.String("policy", "", "literal RBAC policy ID")
    objectID := flag.String("object", "", "network ID for create")
    target := flag.String("target-project", "", "concrete target project ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *policyID, *objectID, *target)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [Install](../../docs/install.md)
- [Rbac](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py#L89)
- [owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L32)
- [Stock RBAC schema](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/rbac.py#L60)
