# Metadeftags

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/add_tag_to_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.NamespaceScope.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithCreateHeaders`

- InNamespace is zeroHTTP and captures required literal namespace plus original API/client/provider/type/endpoint/base/microversion for scope lifetime. Parent and route child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected, colons/Unicode/spaces preserved and escaped once. No parent lookup/Ref/find/search/returned identity routing. Before callbacks and after accepted handling source/context stay fixed, per-request ordinary headers are newly copied and authentication remains live from original provider.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCompletePreflight`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`.
- Create Tag requires whole validUTF8 nonnullJSONobject with exact-case optional nullable Name and literal CreatedAt/UpdatedAt strings; canonical wrong nonnull types fail atomically. Unknown fields and large JSON precision stay independently owned in Body; no ID/parent/self/schema/links/date seed or parser. Returned Name is passive and cannot retarget later calls.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCanonicalPresenceAndAtomicDecoder`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsCanonicalModelsAndRawOwnership`.
- Directly related Create/Delete/DeleteAll header-only concrete families use FullOpts whole replacement, lastwins and independent header/helper/config snapshots before and after callbacks. Options slice copied and callback once per request; nil/error callback and invalid/conflicting owned headers fail before HTTP. Seven total SDK families are proved separately; no unrelated Set/List option behavior is assigned to these Python rows.
  Tests: `image/v2/metadeftags/options_test.go:TestMetadefTagsOptionsValidationAndDefaults`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`.
- Original Create201 and Delete/DeleteAll204 are strict. Accepted bytes/Header/status and read/Close/contextErr/customCause/source handling causes are preserved in ResponseError, close once/no replay. Actual204 opaque Acknowledgement carries requested Namespace and literal child Name pointer, nil Name for bulk, and remains available with handling errors. Native rejected response ownership remains native.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect. Exact method/origin/path/query/serialized body and owned response fields are guarded; expanded native OkCodes cannot change original accepted status. Read/Close/ctx.Err/customCause and native ErrOriginal/ErrReauth remain evidence; no accepted-body replay or private404 retry bypass. Existing advanced native RetryFunc.MoreHeaders mutation stays allowed; option/source ordinary header restrictions do not promise provider-header hook rejection.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- One bodyless individual POST201 at fixed namespace/tags/name, matching Namespace.add_tag override rather than common PUT/imageTag. No JSON name body, discovery/fetch/cache merge/rollback. Input literal route80 bound is explicitGo behavior; server individual schema/SQL supplementaryUnicode/protection/duplicate checks stay server-owned. Python returns None through proxy while its Namespace method appends input to mutable cache and returns same Resource without decoding actual201 body.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCompletePreflight`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsFixedScopedRoutesAndPayloads`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`, `internal/cmd/sdkgen/glance_metadef_tags_test.go:TestGlanceMetadefTagsRegistryAndConcreteScope`.
- Create sends a bodyless POST metadefs/namespaces/{namespace}/tags/{tag}, the request Python add_tag sends.
  Tests: `image/v2/metadeftags/contracts_test.go:TestMetadefTagsFixedScopedRoutesAndPayloads`.

Python/Go differences:

- Pinned Python has only three namespace-tag write proxies, generic NamespaceResource reuse/bare-id construction, mutable cache and adapter/session/microversion behavior; Go fixed scoped identity/raw DTO/original-status gate is a bounded difference.
- Python response exception handling accepts status<400 and add/remove/all mutate cached tags rather than decode canonicalTag/opaqueACK. Go validates UTF8/canonical types and exact201/204 while preserving accepted raw evidence.
- Server schema80, SQL supplementaryUnicode/length/collation/duplicate/transaction/protection/policy and authenticatedcloud effects remain unexecuted; local Go fixtures are not Python execution or fullcloud parity.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Python updates the cached tags list on its namespace Resource and returns None; Go returns the created tag or the 204 acknowledgement.

### `python:image/v2/remove_tag_from_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.NamespaceScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithDeleteHeaders`

- InNamespace is zeroHTTP and captures required literal namespace plus original API/client/provider/type/endpoint/base/microversion for scope lifetime. Parent and route child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected, colons/Unicode/spaces preserved and escaped once. No parent lookup/Ref/find/search/returned identity routing. Before callbacks and after accepted handling source/context stay fixed, per-request ordinary headers are newly copied and authentication remains live from original provider.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCompletePreflight`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`.
- Directly related Create/Delete/DeleteAll header-only concrete families use FullOpts whole replacement, lastwins and independent header/helper/config snapshots before and after callbacks. Options slice copied and callback once per request; nil/error callback and invalid/conflicting owned headers fail before HTTP. Seven total SDK families are proved separately; no unrelated Set/List option behavior is assigned to these Python rows.
  Tests: `image/v2/metadeftags/options_test.go:TestMetadefTagsOptionsValidationAndDefaults`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`.
- Original Create201 and Delete/DeleteAll204 are strict. Accepted bytes/Header/status and read/Close/contextErr/customCause/source handling causes are preserved in ResponseError, close once/no replay. Actual204 opaque Acknowledgement carries requested Namespace and literal child Name pointer, nil Name for bulk, and remains available with handling errors. Native rejected response ownership remains native.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect. Exact method/origin/path/query/serialized body and owned response fields are guarded; expanded native OkCodes cannot change original accepted status. Read/Close/ctx.Err/customCause and native ErrOriginal/ErrReauth remain evidence; no accepted-body replay or private404 retry bypass. Existing advanced native RetryFunc.MoreHeaders mutation stays allowed; option/source ordinary header restrictions do not promise provider-header hook rejection.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- One strict individual DELETE204 at required fixed child path. No IgnoreMissing helper/default, swallowed404 or private404 ownership; native rejected404/retry behavior retained. Parent/tag forbidden can be concealed404, so absence cannot be inferred. Python inherited remove_tag also raises HTTP404 and only catches local cache-list.remove ValueError; it mutates cache and proxy returnsNone. No parentGET/check/RMW/unprotect/wait/cleanup.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCompletePreflight`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsFixedScopedRoutesAndPayloads`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- Delete sends DELETE metadefs/namespaces/{namespace}/tags/{tag} and keeps 404 as an error, as Python remove_tag does.
  Tests: `image/v2/metadeftags/contracts_test.go:TestMetadefTagsFixedScopedRoutesAndPayloads`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`.

Python/Go differences:

- Pinned Python has only three namespace-tag write proxies, generic NamespaceResource reuse/bare-id construction, mutable cache and adapter/session/microversion behavior; Go fixed scoped identity/raw DTO/original-status gate is a bounded difference.
- Python response exception handling accepts status<400 and add/remove/all mutate cached tags rather than decode canonicalTag/opaqueACK. Go validates UTF8/canonical types and exact201/204 while preserving accepted raw evidence.
- Server schema80, SQL supplementaryUnicode/length/collation/duplicate/transaction/protection/policy and authenticatedcloud effects remain unexecuted; local Go fixtures are not Python execution or fullcloud parity.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Python updates the cached tags list on its namespace Resource and returns None; Go returns the created tag or the 204 acknowledgement.

### `python:image/v2/remove_tags_from_metadef_namespace`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.API.InNamespace`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.NamespaceScope.DeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithDeleteAllOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithDeleteAllHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags.WithDeleteAllHeaders`

- InNamespace is zeroHTTP and captures required literal namespace plus original API/client/provider/type/endpoint/base/microversion for scope lifetime. Parent and route child are nonempty validUTF8<=80runes with route-breaking/control/dot syntax rejected, colons/Unicode/spaces preserved and escaped once. No parent lookup/Ref/find/search/returned identity routing. Before callbacks and after accepted handling source/context stay fixed, per-request ordinary headers are newly copied and authentication remains live from original provider.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCompletePreflight`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`.
- Directly related Create/Delete/DeleteAll header-only concrete families use FullOpts whole replacement, lastwins and independent header/helper/config snapshots before and after callbacks. Options slice copied and callback once per request; nil/error callback and invalid/conflicting owned headers fail before HTTP. Seven total SDK families are proved separately; no unrelated Set/List option behavior is assigned to these Python rows.
  Tests: `image/v2/metadeftags/options_test.go:TestMetadefTagsOptionsValidationAndDefaults`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsOwnedPreparationAndSource`.
- Original Create201 and Delete/DeleteAll204 are strict. Accepted bytes/Header/status and read/Close/contextErr/customCause/source handling causes are preserved in ResponseError, close once/no replay. Actual204 opaque Acknowledgement carries requested Namespace and literal child Name pointer, nil Name for bulk, and remains available with handling errors. Native rejected response ownership remains native.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- Shared REST/fixedrequest preserves configured native prebody retry/reauth/backoff and identical-target redirect. Exact method/origin/path/query/serialized body and owned response fields are guarded; expanded native OkCodes cannot change original accepted status. Read/Close/ctx.Err/customCause and native ErrOriginal/ErrReauth remain evidence; no accepted-body replay or private404 retry bypass. Existing advanced native RetryFunc.MoreHeaders mutation stays allowed; option/source ordinary header restrictions do not promise provider-header hook rejection.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- One strict collection DELETE204 with no IgnoreMissing and nil childName in actual opaque ACK. Native404/policy/close/context causes remain errors. Server checks namespaceDelete and tagsDelete policy/protection; no automaticunprotect/enumerate/count/transaction/wait or cleanup. Python remove_all_tags clears mutable localcache after HTTPstatus<400 and proxy returnsNone; Go exposes actual ACK/evidence.
  Tests: `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreFixedScopeAndRoutes`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreCompletePreflight`, `image/v2/metadeftags/core_test.go:TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsFixedScopedRoutesAndPayloads`, `image/v2/metadeftags/contracts_test.go:TestMetadefTagsResponseOwnershipAndNativeHooks`.
- DeleteAll sends one DELETE metadefs/namespaces/{namespace}/tags, the request Python remove_all_tags sends.
  Tests: `image/v2/metadeftags/contracts_test.go:TestMetadefTagsFixedScopedRoutesAndPayloads`.

Python/Go differences:

- Pinned Python has only three namespace-tag write proxies, generic NamespaceResource reuse/bare-id construction, mutable cache and adapter/session/microversion behavior; Go fixed scoped identity/raw DTO/original-status gate is a bounded difference.
- Python response exception handling accepts status<400 and add/remove/all mutate cached tags rather than decode canonicalTag/opaqueACK. Go validates UTF8/canonical types and exact201/204 while preserving accepted raw evidence.
- Server schema80, SQL supplementaryUnicode/length/collation/duplicate/transaction/protection/policy and authenticatedcloud effects remain unexecuted; local Go fixtures are not Python execution or fullcloud parity.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.
- Go validates literal route names and accepts only the documented 201/200/204 statuses; Python accepts any status below 400.
- Python updates the cached tags list on its namespace Resource and returns None; Go returns the created tag or the 204 acknowledgement.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags"
)

type TagEvidence struct {
    Created *metadeftags.Tag
    Read *metadeftags.Tag
    Updated *metadeftags.Tag
    Set *metadeftags.SetResult
    Rows []*metadeftags.Tag
    All []*metadeftags.Tag
    Deleted *metadeftags.Acknowledgement
    Cleared *metadeftags.Acknowledgement
}

func manageNamespaceTags(ctx context.Context, service *image.Service, namespace, current, replacement string, setNames []string, clearNamespace bool) (out TagEvidence, err error) {
    scope, err := service.API.MetadefTags.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    out.Created, err = scope.Create(ctx, current,
        metadeftags.WithCreateOpts(metadeftags.CreateOpts{}),
        metadeftags.WithCreateHeader("X-Request-Source", "example"),
        metadeftags.WithCreateHeaders(map[string]string{"X-Trace": "create"}))
    if err != nil { return out, err }
    out.Read, err = scope.Get(ctx, current,
        metadeftags.WithGetOpts(metadeftags.GetOpts{}),
        metadeftags.WithGetHeader("X-Request-Source", "example"),
        metadeftags.WithGetHeaders(map[string]string{"X-Trace": "get"}))
    if err != nil { return out, err }
    out.Updated, err = scope.Update(ctx, current,
        metadeftags.WithUpdateOpts(metadeftags.UpdateOpts{}),
        metadeftags.WithUpdateHeader("X-Request-Source", "example"),
        metadeftags.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadeftags.WithUpdateName(replacement))
    if err != nil { return out, err }
    out.Set, err = scope.Set(ctx, setNames,
        metadeftags.WithSetOpts(metadeftags.SetOpts{}),
        metadeftags.WithSetHeader("X-Request-Source", "example"),
        metadeftags.WithSetHeaders(map[string]string{"X-Trace": "set"}),
        metadeftags.WithSetAppend(true))
    if err != nil { return out, err }

    listOptions := []metadeftags.ListOption{
        metadeftags.WithListOpts(metadeftags.ListOpts{}),
        metadeftags.WithListHeader("X-Request-Source", "example"),
        metadeftags.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadeftags.WithListLimit(2),
        metadeftags.WithListMarker(""),
        metadeftags.WithListSortKey("name"),
        metadeftags.WithListSortDir("asc"),
        metadeftags.WithListMaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }
    out.Deleted, err = scope.Delete(ctx, replacement,
        metadeftags.WithDeleteOpts(metadeftags.DeleteOpts{}),
        metadeftags.WithDeleteHeader("X-Request-Source", "example"),
        metadeftags.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadeftags.WithDeleteAllOpts(metadeftags.DeleteAllOpts{}),
        metadeftags.WithDeleteAllHeader("X-Request-Source", "example"),
        metadeftags.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

```python
from openstack.image.v2.metadef_namespace import MetadefNamespace


def proxy_namespace_tags(conn, namespace, tag, clear_namespace=False):
    conn.image.add_tag_to_metadef_namespace(namespace, tag)
    conn.image.remove_tag_from_metadef_namespace(namespace, tag)
    if clear_namespace:
        conn.image.remove_tags_from_metadef_namespace(namespace)


def resource_namespace_tags(conn, namespace, names):
    resource = MetadefNamespace.new(id=namespace)
    resource.set_tags(conn.image, names, append=True)
    resource.fetch_tags(conn.image)
    if names:
        resource.check_tag(conn.image, names[0])
    return resource
```

## References

- [Router](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L355-L396)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1487-L1539)
- [Namespace.add_tag·set_tags](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py#L109-L152)
- [inherited TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L28-L142)
- [Metadef Tags](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L288-L337)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#create-tags)
- [SQL create_tags](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/metadef_api/tag.py#L114-L141)
- [index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L139-L183)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-tags)
- [deserializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L411-L429)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection Image Metadef Tags Test](../../../connectiontest/connection_image_metadef_tags_test.go)
- [Glance Metadef Tags Test](../../../internal/cmd/sdkgen/glance_metadef_tags_test.go)
