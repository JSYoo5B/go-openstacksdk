# Swift 디렉터리 marker 객체

`service.Objects.CreateDirectoryMarkerObject`는 기존 container의 literal 이름에 zero-byte 객체를 PUT하고 `Content-Type: application/directory`를 설정합니다. Static Web/Web Listings용 marker가 필요한 환경에서 사용합니다. container나 디렉터리 경로를 자동으로 만들지 않고 이름에 `/`를 붙이거나 제거하지 않습니다.

| 정책 | pinned openstacksdk cloud helper | Go |
|---|---|---|
| 입력 | container, name, `**headers` | literal container, name, concrete 함수 옵션 |
| payload | empty string, generate_checksums=False | SDK가 소유한 empty bytes |
| media | lowercase content-type를 설정한 뒤 Resource 변환 | 검증 뒤 owned application/directory 강제 |
| 조회·분할 | data branch의 단일 PUT | 하나의 logical PUT; HEAD/info/hash/segment 없음 |
| 반환 | mutable Object | 기존 `CreateObjectResult`의 실제 Ordinary 증거 |

이미 인증한 Connection과 존재하는 `backups` container를 사용합니다. 아래 `reports/`는 호출자가 선택한 object 이름이며 helper가 만든 suffix가 아닙니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    sdk "gophercloudsdk"
    "gophercloudsdk/objectstorage/v1/objects"
    "gophercloudsdk/resource"
)

func showMarker(result *objects.CreateObjectResult, err error) {
    if result != nil {
        fmt.Printf("source=%s mode=%s size=%d md5=%q sha256=%q\n",
            result.Source, result.Mode, result.Size, result.MD5, result.SHA256)
        if result.Ordinary != nil {
            for _, attempt := range result.Ordinary.Attempts {
                fmt.Printf("logical=%d physical=%d error=%v\n",
                    attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
                if attempt.Response != nil {
                    fmt.Printf("HTTP%d headers=%v body=%q\n",
                        attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
                }
            }
            if result.Ordinary.Acknowledgement != nil {
                fmt.Println("acknowledged HTTP", result.Ordinary.Acknowledgement.StatusCode)
            }
        }
    }
    if err != nil {
        fmt.Println(err)
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
                proof.StatusCode, proof.Header, proof.Body)
        }
    }
}

func markerExample(ctx context.Context, conn *sdk.Connection) {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        showMarker(nil, err)
        return
    }
    options := []objects.DirectoryMarkerOption{
        objects.WithDirectoryMarkerOpts(objects.DirectoryMarkerOpts{
            Headers: map[string]string{"X-Client": "directory-marker"},
            Metadata: map[string]string{"Owner": "sdk"},
        }),
        objects.WithDirectoryMarkerHeader("X-Trace", "initial"),
        objects.WithDirectoryMarkerHeaders(map[string]string{"x-trace": "marker"}),
        objects.WithDirectoryMarkerMetadata(map[string]string{"Purpose": "web-listing"}),
        objects.WithDirectoryMarkerMetadataValue("Build", "2026-10-04"),
    }
    result, err := service.Objects.CreateDirectoryMarkerObject(ctx, "backups", "reports/", options...)
    showMarker(result, err)
}

func main() {}
```

`DirectoryMarkerOpts`는 `Headers`와 `Metadata`만 받습니다. full opts는 설정 전체를 교체하고 factory와 option callback마다 maps를 snapshot합니다. header helpers는 canonical 이름으로 순서대로 overlay합니다. metadata는 `Owner`처럼 suffix만 지정하면 `X-Object-Meta-Owner`로 전송합니다. arbitrary Python kwargs나 Resource descriptor coercion을 그대로 노출하지 않습니다.

caller/source의 유효한 Content-Type은 옵션 처리 뒤 application/directory로 고정합니다. 그 전에 alias 중복·잘못된 이름·UTF8/control·reserved header 입력을 검증합니다. native retry/reauth/redirect와 실제 wire 경계에서는 owned Content-Type의 변경·제거·복수 값을 허용하지 않습니다. `X-Detect-Content-Type`도 옵션·원래 client·native callback·wire 경계에서 거부합니다. pinned Swift는 이 flag가 true이면 명시한 Content-Type을 무시하고 경로에서 media를 추측하므로 marker의 directory media를 유지하기 위한 정책입니다.

SDK는 zero-byte payload를 소유하고 replay마다 독립 cursor와 정확한 framing을 제공합니다. 한 logical PUT 안에서 native retry/backoff/reauth 또는 같은 target redirect가 여러 physical requests를 보낼 수 있습니다. option callbacks는 한 번 준비하며 captured ordinary headers와 literal route를 유지하고 live auth를 사용합니다. source/body/status-code 정책을 바꾸는 callback은 오류가 되며 실제 시작한 요청과 응답 증거는 남습니다.

결과는 `Source="bytes"`, `Mode="ordinary"`, `Size=0`이며 MD5/SHA256은 빈 문자열입니다. `Ordinary.Attempts`는 실제 physical attempts, `Ordinary.Acknowledgement`는 실제201/202 접수 응답입니다. read/Close/context/source 처리 오류가 접수 뒤 발생하면 결과와 오류를 함께 확인해야 합니다. acknowledgement와 attempt/error proof는 header/body 저장 공간을 독립 소유합니다.201/202는 cluster 전체 완료나 Static Web에서 즉시 보인다는 보장이 아닙니다.

이 작업은 container 생성, 사전 HEAD, capability 조회, checksum 생성, stale 비교, SLO/DLO 분할, readback이나 cleanup을 수행하지 않습니다. missing container와 권한·통신·서버 오류는 반환합니다. 기존 ordinary `CreateObject`의 옵션과 media 정책은 별도로 유지합니다.

[pinned cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L231-L258)는 lowercase content-type를 설정하고 `create_object(data='', generate_checksums=False)`에 위임합니다. [cloud create_object](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_object_store.py#L260-L321)와 [canonical proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L398-L526)의 docstring은 container 자동 생성을 말하지만 이 data 경로에는 그런 호출이 없습니다. [Object.create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L359-L372)는 raw data PUT 뒤 mutable Object를 반환합니다.

Python의 case-insensitive [Resource header 변환](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L917-L950)은 입력 순서에 따라 마지막 alias를 선택하고, [BaseResource metadata 변환](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L48-L66)은 기존 header 뒤에 system metadata를 적용합니다. 따라서 helper의 lowercase 설정만으로 모든 alias/metadata 조합의 media가 고정되지는 않습니다. Go의 strict maps·owned media·[Swift detect flag 거부](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/obj.py#L708-L717)는 명시적인 차이입니다. Python Resource/session/cache와 서버 Static Web 동작의 전체 parity가 완료된 판정은 아닙니다.
