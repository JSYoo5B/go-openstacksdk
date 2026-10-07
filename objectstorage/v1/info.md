# Swift capability 조회와 segment 크기 선택

`service.GetInfo(ctx, options...)`는 catalog endpoint에서 `/info` 경로를 유도하여 fresh GET을 수행합니다. 성공은 실제 HTTP 200이며 사전 HEAD·목록·SDK 캐시 조회를 하지 않습니다. `GetObjectSegmentSize`도 호출마다 이 GET을 수행하여 요청 크기, 광고된 `swift.max_file_size`, `slo.min_segment_size`를 비교합니다.

`Info`의 `Swift`, `SLO`, `BulkDelete`, `StaticWeb`, `TempURL`은 `map[string]json.RawMessage`입니다. section이 없거나 null이면 nil, `{}`이면 non-nil empty map입니다. root JSON object의 모든 필드는 `Info.Body`에 보존하므로 알려지지 않은 plugin과 큰 정수도 float64로 바꾸지 않습니다. 각 section과 raw field·header는 독립적으로 복사됩니다. `Info.UnmarshalJSON`은 같은 atomic parser를 사용하며 inherited `CreatedAt`·`UpdatedAt`·`Links`는 해석하거나 따라가지 않고 nil로 둡니다. 중복 JSON key는 encoding/json의 마지막 값 정책을 따릅니다. root가 유효한 UTF-8 JSON object가 아니거나 존재하는 non-null canonical section이 object가 아니면 전체 typed projection을 실패시킵니다.

다음 예제는 구성된 Service를 받아 두 작업과 concrete options를 사용합니다. full options가 먼저 설정한 Size를 Without helper로 해제하여 기본 1 GiB를 선택하고, 다음 호출에서는 8 GiB를 명시합니다. HTTP 오류나 응답 처리 오류의 실제 증거도 확인할 수 있습니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    objectstorage "github.com/JSYoo5B/gophercloudsdk/objectstorage/v1"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func reportResponseError(err error) {
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Printf("failed status=%d transaction=%q observed-bytes=%d\n",
            response.StatusCode, response.Header.Get("X-Trans-Id"), len(response.Body))
    }
}

func showSegment(result *objectstorage.ObjectSegmentSizeResult) {
    if result == nil { return }
    fmt.Printf("requested=%d max=%d min=%d selected=%d fallback=%t status=%d raw-bytes=%d\n",
        result.RequestedSize, result.MaxFileSize, result.MinSegmentSize,
        result.Size, result.UsedFallback, result.StatusCode, len(result.Body))
}

func capabilities(ctx context.Context, service *objectstorage.Service) error {
    info, err := service.GetInfo(ctx,
        objectstorage.WithGetInfoOpts(objectstorage.GetInfoOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        objectstorage.WithGetInfoHeaders(map[string]string{"X-Trace-Label": "capabilities"}),
        objectstorage.WithGetInfoHeader("X-Request-Label", "info"),
    )
    if err != nil {
        reportResponseError(err)
        return err
    }
    fmt.Printf("info status=%d transaction=%q\n", info.StatusCode, info.Header.Get("X-Trans-Id"))
    if info.Swift != nil {
        if maximum, ok := info.Swift["max_file_size"]; ok {
            fmt.Println("advertised max_file_size:", string(maximum))
        }
    }
    fmt.Println("SLO section present:", info.SLO != nil)
    if plugin, ok := info.Body["my_plugin"]; ok {
        fmt.Println("raw plugin:", string(plugin))
    }

    large := int64(8 * 1024 * 1024 * 1024)
    defaultOptions := []objectstorage.ObjectSegmentSizeOption{
        objectstorage.WithObjectSegmentSizeOpts(objectstorage.ObjectSegmentSizeOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            Size: &large,
        }),
        objectstorage.WithObjectSegmentSizeHeaders(map[string]string{"X-Trace-Label": "segment"}),
        objectstorage.WithObjectSegmentSizeHeader("X-Request-Label", "default"),
        objectstorage.WithObjectSegmentSize(2 * 1024 * 1024 * 1024),
        objectstorage.WithoutObjectSegmentSize(),
    }
    selected, err := service.GetObjectSegmentSize(ctx, defaultOptions...)
    showSegment(selected)
    if err != nil {
        reportResponseError(err)
        return err
    }

    selected, err = service.GetObjectSegmentSize(ctx,
        objectstorage.WithObjectSegmentSize(large),
    )
    showSegment(selected)
    if err != nil { reportResponseError(err) }
    return err
}

func main() {}
```

`ObjectSegmentSizeOpts.Size`의 nil은 요청 기본값 1,073,741,824 bytes입니다. 명시한 0은 유지하고 음수는 HTTP 전에 거부합니다. 성공한 capability에서 bound가 없거나 null이면 0이며, 알려진 section이 없어도 0으로 처리합니다. bound는 non-negative int64 JSON 정수 token만 받습니다. string·fraction·overflow·음수는 actual 200의 raw 증거를 보존한 오류입니다. 최대 비교를 먼저 수행하고, 다음 최소 비교를 수행합니다. 따라서 max보다 크면 max, 아니면 min보다 작으면 min을 선택하며 서로 뒤집힌 bound를 정규화하지 않습니다. `Info` 조회 자체는 bound를 숫자로 변환하지 않습니다.

segment 선택에만 실제 404 또는 412 fallback을 적용합니다. body 읽기·Close·context·원래 source 확인이 모두 성공한 응답에서 max=2,684,354,561, min=0을 사용하여 같은 비교를 수행합니다. `UsedFallback=true`, `Info=nil`이고 `StatusCode`, `Header`, `Body`에는 실제 404/412 증거가 남습니다. `GetInfo`는 404/412를 fallback 성공으로 바꾸지 않으며 기존 native retry 정책을 유지합니다. transport/nested native error에 들어 있는 status, 403, malformed 200 JSON, read·Close·취소·source 변경 오류에는 fallback하지 않습니다. 2,684,354,561은 Python `(5 * 1024 * 1024 * 1024 + 2) / 2`의 정수값이며 Swift의 전체 기본 max와 구분합니다.

full options는 설정을 교체하고 Header/Headers는 canonical name으로 마지막 값을 병합합니다. 한 map 안의 case alias는 오류입니다. map과 Size pointer를 option 생성 시 snapshot하고 각 callback 뒤에도 복사합니다. nil option, 잘못된 header, auth·framing·metadata mutation 등 예약 header와 지원하지 않는 client capability는 HTTP 전에 거부합니다. callbacks는 한 번씩 호출하며 마지막 Size 또는 Without helper가 presence를 결정합니다.

경로는 catalog `Endpoint`의 escaped path를 사용하고 `ResourceBase`는 route에 사용하지 않습니다. reverse-proxy prefix와 literal percent encoding을 유지하며 ASCII version pattern을 치환합니다. 예를 들어 `/reverse/v1.0/AUTH_x`는 `/reverse/info`가 됩니다. pinned Python의 unanchored regex처럼 `/v1beta/AUTH_x`는 `/infobeta/AUTH_x`, `/v1.2.3/AUTH_x`는 `/info.3/AUTH_x`로 변환됩니다. version match가 없으면 path에 `info`를 붙입니다. query·fragment·userinfo·opaque URL은 거부하고 signed admin query를 만들지 않습니다. Python은 Unicode digits를 version으로 보고 URL params/query/fragment를 유지합니다. literal semicolon도 Go에서는 path 일부지만 Python urlparse에서는 params로 분리되므로 `/v1/AUTH_x;opaque`는 Go `/info`, Python `/info;opaque`가 됩니다.

workflow는 callback 전과 후, HTTP 및 body Read/Close 경계에서 원래 Service·client·provider와 API identity, Endpoint·ResourceBase·Type·Microversion을 확인합니다. source/options header를 복사하여 요청하고 원래 provider의 live auth 및 configured native prebody retry·reauth 정책을 유지합니다. 이후 valid ordinary source header 값 변경은 요청 snapshot을 바꾸거나 실패시키지 않으며 malformed/reserved source header는 guard 오류입니다. 응답 처리가 시작된 뒤 read·Close·decode 오류를 이유로 재전송하지 않습니다. 결과와 오류의 raw header/status/body는 독립적입니다. `GetInfo` projection 실패는 nil Info와 `resource.ResponseError`를 반환하며 segment의 실제 허용 응답을 받은 뒤 실패하면 raw 결과와 오류가 함께 올 수 있습니다. segment의 200 JSON/model/bound 실패는 `RequestedSize`와 실제 raw 응답을 보존하고 `Size=0`, `UsedFallback=false`로 반환합니다. 이미 decode된 Info는 유지하며 두 bound는 모두 유효할 때만 함께 확정합니다. client 설정은 병렬 요청 전에 완료합니다.

[고정된 Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L860-L919)는 다음처럼 사용합니다. `get_info()`에는 request argument가 없으며, `get_object_segment_size`에는 `None` 또는 크기를 명시합니다.

```python
import openstack

conn = openstack.connect()
caps = conn.object_store.get_info()
print("Swift capability:", caps.swift)
print("SLO capability:", caps.slo)
print("default selection:", conn.object_store.get_object_segment_size(None))
print("large selection:", conn.object_store.get_object_segment_size(8 * 1024**3))
```

Python은 fresh mutable [Info Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/info.py#L25-L96)를 fetch하고 같은 Resource에 response를 번역합니다. five known descriptors는 누락/null을 None, non-dict non-null을 empty dict로 바꿉니다. `caps.swift`나 `caps.slo`가 None이면 segment 함수의 `.get`에서 AttributeError가 발생합니다. Go는 nullable object를 엄격히 검증하고 누락된 section의 bound를 0으로 처리합니다. Python은 int64 범위·음수 입력을 사전 검증하지 않으며 404/412 fallback max는 float `2684354561.0`입니다. Resource의 unknown-field filtering, dirty state, JSON parse/status 처리와 adapter/session/cache 정책도 이 Go raw 결과와 동일하지 않습니다.

[Swift `/info` API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage_info.inc#L1-L46)는 public capability JSON을 설명합니다. [고정 controller](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/info.py#L55-L108)는 expose_info가 꺼지면 403을 반환하며 public GET은 200입니다. 이 controller는 기본 404/412를 반환하지 않습니다. 선택한 정보는 설정에 따라 숨길 수 있고 signed admin 조회는 별도 key·signature·expiry 계약입니다.

Swift proxy는 [effective constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/server.py#L359-L378)를 등록하므로 `max_file_size`는 설정에 따라 달라지고 [기본값](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/constraints.py#L28-L66)은 5,368,709,122 bytes입니다. [선택적인 SLO middleware](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1893-L1922)는 `min_segment_size=1`을 광고합니다. SLO section이 없다는 사실만으로 기능·권한을 확정할 수 없으며, 선택한 크기는 업로드의 성공이나 전체 large-object 크기를 보장하지 않습니다.

pinned Gophercloud v2.15.0의 objectstorage/v1에는 `/info` 조회 또는 segment 선택 공개 연산이 없습니다. 기존 listing의 `ExtractInfo`는 다른 함수입니다. 이 두 workflow는 SDK 소유의 partial 구현이며 Python Resource·descriptor·adapter·exception·URL coercion까지 완료한 것으로 판정하지 않습니다.

회귀 테스트는 [raw model·선택·응답 소유권](info_core_test.go), [옵션·사전검증](info_options_test.go), [public HTTP·native retry·fallback](info_contracts_test.go)을 다룹니다. [Connection 공유](../../connection_objectstorage_info_test.go)와 [native facade·생성물 보존](../../internal/cmd/sdkgen/swift_info_test.go)도 별도 계약입니다.
