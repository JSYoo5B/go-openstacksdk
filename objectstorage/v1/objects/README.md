# Swift 리소스와 container 범위

container와 object는 이름이 곧 식별자입니다. 공통 Collection은 `resource.ID(name)`과 `resource.Name(name)`을 구분합니다. ID는 바로 HEAD/DELETE를 실행하고, Name은 prefix 목록을 검색한 뒤 정확한 이름을 비교합니다. object 이름의 `/`, 공백, `?`, `#`, `%`는 SDK가 URL에 인코딩하므로 원래 이름을 전달합니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| container metadata 조회 | `conn.object_store.get_container_metadata(name)` | `service.Containers.Resources.Get(ctx, name)` |
| container 목록 | `conn.object_store.containers()` | `service.Containers.Resources.List(ctx)` |
| object 목록 | `conn.object_store.objects(container)` | `scope.List(ctx)` |
| object metadata 조회 | `conn.object_store.get_object_metadata(object, container=container)` | `scope.Get(ctx, key)` |
| object 삭제 | `conn.object_store.delete_object(object, container=container)` | `scope.Delete(ctx, resource.ID(key))` |

```go
// ctx context.Context, service *swiftv1.Service를 사용하는 오류 반환 함수 안에서
scope, err := service.Objects.InContainer(ctx, resource.ID("assets"))
if err != nil { return err }
object, err := scope.Get(ctx, "images/logo ?.png")
if err != nil { return err }
fmt.Println(object.Name, object.Bytes, object.ContentType, object.Metadata)

for object, err := range scope.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(object.Name, object.Hash)
}

header, err := scope.Create(ctx, "docs/readme.txt", objects.CreateOpts{
    Content: strings.NewReader("hello"),
    Metadata: map[string]string{"Owner": "sdk"},
})
if err != nil { return err }
_ = header

download, err := scope.Download(ctx, resource.ID("docs/readme.txt"))
if err != nil { return err }
defer download.Close()
_, err = io.Copy(destination, download)
if err != nil { return err }
```

`swiftv1`은 `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1`, `objects`는 `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects`, `resource`는 `github.com/JSYoo5B/go-openstacksdk/resource`입니다. `fmt`, `strings`, `io`는 표준 라이브러리이며 `destination`은 `io.Writer`입니다. Connection의 `ObjectStorage(ctx)`가 서비스 객체를 제공합니다.

ID로 parent를 지정하면 사전 조회를 하지 않습니다. `resource.Name(...)`은 container를 한 번 조회해 범위를 고정합니다. 목록의 `ContainerResource/ObjectResource`에는 Swift가 목록에 제공한 필드가 들어 있습니다. `Get`은 HEAD 요청으로 `Details`, 사용자 `Metadata`, 전체 `Header`까지 채웁니다. 목록에서 얻은 모델의 `Details/Metadata/Header`는 nil입니다. metadata key의 대소문자는 HTTP 헤더의 canonical 형태를 따릅니다.

delimiter 목록에서 얻은 디렉터리 항목은 `Object.Subdir`에 보관하며 실제 object 조회 응답을 뜻하지 않습니다. 상태 필드가 없으므로 `Wait`와 `WithStatus`는 `ErrUnsupported`입니다. `WaitDeleted`는 HEAD의 404를 삭제 완료로 판단합니다. Delete는 기본적으로 404를 무시하고, 403 등의 오류는 원래 응답 오류를 보존합니다.

범위 객체는 `Update`, `Copy`, `Download`에도 같은 container와 object 참조를 사용합니다. `CopyOpts.Destination`은 `/container/object` 형식으로 지정합니다. 입력 확장은 기존 API의 `objects.With...Header/Query`를 전달합니다. `Create`는 업로드 응답 헤더를 반환하며 서버의 객체를 추가 HEAD로 읽지 않습니다. 특정 object version, SLO/DLO와 temp URL 등의 API 입력은 전체 API에서 사용할 수 있으며, 이 공통 Collection의 조회·삭제 기본값은 현재 object version입니다.

`Objects.Copy`와 `ObjectScope.Copy`는 기존 native COPY API로 제공하며 `CopyOpts.Destination`을 `/container/object`로 지정합니다. pinned Python의 [`copy_object`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L531-L533)는 `NotImplementedError`를 발생시키는 stub이며 [해당 unit test](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/tests/unit/object_store/v1/test_proxy.py#L251-L252)도 이를 확인합니다. unsupported 판정은 이 pinned Python direct workflow에만 적용하며 Go의 Copy API는 사용할 수 있습니다. `upload_object`는 `create_object`의 alias로 별도 작업을 세지 않습니다.

## 목록의 읽기 범위

Python `conn.object_store.objects(container, max_items=20)`의 읽을 행 수는 Go에서 `resource.WithMaxItems(20)`로 지정합니다. pinned Python Swift proxy는 `paginated=True`를 명시하므로 Go의 `resource.WithPaginated(false)`는 그 proxy에 대한 첫 페이지 확장입니다. 두 옵션은 공통 `scope.List/All`에 사용하며 기존 공개 `service.Objects.List(ctx, container, objects.WithListOptions(...))`와 typed query 입력은 유지됩니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListObjects(ctx context.Context, conn *sdk.Connection, containerName string) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Objects.InContainer(ctx, resource.ID(containerName))
	if err != nil {
		return err
	}
	for object, err := range scope.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(object.Container, object.Name, object.Subdir, object.Bytes)
	}
	return nil
}
```

cap은 공통 `WithName`의 정확한 로컬 비교보다 먼저 응답 행을 셉니다. delimiter로 얻은 `Subdir`도 한 행이므로 실제 object 결과가 cap보다 적을 수 있습니다. scope의 container와 opaque object 이름 처리는 유지하며 목록에서 HEAD를 추가하지 않습니다. `WithMaxItems(0)`은 무제한, 음수는 iterator 순회 시 HTTP 전 오류이며 같은 옵션은 마지막 값이 적용됩니다. cap만으로 wire `limit`을 만들지 않고 명시한 `WithPageSize`만 요청에 사용합니다.

`break`·context 취소는 추가 요청을 중단합니다. 현재 페이지 전체 `IsEmpty/Extract` decode 오류는 cap 이후 행에도 적용됩니다. native marker continuation과 반복 링크 검사를 유지하며, cap이나 첫 페이지가 완료되면 다음 marker 처리 전에 멈춥니다. [공통 목록 가이드](../../../docs/listing.md), [container 목록](../containers/README.md), [native 범위 HTTP 계약](../../../api/native_scope_list_controls_test.go)을 참고합니다.

## Fresh object metadata 읽기와 변경

`service.Objects.GetMetadata/SetMetadata/DeleteMetadata`는 literal container와 object 이름을 받습니다. 기존 `Get/Update`, `ObjectScope`와 native header 모델도 계속 사용할 수 있습니다. 새 조회는 query 없는 HEAD의 실제 200을 읽으며, 변경은 먼저 `?symlink=get` HEAD 200으로 object 자체를 읽고 custom metadata 전체를 병합하거나 지정 suffix를 뺀 뒤 query 없는 POST 202를 보냅니다. object payload를 GET하지 않고 마지막 HEAD도 자동으로 보내지 않습니다.

| 작업 | pinned openstacksdk | Go |
|---|---|---|
| 조회 | `get_object_metadata(obj, container=...)` → mutable `Object` | `GetMetadata(ctx, container, object, ...)` → actual HEAD와 atomic `MetadataInfo` |
| 설정 | cached `Object`를 병합하고 조건부 HEAD 후 POST, 같은 객체 반환 | 항상 fresh HEAD → 병합 POST, `Before`와 `Acknowledgement` 반환 |
| 삭제 | cached metadata가 비면 HEAD, 실제 삭제가 있으면 POST와 마지막 HEAD, proxy는 `None` 반환 | 항상 fresh HEAD → suffix 제외 POST, 추가 refresh 없이 두 phase 증거 반환 |

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func ChangeObjectMetadata(ctx context.Context, conn *sdk.Connection, newest bool) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	container, key := "assets", "docs/readme ?.txt"
	getOptions := []objects.GetMetadataOption{
		objects.WithGetMetadataOpts(objects.GetMetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-read"},
		}),
		objects.WithGetMetadataHeader("X-Example", "read"),
		objects.WithGetMetadataHeaders(map[string]string{"X-Example-Trace": "metadata"}),
	}
	if newest {
		getOptions = append(getOptions, objects.WithGetMetadataNewest(true))
	} else {
		getOptions = append(getOptions, objects.WithoutGetMetadataNewest())
	}
	before, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		if before != nil {
			fmt.Println("HEAD", before.StatusCode)
		}
		return err
	}
	if before.Metadata.ContentLength != nil {
		fmt.Println("object bytes", *before.Metadata.ContentLength)
	}
	mutationOptions := []objects.MetadataOption{
		objects.WithMetadataOpts(objects.MetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-write"},
		}),
		objects.WithMetadataHeader("Content-Disposition", "attachment"),
		objects.WithMetadataHeaders(map[string]string{"Cache-Control": "private"}),
	}
	change, err := service.Objects.SetMetadata(ctx, container, key,
		map[string]string{"owner": "sdk", "empty": ""}, mutationOptions...)
	printObjectMetadataPhases(change)
	if err != nil {
		return err
	}
	change, err = service.Objects.DeleteMetadata(ctx, container, key,
		[]string{"owner"}, mutationOptions...)
	printObjectMetadataPhases(change)
	if err != nil {
		return err
	}
	refreshed, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		return err
	}
	fmt.Println("stored empty value", refreshed.Metadata.Values["empty"])
	return nil
}

func printObjectMetadataPhases(change *objects.MetadataChangeResult) {
	if change == nil {
		return
	}
	if change.Before != nil {
		fmt.Println("before HEAD", change.Before.StatusCode)
	}
	if change.Acknowledgement != nil {
		fmt.Println("POST", change.Acknowledgement.StatusCode)
	}
}
```

Getter의 `Newest` nil은 X-Newest를 생략하며 `WithGetMetadataNewest(false/true)`는 명시한 값을 보냅니다. `WithoutGetMetadataNewest()`는 값을 nil로 되돌립니다. 이 getter 옵션은 mutation HEAD에 적용되지 않습니다. FullOpts는 설정 전체를 교체하고 Header/Headers는 canonical 이름으로 병합합니다. 한 map의 대소문자 alias는 오류이며 singular helper는 마지막 값을 덮어씁니다. Factory는 map·pointer를 복사하고 각 callback 후에도 저장 공간을 분리합니다. Callback은 초기화된 Headers를 받습니다.

Go Set은 빈 string도 실제 metadata 값으로 저장합니다. nil/빈 map, nil/빈 삭제 slice, 없는 suffix 삭제도 HEAD와 POST를 수행합니다. 삭제는 fresh 전체 custom map에서 key를 빼서 다시 전송하며 HTTP DELETE나 X-Remove-Object-Meta를 사용하지 않습니다. 입력은 `owner` 같은 suffix-only ASCII HTTP token이고 full prefix·case alias·중복 삭제 key를 거부합니다. 값은 literal UTF-8 HTTP field value이며 HTAB은 허용합니다. trim, URL decode/encode, Python `str(any)` 변환이나 system alias mapping을 하지 않습니다.

Pinned Python은 다음과 같이 비교할 수 있습니다. 이 코드는 metadata 변경만 사용하며 object contents를 전송하지 않습니다.

```python
obj = conn.object_store.get_object_metadata(
    "docs/readme ?.txt", container="assets"
)
same = conn.object_store.set_object_metadata(obj, owner="sdk", empty="")
assert same is obj
# pinned Object.set_metadata는 empty=""를 무시합니다.
conn.object_store.delete_object_metadata(obj, keys=["owner"])
refreshed = conn.object_store.get_object_metadata(obj)
print(refreshed.metadata)
```

Object의 setter는 `refresh=True` 인자를 받지만 해당 override에서 사용하지 않습니다. `last_modified_at`이 없을 때만 사전 HEAD를 하고 cached metadata와 truthy system descriptor를 병합한 뒤 POST하여 같은 Object를 반환합니다. Delete는 빈 keys면 즉시 끝내고 cached metadata가 비면 HEAD합니다. 삭제할 항목이 없으면 POST를 생략하며, 실제 삭제 POST 후에는 HEAD로 객체를 갱신합니다. Go의 항상 fresh read, literal empty, 항상 변경 요청과 phase 결과는 의도적인 차이입니다. [Pinned Object 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L222)을 참고합니다.

Swift POST는 이전 custom metadata를 교체합니다. 새 API는 관찰한 custom map 전체와 다음 9개 mutable wire header만 보존합니다: Content-Type, Content-Encoding, Content-Disposition, X-Delete-At, X-Object-Manifest, Cache-Control, Content-Language, Expires, X-Robots-Tag. POST의 우선순위는 관찰한 값 → captured source ordinary headers → mutation option Headers입니다. Caller가 명시한 빈 system header도 그대로 전달하며 서버가 의미를 결정합니다. ETag·Content-Length·Date·Last-Modified·X-Timestamp·X-Static-Large-Object·알 수 없는 sysmeta나 서버 설정으로 추가한 allowed header는 자동 재전송하지 않습니다. 정확한 system wire 이름을 Headers로 전달하고 Python `content_type` 같은 alias를 custom suffix에 넣지 않습니다. [Pinned POST와 HEAD API 설명](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L631), [backend replacement 구현](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/obj/server.py#L665)에 따른 범위입니다.

`Before`는 변경 직전 HEAD의 Body/Header/Status이며 변경된 상태를 뜻하지 않습니다. Getter의 `Metadata`는 전체 header projection이 성공한 후에만 들어갑니다. Values는 lowercase suffix의 single literal value로 빈 값도 유지합니다. ContentLength는 exact signed int64 pointer로 missing과 0을 구분하고, ETag·content fields·Timestamp·LastModified·DeleteAt·manifest·cache fields는 nullable literal string입니다. 날짜·float·checksum을 파싱하지 않습니다. 원래 header 이름을 ASCII로 검증한 뒤 case fold하며, known/custom header의 중복·여러 값·잘못된 counter는 actual HEAD 증거를 유지하고 projection 전체를 실패시킵니다. Unknown header는 raw Header에 남습니다.

HEAD가 받아들여진 뒤 read/Close/context/source/projection 오류가 나면 raw 결과와 `ResponseError`를 함께 반환합니다. Mutation의 POST 이전 또는 native POST 실패에는 완료된 `Before`를 보존하고 Acknowledgement는 nil입니다. 실제 POST 202를 받은 경우에만 Acknowledgement를 만들며, 이후 body/Close/context/source 오류에도 opaque 실제 bytes·header·status를 오류와 함께 남깁니다. 응답을 입력으로 합성하지 않고 accepted body를 다시 요청하지 않습니다. Missing 404 등 unexpected 응답은 원래 native cause와 HTTP 증거를 보존하며 missing을 무시하지 않습니다.

이름은 literal container와 opaque object입니다. Parent 이름 조회나 parent HEAD는 하지 않습니다. Object의 `/`를 포함한 전체 이름을 한 번 escape하므로 literal `%2F`를 미리 decode하지 않습니다. 공백·Unicode·`%`·`?`·`#`·`:`를 허용하고 empty·잘못된 UTF-8·backslash·ASCII control·DEL을 거부합니다. Container의 slash와 exact `.`/`..`, object의 `.`/`..` path component도 거부합니다. Native empty/slash 원인 타입은 유지합니다. Endpoint/effective ResourceBase는 같은 origin의 query 없는 HTTP(S) base와 trailing slash를 요구하며 encoded reverse prefix를 정규화하지 않습니다.

일반 Getter HEAD는 서버의 symlink 동작을 따릅니다. Mutation은 `?symlink=get`로 own metadata를 읽고 X-Symlink-Target의 존재를 확인하면 POST 전에 `ErrUnsupported`와 HEAD 증거를 반환합니다. 이 제한은 current versioned alias도 포함합니다. Version query, temp URL, SLO와 더 넓은 conditional/native 옵션은 기존 API의 범위입니다. Object가 두 phase 사이 symlink가 되면 POST가 적용된 뒤 307을 반환할 수 있습니다. 오류의 `Before`와 native 응답을 확인해야 하며 실패 응답이 변경 없음의 증거는 아닙니다.

HEAD→merge/subtract→POST는 atomic하지 않습니다. Pinned Swift POST에는 이 흐름을 보호하는 일반적인 If-Match/If-Unmodified-Since CAS가 없어 동시 변경을 덮어쓸 수 있습니다. Whole RMW를 재시작하거나 자동 rollback하지 않습니다. Source·provider·base·target·headers를 capture하고 원래 live auth와 native prebody retry/reauth/backoff/timeout, 같은 fixed target의 configured redirect policy를 유지합니다. Method/URL/body/status는 SDK가 지키지만 고급 `RetryFunc.MoreHeaders` 변경은 기존 provider policy입니다. Shared client는 바꾸지 않으며 동시 사용 전 설정합니다.

실제 HTTP 계약은 [routes와 literal identity](metadata_contracts_test.go#L108), [preflight와 옵션 snapshots](metadata_contracts_test.go#L314), [atomic header projection](metadata_contracts_test.go#L537), [두 phase의 실패 증거](metadata_contracts_test.go#L710), [source와 native policy](metadata_contracts_test.go#L925)에서 확인할 수 있습니다.
