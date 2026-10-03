# Swift object 삭제

`service.Objects.DeleteObject(ctx, container, object, options...)`는 기본적으로 fresh HEAD로 SLO 여부를 확인한 뒤 같은 literal object에 DELETE를 보냅니다. `StaticLargeObject`가 nil이면 HEAD를 수행하고, caller가 알고 있는 false/true를 명시하면 HEAD를 생략합니다. true이면 DELETE에 `multipart-manifest=delete`를 붙여 manifest와 segment 삭제를 요청합니다. 이름 조회·캐시·삭제 대기는 수행하지 않습니다.

`DeleteObjectResult`는 HEAD의 `Discovery`, DELETE의 `Deletion`, nullable `StaticLargeObject`, `IgnoredMissing`, 선택적인 `Bulk`를 구분합니다. 각 phase의 `Body`, `Header`, `StatusCode`는 실제 응답이며 앞선 HEAD 증거는 후속 DELETE 오류에도 남습니다. accepted 응답을 받은 뒤 read·Close·context·source·decode 오류가 발생하면 non-nil 결과와 오류가 함께 올 수 있습니다.

HEAD는 200/204, 일반 DELETE는 202/204를 허용합니다. SLO DELETE는 200도 허용하며 이 경우에만 bulk JSON을 해석합니다. 202/204는 raw acknowledgement이고 `Bulk=nil`, 삭제·미존재 counter는 unknown입니다. HTTP 성공이나 bulk counter가 cluster 전체 cleanup 완료 또는 이후 조회의 즉시 미존재를 보장하지 않습니다.

다음 예제는 full options의 known-SLO·strict-missing·Newest 설정을 Without helper로 해제하여 기본 자동 HEAD와 missing 허용을 사용합니다. 다음 호출은 이미 SLO임을 아는 manifest를 명시적으로 삭제합니다. 두 호출은 각각 독립적인 작업입니다.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "gophercloudsdk/objectstorage/v1/objects"
    "gophercloudsdk/resource"
)

func showDelete(result *objects.DeleteObjectResult) {
    if result == nil { return }
    fmt.Println("ignored missing:", result.IgnoredMissing)
    if result.StaticLargeObject == nil {
        fmt.Println("SLO flag unknown")
    } else {
        fmt.Println("SLO:", *result.StaticLargeObject)
    }
    for _, phase := range []struct {
        name string
        response *objects.DeleteObjectResponse
    }{{"HEAD", result.Discovery}, {"DELETE", result.Deletion}} {
        if phase.response != nil {
            fmt.Printf("%s status=%d transaction=%q observed-bytes=%d\n",
                phase.name, phase.response.StatusCode,
                phase.response.Header.Get("X-Trans-Id"), len(phase.response.Body))
        }
    }
    if result.Bulk != nil {
        fmt.Printf("bulk status=%d deleted=%d not-found=%d errors=%d\n",
            result.Bulk.ResponseCode, result.Bulk.NumberDeleted,
            result.Bulk.NumberNotFound, len(result.Bulk.Errors))
    }
}

func showDeleteError(err error) {
    var response *resource.ResponseError
    if errors.As(err, &response) {
        fmt.Printf("failed physical status=%d observed-bytes=%d\n",
            response.StatusCode, len(response.Body))
    }
    var bulk *objects.ObjectDeleteBulkError
    if errors.As(err, &bulk) {
        fmt.Printf("embedded bulk failure=%q errors=%d\n",
            bulk.ResponseStatus, len(bulk.Errors))
    }
}

func deleteObjects(ctx context.Context, api *objects.API, versionID string) error {
    knownSLO, allowMissing, newest := false, false, true
    auto := []objects.DeleteObjectOption{
        objects.WithDeleteObjectOpts(objects.DeleteObjectOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            StaticLargeObject: &knownSLO, IgnoreMissing: &allowMissing, Newest: &newest,
        }),
        objects.WithDeleteObjectHeaders(map[string]string{"X-Trace-Label": "object-delete"}),
        objects.WithDeleteObjectHeader("X-Request-Label", "auto"),
        objects.WithoutDeleteObjectStaticLargeObject(),
        objects.WithDeleteObjectIgnoreMissing(false),
        objects.WithoutDeleteObjectIgnoreMissing(),
        objects.WithDeleteObjectVersionID(versionID),
        objects.WithDeleteObjectNewest(true),
        objects.WithoutDeleteObjectNewest(),
        objects.WithDeleteObjectNewest(false),
    }
    result, err := api.DeleteObject(ctx, "backups", "report.txt", auto...)
    showDelete(result)
    if err != nil {
        showDeleteError(err)
        return err
    }

    result, err = api.DeleteObject(ctx, "backups", "manifest",
        objects.WithDeleteObjectStaticLargeObject(true),
        objects.WithDeleteObjectIgnoreMissing(false),
    )
    showDelete(result)
    if err != nil { showDeleteError(err) }
    return err
}

func main() {}
```

`IgnoreMissing` nil은 true이고 Without helper는 nil/default로 되돌립니다. body 읽기·Close·context·source 확인을 모두 통과한 실제 HEAD404 또는 DELETE404만 `IgnoredMissing=true`로 처리합니다. HEAD404는 DELETE를 보내지 않고 `StaticLargeObject=nil`을 유지합니다. strict missing은 native 오류이며 앞선 HEAD 결과가 있으면 함께 반환합니다. transport·retry·reauth 오류 안의 404, dirty 404와 bulk의 embedded 404는 무시하지 않습니다. `Bulk.NumberNotFound`는 server가 보고한 manifest/segment 미존재 수이며 top-level missing flag를 결정하지 않습니다.

자동 HEAD의 `X-Static-Large-Object`가 없으면 false입니다. 존재하면 OWS를 제거한 단일 nonempty value가 case-insensitive true/false 또는 1/0이어야 하며 malformed·중복 값은 실제 HEAD 증거를 보존한 오류로 끝납니다. `Newest` nil은 header 생략, false/true는 수행하는 HEAD와 DELETE에 명시한 `X-Newest`를 보냅니다. `VersionID`는 같은 literal `version-id` query를 두 phase에 한 번 encode하여 전달하는 Go 확장입니다. 빈 문자열은 생략하며 UTF-8/control 검증을 적용합니다. 임의 query나 async 옵션은 제공하지 않습니다.

SLO DELETE는 native retry hook 이후에도 `Accept: application/json`을 고정합니다. 실제 200 body는 선행 whitespace·Swift heartbeat 뒤의 non-null UTF-8 JSON object여야 합니다. `ObjectDeleteBulkInfo.UnmarshalJSON`은 `Response Status`, `Response Body` string, non-negative exact int64 JSON integer token인 `Number Deleted`, `Number Not Found`, 두 string으로 이루어진 배열들의 `Errors`를 atomic하게 해석합니다. embedded status는 100부터 599까지의 세 자리 code와 선택적인 nonempty reason이며 `Response Status`의 ASCII·Unicode control을 거부합니다. 성공은 embedded 2xx와 empty Errors를 모두 요구합니다. 다른 outcome은 parsed Bulk와 실제 200 `resource.ResponseError` 안의 `ObjectDeleteBulkError`를 반환합니다. parse 실패는 `Bulk=nil`과 raw DELETE 증거를 유지합니다. raw root field·unknown extension·큰 정수와 literal failure name/message를 보존하며 inherited timestamp/link는 passive입니다. 결과와 오류 증거는 독립적으로 복사합니다.

container/object는 필수 literal UTF-8 이름이며 기존 안전한 object 경로 규칙으로 한 번 escape합니다. effective ResourceBase를 사용하고 `%2F`와 `/`를 구별하며 lookup·trim·사전 percent decode를 하지 않습니다. full options는 전체 설정을 교체하고 Header/Headers는 canonical name으로 마지막 값을 병합합니다. 한 map의 case alias, nil option과 auth·framing·metadata mutation·X-Newest·X-Static-Large-Object 등 예약 header는 사전에 거부합니다. factory의 map·bool pointer와 각 callback 뒤 설정은 snapshot하며 callbacks는 한 번씩 실행합니다.

원래 API·client·provider와 Endpoint·ResourceBase·Type·Microversion·context는 callback, native retry/backoff/reauth/redirect hook, 실제 transport, body Read/Close와 phase 사이에서 확인합니다. valid ordinary source header의 이후 변경은 captured 요청에 영향을 주지 않고 malformed/reserved drift는 실패합니다. 원래 provider의 live auth와 고정 target 안의 native hook 정책을 유지하며 accepted body 처리 오류를 이유로 요청을 재전송하지 않습니다. HEAD와 DELETE 사이의 동시 변경은 막지 못합니다. client 설정은 병렬 요청 전에 완료합니다.

[고정된 Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L535-L561)는 기본 missing 허용과 strict missing을 다음처럼 사용하며 `None`을 반환합니다.

```python
import openstack

conn = openstack.connect()
conn.object_store.delete_object("report.txt", container="backups")
conn.object_store.delete_object("strict-report.txt", container="backups", ignore_missing=False)
```

Python은 기존 Object의 container를 explicit container보다 먼저 사용하고 mutable Object를 다시 사용합니다. shared Resource.delete는 실제 [Object._raw_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L374-L407)를 호출하여 SLO flag가 None일 때 HEAD하고 truthy이면 `multipart-manifest=delete`를 추가합니다. pinned bool descriptor는 `bool(value)`를 사용하므로 nonempty 문자열 `false`도 truthy입니다. Go의 엄격한 header 해석과 explicit boolean은 의도적인 차이입니다. Python의 missing 정책은 전체 delete 경로의 NotFoundException을 잡으므로 probe HEAD의 missing도 포함합니다.

Python Resource.delete는 `has_body=False`로 header만 번역하여 SLO bulk body의 embedded 실패를 검사하지 않습니다. Go는 이 응답을 해석하여 partial failure를 드러냅니다. Python의 dirty header·metadata mutation, Resource/container 우선순위, descriptor coercion과 adapter/session/exception 정책까지 동일한 구현으로 판정하지 않습니다. public Python delete_object에는 query/version-id 인자가 없습니다. 같은 pinned proxy의 `copy_object()`는 명시적으로 NotImplementedError를 발생시키며 구현된 복사 workflow로 사용하지 않습니다.

[Swift object API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L449-L625)는 일반 DELETE204·HEAD200과 eventual consistency, symlink 자체 삭제, SLO segment/manifest 삭제를 설명합니다. Go가 추가로 허용하는 HEAD204·DELETE202는 [기존 native Get](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L455-L480)과 [DELETE default codes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L598)의 호환 정책입니다. [SLO controller](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1801-L1829)와 [bulk outcome](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/bulk.py#L331-L467)은 outer200과 body의 실제 결과를 구분합니다. 기존 native/generated `Objects.Delete`의 `DeleteOpts.MultipartManifest`·`ObjectVersionID`와 `Objects.Copy`는 별도 API로 유지합니다. 이 Go workflow도 전체 Python Resource parity를 완료한 판정은 아닙니다.
