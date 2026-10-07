# Swift container 목록

`conn.ObjectStorage(ctx)`가 반환한 서비스에서 `service.Containers.Resources`를 사용합니다. container는 이름이 식별자이며 `resource.ID(name)`은 원래 이름을 받습니다. SDK가 URL을 escape하므로 호출자가 미리 인코딩하지 않습니다. `resource.Name(name)` 조회는 prefix 목록에서 정확한 이름을 비교합니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| 전체 목록 | `conn.object_store.containers()` | `service.Containers.Resources.List(ctx)` |
| 읽을 행 수 제한 | `containers(max_items=20)` | 목록에 `resource.WithMaxItems(20)` |
| 첫 페이지만 읽기 | pinned proxy는 `paginated=True`를 고정 | 목록에 `resource.WithPaginated(false)` |
| metadata 조회 | `get_container_metadata(name)` | `service.Containers.GetMetadata(ctx, name)` |

`WithMaxItems(0)`은 무제한이며 음수는 iterator 순회 시 HTTP 전에 오류를 반환합니다. 같은 옵션은 마지막 값이 적용됩니다. 제한은 서버 응답 행을 세므로 `WithName`의 정확한 로컬 비교보다 먼저 적용합니다. prefix query 결과 중 이름이 다른 항목도 cap을 소비합니다. native 경로는 cap에서 `limit`을 추정하지 않으며, 아래 `WithPageSize(100)`만 서버 페이지 크기를 요청합니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func ListContainers(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	for container, err := range service.Containers.Resources.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(container.Name, container.Count, container.Bytes)
	}
	return nil
}
```

목록의 `ContainerResource`는 container 이름·object 수·bytes를 보존합니다. `Details`, 사용자 `Metadata`, 전체 `Header`는 nil이며, 목록에서 자동 HEAD를 실행하지 않습니다. `Resources.Get`은 HEAD로 이 값을 채웁니다. 상태 필드가 없으므로 `WithStatus`와 상태 `Wait`는 `resource.ErrUnsupported`입니다.

`break`와 context 취소는 추가 페이지 요청을 중단합니다. native Swift marker pagination과 반복 링크 검사를 유지하며, 현재 페이지의 전체 decode 오류는 cap 이후 행에서도 관찰할 수 있습니다. 첫 페이지 옵션은 다음 marker를 처리하기 전에 멈춥니다. 기존 공개 `service.Containers.List(ctx, containers.WithListOptions(...))`와 typed query 입력은 유지됩니다. 이 문서의 두 로컬 제어는 `Resources.List/All` 옵션입니다.

Python proxy의 첫 페이지 차이와 전체 공통 정책은 [목록 가이드](../../../docs/listing.md), object scope는 [Swift object 사용법](../objects/README.md), HTTP 증거는 [native 범위 목록 계약](../../../api/native_scope_list_controls_test.go)와 [Swift 목록 계약](../../../api/swift_listing_contracts_test.go)을 참고합니다.


## Container metadata 조회·설정·삭제

`service.Containers.GetMetadata(ctx, name)`는 body 없는 HEAD를, `SetMetadata`와 `DeleteMetadata`는 body 없는 POST를 보냅니다. 새 세 메서드는 실제 204만 받아들이며 container 생성, 이름 lookup, 목록, missing 무시 또는 자동 refresh를 수행하지 않습니다. 기존 `Resources.Get`, native/generated `Get`/`Update`는 그대로 사용할 수 있습니다.

[pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L189)는 literal 이름 또는 mutable Container를 받습니다. setter는 [기본 `refresh=True`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)에 따라 POST 뒤 HEAD를 수행하고 같은 Container를 반환합니다. deleter는 빈 metadata value를 POST하고 `None`을 반환합니다.

```python
container = conn.object_store.get_container_metadata("books")
print(container.metadata.get("book"))
refreshed = conn.object_store.set_container_metadata(
    container, book="Moby Dick", note="literal %2F 雪"
)
# 기본 setter는 POST 다음 HEAD를 수행하며 refreshed는 같은 Container입니다.
conn.object_store.delete_container_metadata(refreshed, ["note"])
# HTTP DELETE가 아니라 빈 X-Container-Meta-Note 값을 보내는 POST입니다.
```

Go setter의 반환값은 POST acknowledgement입니다. 변경된 metadata가 필요할 때 명시적으로 `GetMetadata`를 호출합니다. 첫 HEAD는 `X-Newest: false`, refresh HEAD는 옵션을 비워 server 기본값을 사용합니다.

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/containers"
)

func changeContainerMetadata(ctx context.Context, api *containers.API, name string) error {
    getOptions := []containers.GetMetadataOption{
        containers.WithGetMetadataOpts(containers.GetMetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        containers.WithGetMetadataHeader("X-Request-Label", "container-metadata"),
        containers.WithGetMetadataNewest(false),
    }
    before, err := api.GetMetadata(ctx, name, getOptions...)
    if before != nil { fmt.Println(before.StatusCode) }
    if err != nil { return err }
    if before.Metadata.ObjectCount != nil { fmt.Println(*before.Metadata.ObjectCount) }

    mutationOptions := []containers.MetadataOption{
        containers.WithMetadataOpts(containers.MetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        containers.WithMetadataHeader("X-Request-Label", "metadata-mutation"),
    }
    acknowledgement, err := api.SetMetadata(ctx, name, map[string]string{
        "book": "Moby Dick", "note": "literal %2F 雪",
    }, mutationOptions...)
    if acknowledgement != nil { fmt.Println(acknowledgement.StatusCode) }
    if err != nil { return err }

    refreshOptions := append([]containers.GetMetadataOption(nil), getOptions...)
    refreshOptions = append(refreshOptions, containers.WithoutGetMetadataNewest())
    refreshed, err := api.GetMetadata(ctx, name, refreshOptions...)
    if err != nil { return err }
    fmt.Println(refreshed.Metadata.Values["book"])

    removed, err := api.DeleteMetadata(ctx, name, []string{"note"}, mutationOptions...)
    if removed != nil { fmt.Println(removed.StatusCode) }
    return err
}

func main() {}
```

`GetMetadataOpts`는 `Headers`와 `Newest *bool`, mutation의 `MetadataOpts`는 `Headers`를 갖습니다. Newest nil은 생략, false/true는 HEAD에만 명시적으로 전송합니다. `With…Opts`는 전체 설정을 교체하며 Header/Headers는 ordinary header를 canonical case로 병합하고 마지막 값이 이깁니다. 하나의 plural map이나 최종 callback/source map의 case alias는 거부합니다. helper는 map/pointer를 snapshot하고 매 callback 뒤 복사하여 보관된 callback storage와 옵션 재사용을 분리합니다.

Set의 key와 Delete의 각 key는 prefix 없는 nonempty ASCII HTTP token입니다. 이미 `X-Container-Meta-`가 붙은 key, case alias와 중복 Delete key는 허용하지 않습니다. value는 literal UTF-8 HTTP field value이며 CR/LF/NUL/DEL 및 HTAB 이외의 control은 거부합니다. 공백·Unicode·percent·빈 문자열을 trim, stringify, URL encode/decode하지 않습니다. wire에 prefix를 한 번 붙이며 Delete는 빈 value를 사용합니다. 빈 Set value도 해당 key 삭제입니다. nil/빈 map 또는 key slice는 한 번의 POST이며 제출하지 않은 key는 유지되므로 전체 metadata 삭제를 뜻하지 않습니다.

시스템 metadata는 `WithMetadataHeader`에 정확한 wire 이름을 사용합니다. 예를 들어 `X-Container-Read`, `X-Container-Write`, `X-Container-Sync-To`, `X-Container-Sync-Key`, `X-Versions-Location`, `X-History-Location`, `Content-Type`, `X-Detect-Content-Type`은 ordinary header로 허용하며 명시적 빈 value로 server에 삭제를 요청할 수 있습니다. Python의 `read_ACL`, `sync_key` 등 alias를 변환하지 않습니다. 이 alias를 custom metadata key로 제출하면 custom prefix가 붙습니다. auth·hop-by-hop·body length·Newest·container metadata 관련 header는 SDK 입력이 소유합니다. ordinary Accept/If-*도 허용하고 명시적 native `RetryFunc.MoreHeaders`는 advanced provider 정책으로 남습니다.

[pinned 공식 API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L240)는 POST 후 별도 HEAD로 변경을 확인하고 빈 value 또는 `X-Remove-Container-Meta-`로 한 key를 삭제하는 방법을 설명합니다. 이 API는 빈 value를 사용합니다. 문서의 underscore alias와 value URL encoding 설명은 배포 환경의 경계이며 SDK가 변환하지 않습니다. 실제 [proxy header 전달](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)과 backend 저장에는 metadata URL decode가 없습니다. 일반 HTTP transport/server의 header 정규화까지 입력 snapshot과 동일하다고 보장하지 않습니다.

성공한 Get의 `Metadata.Values`는 소문자 suffix와 literal value의 nonnil map이며 `Temp-URL-Key`와 `Temp-URL-Key-2`도 포함합니다. Python은 알려진 header를 descriptor로 먼저 소비하므로 이 key들이 generic metadata에 없을 수 있습니다. original ASCII header/suffix 검증 뒤 case folding하며, 잘못된 suffix·case alias·중복·0개/여러 value는 atomic projection 실패입니다. 이때 `Metadata`는 nil이고 받은 raw 증거는 남습니다.

`BytesUsed`, `ObjectCount`는 optional signed int64로 missing과 0이 다릅니다. optional sign과 decimal digits를 허용하고 whitespace·fraction·overflow·중복은 거부하며 `strconv.NumError` 원인을 보존합니다. 관측된 음수에 unlimited 의미를 부여하지 않습니다. `Timestamp`와 `LastModified`는 optional literal string으로 present empty와 missing을 구분하며 float/date로 parse하지 않습니다. backend의 [Last-Modified는 PUT timestamp에서 생성](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/container/server.py#L59)됩니다. ACL·version·sync·storage policy·quota·기타 반복 header는 raw Header에만 남습니다. mutation 응답은 metadata로 projection하지 않습니다.

결과는 실제 `Body []byte`, `Header`, `StatusCode`를 독립적으로 소유합니다. 실제 204의 read/Close/context/custom cause 또는 source/projection 실패에서도 결과와 whole-response `resource.ResponseError`를 보존합니다. body는 한 번 닫고 accepted 처리 실패 뒤 재전송하지 않습니다. 예상하지 않은 status와 native/transport 오류는 nil 결과와 원래 typed cause를 유지하고 method/containers 문맥을 추가합니다. acknowledgement는 제출한 값을 metadata로 합성하거나 refresh된 상태를 뜻하지 않습니다.

container 이름은 required literal UTF-8입니다. empty/slash는 native `CheckContainerName` typed cause와 `ErrInvalidOption`을 보존하며 backslash·ASCII control·DEL·정확한 `.`/`..`도 거부합니다. trim하지 않고 Unicode·공백·percent·`?`·`#`는 한 번 escape합니다. 배포별 길이·quota를 추론하거나 `resource.Name` lookup을 수행하지 않습니다.

route는 `ResourceBase`가 있으면 그 값을, 없으면 `Endpoint`를 사용합니다. Endpoint와 effective base 모두 query 없는 HTTP(S)여야 하고 effective base는 Endpoint와 같은 origin이며 trailing slash가 필요합니다. userinfo·fragment·opaque URL은 허용하지 않습니다. escaped reverse prefix를 보존하고 container를 정확히 한 segment로 붙입니다. source/provider와 raw Endpoint/ResourceBase/Type/Microversion/target, ordinary header는 callback 전에 capture하고 호출 경계에서 다시 확인합니다. live token은 원래 provider에서 얻으며 원본 설정은 변경하지 않습니다. 기존 prebody retry/reauth/timeout/transport와 고정 scope의 redirect 정책은 유지하고 method/URL/body/status 소유권 guard를 적용합니다.

metadata 크기·개수·container 길이·ACL·authorization·동기화·저장 정책은 server가 결정합니다. raw 증거가 모든 사용자에게 보이거나 cluster의 durable replication을 보장하지 않습니다. 기존 native Get의 200/204, Update의 201/202/204 acceptance와 native 날짜/float/ACL 변환은 보존됩니다. Python `str(any)`·Resource/cache/session·alias 변환·자동 setter refresh까지 완전한 parity는 아닙니다.
HTTP 계약은 [route와 identity](metadata_contracts_test.go#L99), [preflight와 옵션](metadata_contracts_test.go#L272), [atomic header와 소유권](metadata_contracts_test.go#L454), [mutation 증거](metadata_contracts_test.go#L553), [source와 native 정책](metadata_contracts_test.go#L671) 테스트로 검증합니다.

## Container 생성·삭제

`service.Containers.CreateContainer(ctx, name)`는 body 없는 PUT 한 번으로 container를 생성하거나 기존 container의 제출한 metadata를 갱신합니다. 실제 201과 202만 받아들이며 HEAD, 이름 검색, 자동 refresh를 실행하지 않습니다. `DeleteContainer(ctx, name)`는 body 없는 DELETE 한 번이며 object를 먼저 지우거나 container가 비었는지 조회하지 않습니다. [고정된 Swift API 문서](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L117)는 PUT의 201/202와 빈 container DELETE의 204를 설명합니다. 비어 있지 않으면 409가 날 수 있습니다.

[Python 생성·삭제 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L154)는 다음처럼 사용합니다. 생성은 새 mutable Container를 반환하고 삭제는 literal 이름 또는 Container를 받아 `None`을 반환합니다. 기본 `ignore_missing=True`는 NotFoundException만 무시하며 409는 그대로 오류입니다.

```python
created = conn.object_store.create_container(
    "books-lifecycle", metadata={"owner": "ops", "note": "literal %2F 雪"}
)
# PUT 후 자동 HEAD가 없으며 created는 생성에 사용한 Container입니다.
conn.object_store.delete_container(created, ignore_missing=False)
conn.object_store.delete_container("books-lifecycle")
# 두 번째 DELETE의 404는 기본값으로 무시합니다. object를 자동 삭제하지 않습니다.
```

Go는 literal 이름과 concrete option을 받고 실제 응답 증거를 반환합니다. 아래 예제는 object를 넣지 않은 container를 생성하고 삭제합니다. 두 번째 DELETE는 이미 삭제된 이름의 실제 404를 관찰할 수 있습니다.

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/containers"
)

func containerLifecycle(ctx context.Context, api *containers.API, name string) error {
    createOptions := []containers.CreateContainerOption{
        containers.WithCreateContainerOpts(containers.CreateContainerOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            Metadata: map[string]string{"owner": "initial"},
        }),
        containers.WithCreateContainerHeaders(map[string]string{"X-Trace-Label": "container-create"}),
        containers.WithCreateContainerHeader("X-Request-Label", "create"),
        containers.WithCreateContainerMetadata(map[string]string{
            "owner": "ops", "note": "literal %2F 雪",
        }),
    }
    created, err := api.CreateContainer(ctx, name, createOptions...)
    if created != nil { fmt.Println(created.StatusCode) }
    if err != nil { return err }

    deleteOptions := []containers.DeleteContainerOption{
        containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithDeleteContainerHeaders(map[string]string{"X-Trace-Label": "container-delete"}),
        containers.WithDeleteContainerHeader("X-Request-Label", "delete"),
        containers.WithDeleteContainerIgnoreMissing(false),
    }
    deleted, err := api.DeleteContainer(ctx, name, deleteOptions...)
    if deleted != nil { fmt.Println(deleted.StatusCode, deleted.IgnoredMissing) }
    if err != nil { return err }

    quietOptions := append([]containers.DeleteContainerOption(nil), deleteOptions...)
    quietOptions = append(quietOptions, containers.WithDeleteContainerIgnoreMissing(true))
    missing, err := api.DeleteContainer(ctx, name, quietOptions...)
    if missing != nil { fmt.Println(missing.StatusCode, missing.IgnoredMissing) }
    return err
}

func main() {}
```

`CreateContainerOpts`는 `Headers`와 `Metadata` map, `DeleteContainerOpts`는 `Headers`와 `IgnoreMissing *bool`을 갖습니다. nil/빈 create 설정도 한 번의 PUT입니다. FullOpts helper는 전체 설정을 교체하고 Metadata helper는 metadata 전체 map을 교체합니다. nil/빈 metadata map은 제출할 key가 없다는 뜻입니다. Header/Headers는 canonical case로 병합하여 마지막 값이 이기며 한 map 또는 source/callback map 안의 case alias는 거부합니다. factory는 map/pointer를 snapshot하고 각 callback을 한 번 호출한 뒤 config를 다시 복사합니다. callback은 초기화된 Header map을 받으며 create callback의 Metadata map도 초기화되어 있습니다.

Metadata key는 prefix 없는 nonempty ASCII HTTP token이며 이미 `X-Container-Meta-`가 붙은 key나 case alias는 허용하지 않습니다. value는 literal UTF-8 HTTP field value입니다. HTAB을 허용하지만 나머지 control과 DEL은 거부하며 stringify, trim 또는 URL decode를 하지 않습니다. prefix를 한 번 붙여 PUT하고 [server의 timestamp merge](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/db.py#L1015)가 제출하지 않은 기존 key를 유지합니다. 더 새 timestamp의 빈 value는 해당 key의 삭제 tombstone이며 모든 metadata 삭제나 object metadata의 전체 교체를 뜻하지 않습니다.

ACL·sync·storage policy·version/history location·TempURL·If-None-Match 등 system 입력은 alias가 아닌 정확한 wire header를 `WithCreateContainerHeader` 또는 Headers에 넣습니다. custom/remove-container prefix·auth·framing·Host·X-Newest는 ordinary header에서 예약되어 있습니다. header의 원래 ASCII를 검증한 뒤 canonical case로 바꾸며 source, caller map은 변경하지 않습니다. ACL·quota·이름 길이·metadata 크기·storage policy·권한은 server가 결정합니다. 이미 존재하는 container의 storage policy 변경은 409가 될 수 있으며 조건부 생성이나 durable replication을 보장하지 않습니다.

Delete의 IgnoreMissing nil은 true입니다. 실제로 받은 허용 404의 body read, Close, context와 source 검사가 모두 성공한 경우에만 `StatusCode=404`, `IgnoredMissing=true`, nil error를 반환합니다. 이 응답은 container가 과거에 존재했다거나 cluster 전체에서 없다는 증명이 아닙니다. 허용 404 처리 실패는 raw 결과와 whole-response `resource.ResponseError`를 보존하고 flag는 false입니다. explicit false는 204만 허용하므로 404에서 nil 결과와 native unexpected-response 오류를 반환합니다. transport/nested 오류의 404나 401·403·409·202, read/Close/context/source 실패는 missing으로 무시하지 않습니다. 허용 404는 native error RetryFunc를 거치지 않는 명시적인 Go 차이입니다.

`ContainerResponse`의 `Body []byte`, `Header`, `StatusCode`는 실제 허용 응답에서 독립적으로 소유합니다. 허용 201/202/204/404 처리 중 오류가 나도 받은 raw 증거와 원인·custom context cause를 보존하며 body는 한 번 닫습니다. acknowledgement는 제출한 metadata를 합성하거나 refresh된 상태·date/model을 뜻하지 않습니다. 예상하지 않은 status/native/transport 오류는 nil 결과와 원래 typed 원인을 유지합니다. accepted-response 처리 실패 뒤 재전송하지 않습니다.

이름은 required literal UTF-8입니다. empty/slash는 native name 원인과 `ErrInvalidOption`을 보존하고 backslash, 정확한 `.`/`..`, ASCII control과 DEL도 거부합니다. 공백·Unicode·percent·`?`·`#`·colon은 trim하지 않고 한 번 escape합니다. `resource.Ref`, Name lookup, account HEAD를 사용하지 않습니다. Endpoint/effective ResourceBase의 같은 origin, query 없는 HTTP(S)와 trailing slash를 검증하고 escaped reverse prefix를 보존합니다. callback 전에 source/provider/route/ordinary header를 capture하며 호출과 accepted-response 경계에서 다시 확인합니다. 원래 provider의 live token, configured native prebody retry/reauth/timeout 및 같은 fixed scope redirect 정책은 유지합니다. advanced RetryFunc header 변경은 기존 provider 정책이며 새 header-hook 거부를 약속하지 않습니다.

기존 generated/native `Create`의 201/202/204, `Delete`의 202/204 acceptance와 typed header/date 변환, `Resources` Name 조회, metadata API는 유지됩니다. Python은 [Resource create 반환](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519)·descriptor/dirty alias·`str(any)`·service cache/session을 사용하고 generic `<400` 응답을 허용하므로 이 두 메서드는 bounded partial 비교입니다. 필요한 변경 후 metadata 조회는 별도로 `GetMetadata`를 호출합니다.

검증 근거는 [core wire·응답 처리](lifecycle_core_test.go#L80), [option 소유권·재사용](lifecycle_options_test.go#L16), [외부 API 계약](lifecycle_contracts_test.go#L49), [Connection 공유 client](../../../connection_objectstorage_container_lifecycle_test.go#L18), [native·generator 보존](../../../internal/cmd/sdkgen/swift_container_lifecycle_test.go#L11)에 있습니다.

Python `get_container_metadata(name)`의 HEAD는 Go `GetMetadata`의 typed metadata와 독립 raw `Header`로 매핑합니다. literal 이름을 사용하며 parent HEAD나 name lookup을 추가하지 않습니다. Python의 system header descriptor는 raw `Header`에서 확인하고, Go는 실제204와 atomic header validation을 적용합니다. mutable Resource/cache/session과 setter refresh는 별도 구현 범위입니다.
