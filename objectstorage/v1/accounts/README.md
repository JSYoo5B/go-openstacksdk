# Account metadata 조회·설정·삭제

`accounts.API.GetMetadata`는 현재 client의 정확한 `Endpoint`에 body 없는 HEAD를, `SetMetadata`와 `DeleteMetadata`는 body 없는 POST를 보냅니다. 세 메서드는 실제 204만 성공으로 받아들입니다. 별도 account ID/Name, account 생성·삭제, discovery 또는 목록 조회를 수행하지 않습니다. `ResourceBase`는 route를 바꾸지 않습니다.

[pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L119)는 Account Resource를 반환하거나 변경합니다. 공개 setter와 deleter의 반환값은 `None`입니다. setter는 [기본 `refresh=True`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L71)에 따라 POST 뒤 HEAD를 추가하며, deleter는 빈 metadata value를 POST하고 refresh하지 않습니다.

```python
account = conn.object_store.get_account_metadata()
print(account.metadata)
conn.object_store.set_account_metadata(book="Moby Dick", note="literal %2F 雪")
# setter 내부에서 POST 다음 HEAD refresh를 수행합니다. 반환값은 None입니다.
conn.object_store.delete_account_metadata(["note"])
# HTTP DELETE가 아니라 빈 X-Account-Meta-Note 값을 보내는 POST입니다.
```

Go setter는 POST acknowledgement를 반환합니다. 새 metadata가 필요할 때 다음처럼 명시적으로 `GetMetadata`를 호출합니다. 예제의 첫 HEAD는 `X-Newest: false`를 보내고, 두 번째 HEAD는 해당 옵션을 비워 server 기본값을 사용합니다.

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/accounts"
)

func changeAccountMetadata(ctx context.Context, api *accounts.API) error {
    getOptions := []accounts.GetMetadataOption{
        accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        accounts.WithGetMetadataHeader("X-Request-Label", "account-metadata"),
        accounts.WithGetMetadataNewest(false),
    }
    before, err := api.GetMetadata(ctx, getOptions...)
    if err != nil { return err }
    if before.Metadata.BytesUsed != nil { fmt.Println(*before.Metadata.BytesUsed) }

    mutationOptions := []accounts.MetadataOption{
        accounts.WithMetadataOpts(accounts.MetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        accounts.WithMetadataHeader("X-Request-Label", "metadata-mutation"),
    }
    acknowledgement, err := api.SetMetadata(ctx, map[string]string{
        "book": "Moby Dick", "note": "literal %2F 雪",
    }, mutationOptions...)
    if acknowledgement != nil {
        fmt.Println(acknowledgement.StatusCode) // err가 있어도 받은 증거는 남습니다.
    }
    if err != nil { return err }

    refreshOptions := append([]accounts.GetMetadataOption(nil), getOptions...)
    refreshOptions = append(refreshOptions, accounts.WithoutGetMetadataNewest())
    refreshed, err := api.GetMetadata(ctx, refreshOptions...)
    if err != nil { return err }
    fmt.Println(refreshed.Metadata.Values["book"])

    removed, err := api.DeleteMetadata(ctx, []string{"note"}, mutationOptions...)
    if removed != nil { fmt.Println(removed.StatusCode) }
    return err
}

func main() {}
```

`GetMetadataOpts`는 `Headers`와 `Newest *bool`, mutation의 `MetadataOpts`는 `Headers`를 갖습니다. Newest가 nil이면 header를 생략하고 false/true는 HEAD에만 명시적으로 전송합니다. `With…Opts`는 전체 설정을 교체하며 Header/Headers는 ordinary header를 canonical case로 병합하고 마지막 값이 이깁니다. 하나의 plural map이나 최종 callback/source map의 case alias는 거부합니다. 모든 helper가 map/pointer를 snapshot하고, 매 callback 뒤 복사하여 callback이 보관한 map이나 재사용되는 옵션이 다른 호출에 영향을 주지 않게 합니다.

Set의 key와 Delete의 각 key는 prefix 없는 nonempty ASCII HTTP token입니다. 이미 `X-Account-Meta-`가 붙은 key, case alias와 중복 Delete key는 허용하지 않습니다. wire에는 prefix를 정확히 한 번 붙입니다. value는 literal UTF-8 HTTP field value이며 CR/LF/NUL/DEL 및 HTAB 이외의 control은 거부합니다. 공백·Unicode·percent·빈 문자열을 trim, stringify, URL encode/decode하지 않습니다. 빈 Set value는 해당 metadata의 삭제이며 nil/빈 map 또는 key slice도 한 번의 POST입니다. 제출하지 않은 key는 server의 기존 상태로 남으므로 빈 입력이 전체 metadata 삭제를 뜻하지 않습니다.

[pinned 공식 metadata 규칙](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-account-services.inc#L120)은 빈 value 또는 `X-Remove-Account-Meta-`로 삭제하는 방법을 설명합니다. 이 API는 빈 value 방식을 사용합니다. 문서의 underscore alias와 value URL encoding 설명은 배포 환경의 경계이며 SDK가 변환하지 않습니다. 실제 [proxy header 전달](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)과 backend metadata 저장에는 metadata URL decode가 없습니다. 일반 HTTP transport/server가 수행하는 header 정규화까지 입력 snapshot과 동일하다고 보장하지 않습니다.

성공한 Get의 `Metadata.Values`는 소문자 suffix와 literal value의 nonnil map입니다. `Temp-URL-Key`와 `Temp-URL-Key-2`도 포함합니다. Python은 알려진 header를 Resource descriptor로 먼저 소비하여 이 key들이 `metadata` map에 없을 수 있으므로 명시적인 Go 차이입니다. canonical metadata header의 잘못된 suffix, 중복 또는 0개/여러 value는 atomic projection 실패입니다.

`BytesUsed`, `ContainerCount`, `ObjectCount`는 optional signed int64이며 missing과 0을 구분합니다. optional sign과 decimal digits를 허용하고 whitespace, fraction, overflow와 중복 값을 거부합니다. 관측된 음수는 그대로 보존하며 특별한 unlimited 의미를 부여하지 않습니다. `Timestamp`는 optional literal string으로 present empty와 missing이 다릅니다. 날짜·quota·transaction·기타 반복 header는 raw Header에 남고 따로 parse하지 않습니다. mutation 응답에서는 header를 metadata로 해석하지 않습니다.

반환 결과는 실제 `Body []byte`, `Header`, `StatusCode`를 독립적으로 소유합니다. Get의 `Metadata`는 전체 projection이 성공했을 때만 제공됩니다. 실제 204의 read/Close/context/custom cause 또는 source/projection 실패에서도 raw 결과와 원인을 합친 `resource.ResponseError`를 보존하며 body는 한 번 닫고 받은 응답 뒤 재전송하지 않습니다. 예상하지 않은 status와 native/transport 오류는 nil 결과와 원래 typed cause를 유지하고 `request.Wrap`의 method/accounts 문맥을 추가합니다. acknowledgement는 제출한 값이나 refresh된 상태를 metadata로 합성하지 않습니다.

source client/provider, Endpoint/Type/Microversion과 ordinary header는 callback 전에 고정하고 호출 경계에서 다시 확인합니다. token은 원래 provider의 live authentication을 사용하며 원본 설정은 변경하지 않습니다. auth·hop-by-hop·body length·Newest·account metadata 관련 header는 SDK가 소유합니다. ordinary Content-Type/Accept/If-*는 허용하며 native의 명시적 `RetryFunc.MoreHeaders`는 advanced provider 정책으로 남습니다. 기존 prebody retry/reauth/timeout/transport와 같은 scope의 redirect 정책은 유지하고 method/URL/body/status 변경은 guard로 막습니다.

server가 metadata 크기·개수 제한, quota, ACL, account autocreate와 cluster 저장 정책을 결정합니다. SDK는 배포 제한을 추론하거나 metadata POST의 durable replication을 보장하지 않습니다. 기존 native/generated `Get`/`Update`와 Connection의 Account API는 보존됩니다. native `Update`의 201/202/204 acceptance, Python `str(any)`·Resource/cache/session·자동 setter refresh까지 완전한 parity는 아닙니다.
HTTP 계약은 [route와 presence](metadata_contracts_test.go#L88), [preflight와 snapshot](metadata_contracts_test.go#L195), [header decode와 소유권](metadata_contracts_test.go#L352), [mutation 증거](metadata_contracts_test.go#L446), [source와 native 정책](metadata_contracts_test.go#L529) 테스트로 검증합니다.

Python `get_account_metadata()`의 HEAD는 Go `GetMetadata`의 typed metadata와 독립 raw `Header`로 매핑합니다. Account 입력이나 query는 없으며, Python이 descriptor로 노출하는 system header도 raw `Header`에서 확인합니다. Python의 HTTP400 미만 status acceptance와 달리 Go는 실제204만 허용합니다. mutable Resource/cache/session과 관련 setter의 refresh는 별도 구현 범위입니다.
