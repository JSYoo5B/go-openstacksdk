# 이미지 캐시 조회·관리

`image.Service`의 7개 캐시 메서드는 선택한 Glance endpoint에 body/query 없는 고정 요청을 보냅니다. 캐시 상태·node 목록·prune 개수는 실제 200 데이터를 읽고, queue·delete·clear·clean은 실제 응답의 acknowledgement를 반환합니다. 캐시 활성화·권한·driver·이미지 상태는 서버가 판단합니다.

| Go 메서드 | Python proxy | 요청 | 정상 상태·결과 |
| --- | --- | --- | --- |
| `GetImageCache` | `get_image_cache` | `GET cache` | 200, cached/queued 목록 |
| `QueueImage` | `queue_image` | `PUT cache/{id}` | 202, 접수 acknowledgement |
| `CacheDeleteImage` | `cache_delete_image` | `DELETE cache/{id}` | 204, acknowledgement; 기본 clean 404는 nil/nil |
| `ClearCache` | `clear_cache` | `DELETE cache` | 204, 대상이 기록된 acknowledgement |
| `CachedImageNodes` | `cached_image_nodes` | `GET cache/nodes/{id}` | 200, 유한한 node 문자열 배열 |
| `CleanCache` | `clean_cache` | `POST cache/clean` | 200, opaque acknowledgement |
| `PruneCache` | `prune_cache` | `POST cache/prune` | 200, 실제 files/bytes 개수 |

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

type CacheEvidence struct {
    State *image.ImageCache
    Queued *image.ImageCacheAcknowledgement
    Nodes *image.CachedImageNodesResult
    Deleted *image.ImageCacheAcknowledgement
    Cleared *image.ImageCacheAcknowledgement
    Cleaned *image.ImageCacheAcknowledgement
    Pruned *image.CachePruneResult
}

func inspectAndManageCache(ctx context.Context, service *image.Service, ref resource.Ref, target image.CacheTarget) (out CacheEvidence, err error) {
    options := []image.CacheOption{
        image.WithCacheOpts(image.CacheOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithCacheHeader("X-Cache-Request", "manage"),
        image.WithCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
    }
    out.State, err = service.GetImageCache(ctx, options...)
    if err != nil { return out, err }
    out.Queued, err = service.QueueImage(ctx, ref, options...)
    if err != nil { return out, err }
    out.Nodes, err = service.CachedImageNodes(ctx, ref, options...)
    if err != nil { return out, err }
    out.Deleted, err = service.CacheDeleteImage(ctx, ref,
        image.WithCacheDeleteOpts(image.CacheDeleteOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithCacheDeleteHeader("X-Cache-Request", "delete"),
        image.WithCacheDeleteHeaders(map[string]string{"X-Cache-Trace": "sample"}),
        image.WithCacheDeleteIgnoreMissing(false))
    if err != nil { return out, err }
    out.Cleared, err = service.ClearCache(ctx,
        image.WithClearCacheOpts(image.ClearCacheOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithClearCacheHeader("X-Cache-Request", "clear"),
        image.WithClearCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
        image.WithClearCacheTarget(target))
    if err != nil { return out, err }
    out.Cleaned, err = service.CleanCache(ctx, options...)
    if err != nil { return out, err }
    out.Pruned, err = service.PruneCache(ctx, options...)
    return out, err
}

func cacheTargets() [3]image.CacheTarget {
    return [3]image.CacheTarget{image.CacheBoth, image.CacheOnly, image.QueueOnly}
}
```

이 예제는 호출자가 명시적으로 선택한 7개 독립 작업을 순서대로 수행합니다. 중간 실패에서는 이미 받은 값과 오류를 함께 반환합니다. SDK의 batch·transaction·자동 캐시 관리가 아니며, queue 뒤의 nodes 조회가 새 캐시 완료를 증명하지 않습니다. `CacheDeleteImage` 예제는 strict missing을 선택했습니다. 옵션을 생략하거나 전체 설정에서 `IgnoreMissing`을 nil로 두면 기본 true입니다.

## Python Resource와 반환값

```python
def inspect_and_manage_cache(conn, image_id, target="both"):
    state = conn.image.get_image_cache()
    conn.image.queue_image(image_id)
    nodes = conn.image.cached_image_nodes(image_id)
    conn.image.cache_delete_image(image_id, ignore_missing=False)
    conn.image.clear_cache(target=target)
    conn.image.clean_cache()
    pruned = conn.image.prune_cache()
    return state, nodes, pruned
```

고정 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [7개 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L97-L155)에서 get은 mutable `Cache` Resource를, nodes와 prune은 JSON list/dict를 반환합니다. queue·delete·clear·clean의 proxy 반환값은 None입니다. Go는 이 작업들의 실제 status/header/bytes를 acknowledgement로 보존합니다.

Python의 [Cache/CachedImage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/cache.py#L24-L152)는 nested Resource와 descriptor 변환을 사용하며 nodes/prune의 `typing.cast`는 runtime schema 검사가 아닙니다. Go의 canonical typed decoder·raw ownership·명시적인 Name 편의는 이 mutable Resource·dirty/cache·adapter/session 동작 전체와 같다고 주장하지 않습니다. Python string은 image ID이며 자동 Name search가 아닙니다.

## 기본값·clear 대상·옵션 소유권

`CacheOpts`는 일반 `Headers`만 가집니다. `CacheDeleteOpts`는 여기에 `IgnoreMissing *bool`을 추가하며 nil은 true, 명시적인 false/true는 그대로입니다. `ClearCacheOpts.Target`은 `uint8` enum이고 zero `CacheBoth=0`, `CacheOnly=1`, `QueueOnly=2`만 받습니다. 다른 enum 값은 Name lookup·HTTP 전에 오류입니다.

`WithCacheOpts`·`WithCacheDeleteOpts`·`WithClearCacheOpts`는 전체 concrete 설정을 교체합니다. 각 `Header`·`Headers` helper는 일반 header를 추가하고 `WithCacheDeleteIgnoreMissing(bool)`·`WithClearCacheTarget`은 해당 정책을 선택합니다. helper 입력과 map·optional bool pointer·option slice를 snapshot하고 callback config를 한 번 적용한 뒤 다시 복사합니다. 보관한 map/config가 다음 callback이나 준비된 요청을 바꾸지 않습니다. 보호된 auth·framing·version·representation headers, nil/error callback 및 충돌하는 alias는 HTTP 전에 거부합니다.

`x-image-cache-clear-target`은 7개 메서드 모두 SDK가 소유합니다. source나 옵션의 일반 header로 넣으면 값과 alias에 관계없이 거부합니다. Clear의 기본/명시적인 `CacheBoth`는 이 header를 생략하고 `CacheOnly`·`QueueOnly`만 각각 `cache`·`queue`를 보냅니다. Python의 기본 `target="both"`도 header를 생략합니다.

[공식 API reference](https://docs.openstack.org/api-ref/image/v2/index.html#cache-manage)와 고정 Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`의 [clear controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L219-L240)는 header absent/empty를 전체 대상으로 처리하고 literal `both` header는 받지 않습니다. [관리 문서](https://docs.openstack.org/glance/latest/admin/cache.html#controlling-image-cache-using-v2-api)의 header 목록과 이 경계가 다르므로 Go는 실제 controller와 Python 기본 요청을 따릅니다. controller의 내부 `cache_deleted`·`queue_deleted` 개수는 [204 serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L349-L366)가 반환하지 않으며 SDK도 개수를 만들어 넣지 않습니다.

## canonical cache 데이터와 정확한 숫자

`ImageCache`는 embedded `resource.Metadata`, `CachedImages []*CachedImage`, `QueuedImages []string`을 가집니다. canonical `cached_images`와 `queued_images`는 둘 다 required nonnull array이고 null row/ID는 오류입니다. 명시적인 `[]`는 nonnil empty slice로 구분하며 순서와 반복을 유지합니다.

`CachedImage`의 `ImageID *string`, `Hits`·`Size *int64`, `LastAccessed`·`LastModified *json.Number`는 missing/null이면 nil입니다. 명시적인 empty string·0은 nonnil이고 그 외 canonical type은 엄격히 검사합니다. hits/size와 prune 개수는 정확한 signed int64 JSON integer lexeme만 받으며 string·fraction·exponent·overflow를 coercion하지 않습니다.

epoch field는 실제 JSON number token만 받습니다. quoted number는 `json.Number`의 일반 string coercion과 달리 거부하며 fraction·exponent·lexical precision을 그대로 보존합니다. [공식 cache sample](https://docs.openstack.org/api-ref/image/v2/index.html#cache-manage)과 [xattr](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/image_cache/drivers/xattr.py), [sqlite](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/image_cache/drivers/sqlite.py), [centralized DB](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L2306-L2318)는 numeric epoch seconds를 사용합니다. RFC3339/time.Time으로 추정하거나 float64를 거쳐 반올림하지 않습니다.

root/row의 `Body map[string]json.RawMessage`는 canonical·unknown·대소문자 decoy와 arbitrary nested JSON을 독립적으로 소유합니다. typed row/map·root bytes·actual Header도 서로 독립입니다. `Metadata.Links`·`CreatedAt`·`UpdatedAt`는 nil로 남으며 links/date/ID/status/next는 raw 데이터입니다. 이 데이터가 route·pager·capability·추가 요청을 만들지 않습니다. canonical decoder 실패는 receiver를 부분 갱신하지 않습니다.

`CachedImageNodesResult`는 고정 request `ImageID`, `Nodes []string`, 실제 전체 `Body []byte`·`Header`·`StatusCode`를 가집니다. 200의 bare nonnull string array만 받으며 `[]`는 nonnil입니다. empty/foreign/literal URL 문자열도 passive 값으로 보존하고 URL validation·following·host fanout·pagination을 하지 않습니다. 서버 [nodes controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L189-L217)는 centralized_db driver·이미지 존재·정책을 검사합니다.

`CachePruneResult`는 embedded Metadata와 required nonnull `TotalFilesPruned`·`TotalBytesPruned int64`입니다. 실제 200 JSON의 명시적인 0과 unknown raw 데이터를 유지하며 capacity·완료·다른 node의 상태를 추론하지 않습니다.

## acknowledgement·missing·오류 증거

`ImageCacheAcknowledgement.ImageID`는 per-ID queue/delete만 고정 request ID를 가집니다. `Target *CacheTarget`은 Clear에서 기본 `CacheBoth`까지 기록하고 다른 작업에서는 nil입니다. `Body []byte`·`Header`·`StatusCode`는 실제 opaque 응답이며 empty/non-JSON/invalid UTF-8 bytes도 해석하지 않습니다. [queue controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L252-L276)는 registry 존재와 active 상태를 검사하고 worker에 제출합니다. 202는 캐시 저장 완료 증거가 아닙니다. clean은 실제 200 acknowledgement이고 prune만 실제 개수 JSON입니다.

accepted acknowledgement의 read·Close·context 실패에는 nonnil result와 `resource.ResponseError`를 함께 반환합니다. Get/Nodes/Prune의 accepted read·Close·context·UTF-8·JSON·canonical schema 실패에는 nil typed result와 실제 200 raw bytes/header/status를 가진 ResponseError가 남습니다. 원래 read/Close·`ctx.Err()`·custom cause를 유지하고 body는 한 번 닫으며 accepted body 뒤에는 replay하지 않습니다. 다른 actual status·pre-body transport/hook 오류는 기존 native 오류로 반환합니다.

`CacheDeleteImage`만 기본 missing policy를 가집니다. SDK가 직접 소유한 DELETE 404 body를 끝까지 읽고 Close/context까지 모두 성공했을 때만 `(nil, nil)`입니다. read·Close·cancellation·custom cause·transport·hook·reauth·Name resolve 오류와 logical missing은 숨기지 않습니다. accepted 404 body 실패는 nil acknowledgement와 404 ResponseError 증거를 반환합니다. source의 native RetryFunc가 404를 처리하도록 설정되어 있어도 이 private 404 acceptance는 delete에서만 해당 retry를 우회합니다. `IgnoreMissing=false`는 strict 204/native 404 정책입니다. disabled cache와 missing image는 ordinary 404만으로 구분할 수 없어 이 좁은 기본 policy에서 함께 quiet일 수 있습니다. 다른 메서드의 missing은 오류입니다.

## 고정 source와 서버 소유 경계

source/context/client/provider·prefix·일반 header·Ref를 callback 전에 검증·capture합니다. ID는 선택한 endpoint로 직접 사용하고 명시적인 `resource.Name`만 기존 native all-pages exact-name resolver를 거칩니다. Name의 typed model·body ownership·continuation은 기존 경계이며 실패하면 cache 요청을 보내지 않습니다. callback 및 Name lookup 뒤 context·원래 provider identity·SDK 소유 header를 다시 확인합니다. 준비한 route/header를 유지하면서 원래 provider의 live auth token을 사용합니다.

공통 `DoJSON`은 configured native pre-body retry·reauth·backoff 및 같은 target의 redirect 정책을 유지합니다. method·origin·path·query 변경은 transport 전에 차단합니다. RetryFunc가 nil body를 JSON null로 바꾸거나 RawBody·KeepResponseBody·JSONResponse 등 SDK 소유 필드를 바꾸면 다음 요청 전에 원래 오류와 hook/encoding cause를 보존해 거부합니다. native OkCodes가 확장되어도 위의 실제 success status만 받습니다. native reauth 원인들은 기존 `ErrOriginal`·`ErrReauth` 필드에 남습니다.

[공식 API reference](https://docs.openstack.org/api-ref/image/v2/index.html#cache-manage)는 기본 cache API를 v2.14, clean/prune을 v2.18부터 설명합니다. 고정 Python `Cache._max_microversion`은 `2.15`입니다. Go는 source의 version/header 정책을 유지하고 Python cap·자동 discovery/minimum-version gate를 추가하지 않습니다. cache enablement·node-local deployment·policy·driver·worker 실행은 서버 소유이며 SDK는 cache tracking·active prefetch·wait·자동 cleanup·재시도 후 상태 추정·다른 node 작업을 추가하지 않습니다. local 계약과 컴파일 검증은 실제 cloud의 권한·운영 정책이나 full Python parity를 증명하지 않습니다.

local 계약은 [외부 HTTP tests](cache_contracts_test.go), [core tests](cache_core_test.go), [option tests](cache_options_test.go)에 명시합니다. [Connection test](../connection_image_cache_test.go)는 공유 client·기본값·passive 측정을, [generator test](../internal/cmd/sdkgen/glance_cache_test.go)는 기존 native binding 보존과 수동 cache 문서를 확인합니다.
