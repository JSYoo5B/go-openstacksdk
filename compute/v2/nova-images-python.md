# Legacy Nova image proxy Python 대응

`compute/v2/images`는 pinned openstacksdk의 deprecated Nova image proxy 8개를 명시적 버전 범위 안에서 제공합니다. Glance의 `image/v2/images`와 경로·응답·메타데이터 계약이 다릅니다. 이 leaf는 자동 `Connection` service binder에 등록하지 않으며, 선택한 Compute client로 `images.New(service.RawClient())`를 구성합니다. 새 이미지 기능에는 Glance 서비스를 사용합니다.

[Nova의 현재 API 문서](https://docs.openstack.org/api-ref/compute/#images-deprecated)는 일반 `/images`를 **2.1–2.35**, `/images/{id}/metadata`를 **2.1–2.38**에서 지원합니다. 이후 각각 2.36/2.39부터 404입니다. [현재 Nova route 등록](https://github.com/openstack/nova/blob/master/nova/api/openstack/compute/routes.py)과 [버전 상수](https://github.com/openstack/nova/blob/master/nova/api/openstack/api_version_request.py)에도 이 경로와 경계가 남아 있습니다. 따라서 이 API를 최신 Nova 모든 버전에서 사용 가능하다고 선언하지 않습니다.

| pinned openstacksdk | Go | 요청·기본값 |
|---|---|---|
| `get_image(image)` | `api.GetRecord(ctx, imageID)` | literal ID로 GET `/images/{id}` |
| `images(details=True, **query)` | `api.ListRecords(ctx, options...)` | 기본 `/images/detail`, false면 `/images` |
| `find_image(name_or_id, ignore_missing=True)` | `api.FindRecord(ctx, nameOrID, options...)` | GET-first 후 `/images?name=...` summary fallback |
| `delete_image(image, ignore_missing=True)` | `api.DeleteRecord(ctx, imageID, options...)` | clean 404만 기본으로 억제 |
| `fetch_image_metadata(image)` | `api.FetchMetadata(ctx, imageID)` | GET metadata, JSON 객체 필요 |
| `get_image_metadata(image)` | `api.GetMetadata(ctx, imageID)` | FetchMetadata와 같은 deprecated alias |
| `set_image_metadata(image, **metadata)` | `api.SetMetadata(ctx, imageID, metadata)` | POST, 제출한 metadata map을 view로 반환 |
| `delete_image_metadata(image, keys=None)` | `api.DeleteMetadata(ctx, imageID, options...)` | 기본 PUT `{"metadata":{}}`; 명시 keys는 항목별 DELETE |

Source `Image`는 `_max_microversion`을 선언하지 않아 inherited `None`입니다. 기본 버전 미선택 시 `maximum_supported_microversion(..., None)`은 discovery 없이 None을 반환하고, 명시한 session default는 그대로 사용합니다. 이 Go leaf는 **선택된 client 버전만** 사용하며 자동 downgrade·discovery를 하지 않습니다. 빈 버전은 Nova의 기본 2.1로 판단합니다. 잘못된 버전 문자열은 `resource.ErrInvalidOption`, 정상 문자열이 지원 범위를 벗어나거나 `latest`이면 `resource.ErrUnsupported`이며 HTTP를 보내지 않습니다. 이 guard는 `DeleteRecord`의 ignore_missing보다 먼저 적용합니다.

```python
# compute_api_version은 clouds.yaml에서 2.35로 설정합니다.
images = conn.compute.images(details=False, min_disk=4, max_items=100)
image = conn.compute.get_image(image_id)
found = conn.compute.find_image(image_id)
metadata = conn.compute.fetch_image_metadata(image_id)
conn.compute.set_image_metadata(image_id, owner="ci")
conn.compute.delete_image_metadata(image_id, keys=["owner"])
```

## 실행 가능한 Go main

[설치 안내](../../docs/install.md)대로 모듈을 준비하고 아래를 `main.go`로 저장합니다. `go run . -cloud dev -id IMAGE_ID`는 legacy summary 목록·GET·Find·metadata 조회를 실행합니다. `-set-owner`·`-delete-keys`·`-clear-metadata`·`-delete-image`를 명시할 때만 해당 변경 요청을 실행합니다. 예제는 모든 경로를 지원하는 2.35를 명시합니다.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/images"
)

func printImage(value *images.ImageRecord) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "Nova legacy image ID")
    owner := flag.String("set-owner", "", "metadata owner to POST")
    keys := flag.String("delete-keys", "", "comma-separated metadata keys")
    clear := flag.Bool("clear-metadata", false, "replace metadata with empty object")
    remove := flag.Bool("delete-image", false, "delete image strictly")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Images
    for value, err := range api.ListRecords(ctx,
        images.WithListDetails(false), images.WithListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printImage(value)
    }
    if *id == "" { return }
    value, err := api.GetRecord(ctx, *id)
    if err != nil { log.Fatal(err) }
    printImage(value)
    value, err = api.FindRecord(ctx, *id, images.WithFindIgnoreMissing(false))
    if err != nil { log.Fatal(err) }
    printImage(value)
    value, err = api.FetchMetadata(ctx, *id)
    if err != nil { log.Fatal(err) }
    printImage(value)
    if *owner != "" {
        value, err = api.SetMetadata(ctx, *id, map[string]any{"owner": *owner})
        if err != nil { log.Fatal(err) }
        printImage(value)
    }
    if *keys != "" {
        if err := api.DeleteMetadata(ctx, *id,
            images.WithDeleteMetadataKeys(strings.Split(*keys, ",")...)); err != nil { log.Fatal(err) }
    }
    if *clear {
        if err := api.DeleteMetadata(ctx, *id); err != nil { log.Fatal(err) }
    }
    if *remove {
        if err := api.DeleteRecord(ctx, *id,
            images.WithDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## 응답·옵션 계약

`ImageRecord.Resource`는 12개 선언 필드 `id/name/links/created_at/min_disk/min_ram/progress/status/updated_at/size/metadata/location`를 가진 view입니다. 누락·null은 null이며 location도 null입니다. `created`, `minDisk`, `minRam`, `updated`, `OS-EXT-IMG-SIZE:size`를 해당 Go view 필드로 투영하고, 응답에 양쪽 alias가 있으면 Python dictionary의 마지막 alias 순서를 따릅니다. 네 integer descriptor는 공통 Source descriptor 변환을 사용합니다. `metadata`는 dictionary descriptor라 null을 보존하고 다른 nonobject 값은 `{}`로 투영합니다. 날짜 문자열·name·links 등 untyped 값은 JSON 타입 그대로 유지합니다.

`Wire`에는 원래 응답 객체의 unknown fields와 정확한 큰 정수, `Envelope`에는 전체 HTTP 본문, `Header`·`StatusCode`에는 실제 응답 증거가 남습니다. 각 view·wire·receipt의 bytes/header는 독립 복사합니다. GET은 누락한 id를 요청 ID로 유지하며, singular `image`가 있으면 unpack하고 없으면 root를 사용합니다. Source `_translate_response`처럼 빈 본문·JSON syntax 오류는 seed만 가진 view로 허용합니다. 유효한 nonobject JSON은 accepted response proof를 포함한 오류입니다. List는 plural `images`를 요구하고 singleton 객체도 지원합니다.

`WithListQuery(key, value)`는 kwargs를 분류합니다. `server/name/status/type/min_disk/min_ram/changes_since`는 wire query이며 서버 spelling `minDisk/minRam/changes-since`도 지원합니다. client spelling과 wire spelling을 함께 주면 client spelling이 우선합니다. Requests JSON-domain encoder를 재사용하므로 null wire 값은 생략하고 반복 array·object keys·`True`/`False` 표현을 따릅니다. 다른 선언된 Body attribute는 local JSON filter이며 미등록 query는 버립니다. `WithListLimit/Marker/MaxItems/Paginated`는 concrete control입니다. max_items는 local filter **이전 raw row**를 세고, 생략·0인 limit의 초기 힌트가 됩니다. List iterator는 lazy이며 반복할 때마다 Source snapshot을 새로 잡습니다.

Find는 Source formal option과 같이 missing 정책만 받습니다. clean GET 400/403/404만 fallback을 허용하고, summary `/images`에 name hint를 추가해 모든 페이지의 정확한 ID/name을 검사합니다. 이름이 중복이면 `resource.ErrAmbiguous`, strict missing이면 `resource.ErrNotFound`입니다. 앞 페이지에서 찾은 뒤 발생한 후속 오류도 반환합니다. **`find_image`에는 details option이 없으며 fallback은 `/images/detail`이 아닙니다.**

Metadata Fetch/Get은 Source `response.json()`과 같이 syntax 오류·빈 본문을 허용하지 않습니다. 이 Go profile은 nonnull JSON 객체를 요구하며, metadata가 없으면 null view를 유지합니다. Set은 호출 시 metadata map을 JSON으로 snapshot하여 option callback이나 caller가 바꿔도 실제 요청·반환 view가 바뀌지 않습니다. POST 응답을 metadata 교체용으로 decode하지 않으며 제출한 map을 반환합니다. nil map은 `{}`입니다. Delete는 기본으로 빈 map을 PUT하고, `WithDeleteMetadataKeys()`의 명시적 empty slice는 요청 0회입니다. 키는 미리 검증·복사하고 중복을 제거한 첫 등장 순서대로 DELETE합니다. 항목 404는 strict이며 첫 오류·취소에서 중단합니다.

모든 HTTP operation은 200..399를 허용합니다. Native transport policy의 redirect/retry 동작을 유지하고, mutation의 opaque body를 JSON decode하지 않습니다. `DeleteRecord`만 clean 404를 기본으로 억제합니다. 본문 read/close·retry callback·취소·Source 변경이 결합된 오류는 억제하지 않으며, `resource.OperationError`와 실제 `resource.ResponseError` 증거를 보존합니다. Source endpoint/resource base/provider/type/microversion과 option 소유권은 요청 전과 재시도 사이에 공통 guard로 검증합니다.

## Go profile 차이와 검증 범위

이 8개는 **호출 가능한 버전 제한 legacy Go profile**입니다. Python mutable Resource 동일 인스턴스·Munch·cached metadata·Connection location·custom Session/subclass hooks·warnings는 별도 Resource/session 범위입니다. 메서드는 유효한 UTF-8 literal ID/키를 받고 한 번 escape합니다. Go의 nonnegative typed controls와 raw JSON projection, preflight version/source/header guards도 명시적 차이입니다. 임의 Python kwargs로 경로·session·microversion을 바꾸는 동작과 JMESPath surface를 제공하지 않습니다.

공통 REST paging은 Source의 array/top-level/HTTP next·positive initial limit short-page marker fallback·empty-page 종료·falsey next·singleton을 재사용합니다. dictionary `links:{next:...}`는 Source generic Resource가 아닌 Go extension입니다. continuation은 같은 origin·고정 collection path와 기존 query를 지켜야 하며, 새로운 임의 `page` filter를 추가하지 못합니다. local JSON filter는 큰 정수·중첩 subset·null을 보존하지만 Python bool/number equality나 nonobject nested-filter 예외를 복제하지 않습니다. Metadata Fetch는 JSON array/scalar에 대한 Python membership 동작 대신 nonnull object를 요구하며, Delete key set의 순서는 deterministic입니다.

[계약 테스트](images/image_records_test.go)는 public Gophercloud `testhelper`를 재사용하는 `internal/testcloud` fixture로 실제 요청 경로·body·query·header·status와 위 차이를 검증합니다. 집중 이름은 `^TestPythonNovaImage`이며 7개 그룹입니다. live OpenStack 결과와 최신 모든 microversion에서의 지원을 주장하지 않습니다. standalone main compile과 전체/race 검증, 중앙 지원 판정·commit은 별도 통합 gate에서 완료합니다.

Source pin: `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`; `openstack/compute/v2/_proxy.py:645-830`, `openstack/compute/v2/image.py`, `openstack/common/metadata.py`, `openstack/resource.py:Resource.list/find/_translate_response/_get_microversion`, `openstack/fields.py:_convert_type`, `openstack/utils.py:maximum_supported_microversion`를 대조했습니다.


## 거부 응답의 읽기·Close 오류와 find fallback

Member GET의 400/403/404와 DeleteRecord의 404는 공통 `DoJSONGuardedRejections`로 관찰합니다. 정상 404만 기본 삭제에서 무시하며, Read·Close·취소 cause·Source 변경·retry callback 오류가 섞인 거부 응답은 native 상태/본문/헤더와 원인을 함께 반환하고 find의 목록 fallback을 차단합니다. Read에서 변한 Source가 Close에서 원복되어도 이미 관측한 실패는 유지됩니다. native retry hook이 같은 원래 HTTP 거부 오류를 그대로 반환한 경우에는 정상 거부 상태를 유지해 기본 ignore_missing과 find fallback이 계속 동작합니다. accepted 응답 증거와 native ABI는 유지합니다.

회귀는 `TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind`와 `TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook`이며 기존 package fault body/transport 및 testcloud를 재사용합니다. 검증 실행은 부모 runner가 담당합니다.
