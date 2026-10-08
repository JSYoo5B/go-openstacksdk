# Temp URL key 설정과 선택

Swift Temp URL key는 별도 endpoint가 아닌 account/container metadata입니다. `service.Accounts.SetTempURLKey(ctx, key)`와 `service.Containers.SetTempURLKey(ctx, container, key)`는 body 없는 POST 한 번으로 실제 204 acknowledgement를 반환합니다. primary가 기본이며 `WithSetTempURLKeySecondary(true)`로 secondary를 고릅니다. HEAD refresh는 자동 실행하지 않습니다. 갱신 상태가 필요하면 기존 `GetMetadata` 또는 `service.GetTempURLKey`를 명시적으로 호출합니다.

[고정된 Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L923)는 같은 세 작업을 다음처럼 제공합니다. setter는 `None`을 반환하며 [기본 metadata refresh](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)에 따라 POST 다음 HEAD를 수행합니다. getter도 metadata HEAD를 새로 요청하며 기존 Container를 받으면 그 mutable Resource를 사용합니다.

```python
conn.object_store.set_account_temp_url_key(account_secret)
conn.object_store.set_container_temp_url_key(
    "books", container_secret, secondary=True
)
key = conn.object_store.get_temp_url_key("books")
# container secondary → primary, 없으면 account secondary → primary.
# 조회 오류는 fallback하지 않고 그대로 전파합니다. key는 출력하지 않습니다.
```

Go의 getter는 concrete option의 `Container`가 비어 있으면 account HEAD 한 번만 실행합니다. nonempty container이면 container HEAD를 먼저 수행하고 nonempty secondary, nonempty primary 순서로 선택합니다. 둘 다 usable하지 않을 때만 account HEAD로 넘어가 같은 순서로 선택합니다. 두 phase 모두 실제 204를 요구합니다. 어느 단계의 HTTP·transport·read·Close·context·source·metadata projection 오류도 fallback으로 숨기지 않습니다.

```go
package main

import (
    "context"
    "fmt"

    objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func tempURLKeys(ctx context.Context, service *objectstorage.Service, name, accountSecret, containerSecret string) error {
    accountOptions := []accounts.SetTempURLKeyOption{
        accounts.WithSetTempURLKeyOpts(accounts.SetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "account-key"}),
        accounts.WithSetTempURLKeyHeader("X-Request-Label", "account"),
        accounts.WithSetTempURLKeySecondary(false),
    }
    accountAck, err := service.Accounts.SetTempURLKey(ctx, accountSecret, accountOptions...)
    if accountAck != nil { fmt.Println(accountAck.StatusCode) }
    if err != nil { return err }

    containerOptions := []containers.SetTempURLKeyOption{
        containers.WithSetTempURLKeyOpts(containers.SetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "container-key"}),
        containers.WithSetTempURLKeyHeader("X-Request-Label", "container"),
        containers.WithSetTempURLKeySecondary(true),
    }
    containerAck, err := service.Containers.SetTempURLKey(ctx, name, containerSecret, containerOptions...)
    if containerAck != nil { fmt.Println(containerAck.StatusCode) }
    if err != nil { return err }

    getOptions := []objectstorage.GetTempURLKeyOption{
        objectstorage.WithGetTempURLKeyOpts(objectstorage.GetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        objectstorage.WithGetTempURLKeyContainer(name),
        objectstorage.WithGetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "key-discovery"}),
        objectstorage.WithGetTempURLKeyHeader("X-Request-Label", "read"),
        objectstorage.WithGetTempURLKeyNewest(true),
        objectstorage.WithoutGetTempURLKeyNewest(),
        objectstorage.WithGetTempURLKeyNewest(false),
    }
    selected, err := service.GetTempURLKey(ctx, getOptions...)
    if selected != nil {
        if selected.Container != nil { fmt.Println(selected.Container.StatusCode) }
        if selected.Account != nil { fmt.Println(selected.Account.StatusCode) }
    }
    if err != nil { return err }
    fmt.Println(selected.Key != nil, selected.FromContainer, selected.Secondary)
    return nil
}

func main() {}
```

setter의 key는 literal UTF-8 HTTP field value입니다. 공백·빈 문자열·HTAB을 보존하고 coercion, trim, URL decode를 하지 않으며 다른 control과 DEL은 거부합니다. 빈 문자열도 선택한 metadata header를 전송하여 해당 key를 제거합니다. 다른 primary/secondary key를 함께 제거하거나 key를 자동 생성하지 않습니다. Python은 metadata 값을 `str(any)`로 바꾸지만 Go 입력은 `string`입니다. SDK는 입력을 trim하지 않지만 native HTTP transport의 직렬화·파싱은 field value 양끝의 HTTP 공백(OWS)을 정규화합니다.

`SetTempURLKeyOpts`의 `Secondary bool`은 기본 false, `GetTempURLKeyOpts`의 `Container string`은 기본 empty, `Newest *bool`은 기본 nil입니다. `WithGetTempURLKeyContainer("")`는 account-only로 재설정합니다. Newest nil은 header를 생략하고 false/true는 수행하는 모든 HEAD phase에 명시적으로 전송하며 Without helper로 지웁니다. FullOpts는 전체 설정을 교체합니다. Header/Headers는 canonical case로 병합하여 마지막 값이 이기고 한 map 안의 case alias는 거부합니다. factory는 map과 pointer를 snapshot하고, 호출마다 초기화된 Headers와 각 callback 뒤 config를 독립적으로 복사합니다. callback은 한 번씩 실행합니다.

getter의 ordinary header는 account/container의 custom/remove-metadata prefix와 auth·framing·Host·X-Newest를 함께 예약합니다. Newest는 전용 helper로 지정합니다. Setter는 각 기존 metadata API의 header 정책을 유지합니다. 원래 ASCII header name과 literal UTF-8 value를 검증한 뒤 case를 바꾸며 source와 caller map을 수정하지 않습니다. 명시적으로 설정한 native RetryFunc의 advanced header 변경은 기존 provider 정책입니다.

`TempURLKeyResult`는 `Key []byte`, `FromContainer`, `Secondary`와 각 phase의 `Container *containers.GetMetadataResult`, `Account *accounts.GetMetadataResult`를 갖습니다. 첫 실제 허용 응답 전의 preflight/native unexpected 오류는 nil 결과입니다. container의 raw 증거를 받은 뒤 account 요청이 실패하면 앞선 Container 증거를 보존합니다. 실제 허용 응답 처리 실패도 해당 Body/Header/StatusCode와 원인·custom context Cause를 유지합니다. 원래 source 변경 오류는 현재 phase의 typed Metadata를 비우고 실제 응답의 `resource.ResponseError`로 보존합니다. 어떤 오류에서도 Key와 선택 flag를 확정하지 않습니다.

성공한 selected 문자열은 독립적인 UTF-8 byte slice로 복사합니다. usable key가 없으면 Key는 nil이며 flag도 false입니다. raw phase의 `Metadata.Values`는 key 없음과 명시적인 빈 value를 그대로 구분합니다. Python의 type annotation은 이 차이를 정규화하지 않아 마지막 account primary가 명시적으로 비면 empty string이 반환될 수 있습니다. Go는 두 경우 모두 nil Key로 표현하고 raw phase를 남깁니다.

각 HEAD는 기존 전체 `MetadataInfo`의 atomic projection을 사용합니다. 따라서 key와 무관한 recognized counter의 형식 오류, 중복/잘못된 metadata header도 전체 typed Metadata 오류이며 다음 phase로 넘어가지 않습니다. lower-case suffix의 `Values`에는 Temp URL key도 들어갑니다. Python은 이 known header를 descriptor로 옮기므로 residual custom metadata map과 일치한다고 가정하지 않습니다. 응답 body는 한 번 닫고 accepted-response 처리 실패 뒤 재전송하지 않습니다.

getter는 callback 전에 원래 Service/client/provider, public Accounts/Containers/Objects/Swauth API identity, Endpoint/ResourceBase/Type/Microversion과 source header snapshot을 capture하고 phase 전후 및 fallback 전에 다시 확인합니다. 실제 요청은 그 header snapshot과 option overlay를 쓰며 원래 provider의 live auth는 유지합니다. account는 Endpoint를 사용하고 unused ResourceBase URL로 route하지 않지만 raw identity 변경은 검사합니다. nonempty container는 기존 안전한 literal 이름 규칙과 같은 origin의 query 없는 trailing-slash effective ResourceBase를 검증하여 한 번 escape합니다. lookup, Ref, trim, 사전 percent decode, middleware/capability 조회는 없습니다. configured native prebody retry/reauth/timeout 및 fixed scope redirect 정책을 유지합니다.

기존 generated/native metadata API와 `Objects.CreateTempURL`의 signing 동작은 유지됩니다. native signing의 자동 discovery는 primary만 사용하므로 이 secondary-aware getter와 선택 순서가 다릅니다. [고정 Swift middleware](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py)는 account/container key를 사용할 수 있지만 metadata POST 성공이 middleware 활성화, signed URL의 권한·기간·사용 가능성이나 key의 cluster 전체 persistence를 보장하지 않습니다. 이 workflow는 key 설정과 선택을 비교한 partial 구현입니다.

요청·옵션·응답 소유권은 [account core](accounts/temp_url_key_core_test.go#L15)와 [options](accounts/temp_url_key_options_test.go#L16), [container core](containers/temp_url_key_core_test.go#L16)와 [options](containers/temp_url_key_options_test.go#L16), [getter core](temp_url_key_core_test.go#L58)와 [options](temp_url_key_options_test.go#L16), [외부 계약 테스트](temp_url_key_contracts_test.go#L111)에서 검증합니다. [Connection 테스트](../../connection_objectstorage_temp_url_key_test.go#L18)는 공유 client와 각 workflow의 독립 snapshot을, [생성기 테스트](../../internal/cmd/sdkgen/swift_temp_url_key_test.go#L11)는 기존 native API와 registry 보존을 확인합니다.
