# Glance 메타데이터 네임스페이스

`MetadefNamespaces`는 namespace 이름을 직접 지정하는 SDK 소유 leaf API입니다. `conn.ImageV2(ctx).MetadefNamespaces`, `image.Service.API.MetadefNamespaces`, `metadefnamespaces.New(client)`에서 같은 API를 사용합니다. namespace는 UUID가 아니라 `OS::Example` 같은 이름이며, 별도 Find나 ID·Name resolver를 사용하지 않습니다.

| Go 메서드 | 고정 Python proxy | 요청·정상 응답 |
| --- | --- | --- |
| `Create` | `create_metadef_namespace` | `POST metadefs/namespaces`, 201 |
| `Get` | `get_metadef_namespace` | `GET metadefs/namespaces/{namespace}`, 200 |
| `Update` | `update_metadef_namespace` | `PUT metadefs/namespaces/{current}`, 200 |
| `Delete` | `delete_metadef_namespace` | `DELETE metadefs/namespaces/{namespace}`, 204 |
| `List` | `metadef_namespaces` | lazy page 순회, 실제 200 |
| `All` | 같은 generator의 수집 | 전체 수집, 실제 200 |

## 생성·조회·목록·교체·삭제

다음 예제는 호출자가 선택한 namespace를 생성하고 조회한 뒤 목록을 읽고, scalar 설정을 교체하고 삭제합니다. `replacementNamespace`는 호출자가 정한 새 이름 또는 같은 이름입니다. 각 요청은 독립적이며 중간 실패에서는 이미 받은 결과를 반환합니다. `Update`는 생략한 설정을 보존하지 않으므로 아래처럼 유지할 값을 명시해야 합니다.

```go
package example

import (
    "context"

    "gophercloudsdk/image"
    "gophercloudsdk/image/v2/metadefnamespaces"
)

type NamespaceEvidence struct {
    Created *metadefnamespaces.Namespace
    Read *metadefnamespaces.Namespace
    Rows []*metadefnamespaces.Namespace
    All []*metadefnamespaces.Namespace
    Updated *metadefnamespaces.Namespace
    Deleted *metadefnamespaces.Acknowledgement
}

func manageNamespace(ctx context.Context, service *image.Service, namespace, replacementNamespace, owner, resourceType string, pageLimit int) (out NamespaceEvidence, err error) {
    api := service.API.MetadefNamespaces
    description := "First line\nSecond line"
    protected := false
    out.Created, err = api.Create(ctx, namespace,
        metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{
            Description: &description, Protected: &protected,
        }),
        metadefnamespaces.WithCreateHeader("X-Request-Source", "example"),
        metadefnamespaces.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        metadefnamespaces.WithCreateDisplayName("Example"),
        metadefnamespaces.WithCreateDescription(description),
        metadefnamespaces.WithCreateVisibility("private"),
        metadefnamespaces.WithCreateOwner(owner),
        metadefnamespaces.WithCreateProtected(false))
    if err != nil { return out, err }

    out.Read, err = api.Get(ctx, namespace,
        metadefnamespaces.WithGetOpts(metadefnamespaces.GetOpts{}),
        metadefnamespaces.WithGetHeader("X-Request-Source", "example"),
        metadefnamespaces.WithGetHeaders(map[string]string{"X-Trace": "get"}),
        metadefnamespaces.WithGetResourceType(resourceType))
    if err != nil { return out, err }

    listOptions := []metadefnamespaces.ListOption{
        metadefnamespaces.WithListOpts(metadefnamespaces.ListOpts{}),
        metadefnamespaces.WithListHeader("X-Request-Source", "example"),
        metadefnamespaces.WithListHeaders(map[string]string{"X-Trace": "list"}),
        metadefnamespaces.WithListLimit(pageLimit),
        metadefnamespaces.WithListMarker(""),
        metadefnamespaces.WithListVisibility("private"),
        metadefnamespaces.WithListSortKey("created_at"),
        metadefnamespaces.WithListSortDir("desc"),
        metadefnamespaces.WithListResourceTypes("OS::Glance::Image,OS::Nova::Server"),
        metadefnamespaces.WithListMaxItems(20),
        metadefnamespaces.WithListSinglePage(false),
    }
    for row, listErr := range api.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = api.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = api.Update(ctx, namespace,
        metadefnamespaces.WithUpdateOpts(metadefnamespaces.UpdateOpts{}),
        metadefnamespaces.WithUpdateHeader("X-Request-Source", "example"),
        metadefnamespaces.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        metadefnamespaces.WithUpdateNamespace(replacementNamespace),
        metadefnamespaces.WithUpdateDisplayName("Example"),
        metadefnamespaces.WithUpdateDescription(description),
        metadefnamespaces.WithUpdateVisibility("private"),
        metadefnamespaces.WithUpdateOwner(owner),
        metadefnamespaces.WithUpdateProtected(false))
    if err != nil { return out, err }

    out.Deleted, err = api.Delete(ctx, replacementNamespace,
        metadefnamespaces.WithDeleteOpts(metadefnamespaces.DeleteOpts{}),
        metadefnamespaces.WithDeleteHeader("X-Request-Source", "example"),
        metadefnamespaces.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        metadefnamespaces.WithDeleteIgnoreMissing(false))
    return out, err
}
```

`pageLimit=0`은 명시적인 `limit=0`을 보냅니다. limit을 생략하려면 `WithListLimit`을 사용하지 않고 `ListOpts.Limit`을 nil로 둡니다. 빈 목록은 다음 page를 합성하지 않고 끝납니다. 위 예제의 `List`와 `All`은 같은 목록을 각각 새로 요청하며 cache를 공유하지 않습니다. `Delete` 예제는 strict missing을 선택했습니다.

고정 openstacksdk의 대응 호출은 다음과 같습니다.

```python
def manage_namespace(conn, namespace, replacement_namespace, owner, page_limit):
    created = conn.image.create_metadef_namespace(
        namespace=namespace, display_name="Example",
        description="First line\nSecond line", visibility="private",
        owner=owner, is_protected=False)
    fetched = conn.image.get_metadef_namespace(namespace)
    rows = list(conn.image.metadef_namespaces(
        limit=page_limit, visibility="private", sort_key="created_at",
        sort_dir="desc", resource_types="OS::Glance::Image,OS::Nova::Server"))
    updated = conn.image.update_metadef_namespace(
        namespace, namespace=replacement_namespace, display_name="Example",
        description="First line\nSecond line", visibility="private",
        owner=owner, is_protected=False)
    conn.image.delete_metadef_namespace(replacement_namespace, ignore_missing=False)
    return created, fetched, rows, updated
```

Python의 [5개 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1390-L1485)는 namespace 이름 또는 mutable `MetadefNamespace` Resource를 받고 create/get/update는 Resource, delete는 None, list는 generator를 반환합니다. 같은 소스는 namespace 자체가 identity라는 이유로 Find를 제공하지 않는다고 명시합니다. Python의 [모델](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py)은 `is_protected`를 wire `protected`에 대응시키며 resource type association을 nested descriptor로 처리합니다. Go는 canonical `protected` bool pointer와 raw nested 응답을 제공합니다.

## Scalar 입력과 replacement update

namespace는 nonempty valid UTF-8, 최대 80 rune입니다. colon·Unicode·space는 그대로 유지하고 path component를 한 번 escape합니다. 정확한 `.`·`..`, slash·backslash·percent·query·fragment 문자와 ASCII control/DEL은 거부합니다. route 이름과 필수 create 입력은 callback 전에, optional rename은 callback 적용 뒤 HTTP 전에 검사합니다. 자동 trim·case 변경·UUID 해석을 하지 않습니다.

`CreateOpts`의 DisplayName·Description·Visibility·Owner는 optional string pointer, Protected는 optional bool pointer입니다. nil은 생략하고 explicit empty·false는 그대로 전송합니다. Visibility는 public/private만 허용합니다. DisplayName은 최대 80, Description은 500, Owner는 255 rune의 valid UTF-8이며 Description은 newline을 포함할 수 있습니다. 기본 create는 namespace만 보내고 private·false·인증 context owner 등의 기본값을 서버에 맡깁니다. nested properties·objects·tags·resource_type_associations 입력은 이 API에서 제공하지 않습니다.

`Update(ctx, current, ...)`는 선택한 current 이름에 한 번 PUT합니다. body의 namespace는 기본 current이며 `WithUpdateNamespace`로 rename을 지정할 수 있습니다. [공식 replacement 경고](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#update-namespace)와 고정 [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L342-L387)에 따라, 생략한 display_name·description은 null로, visibility·protected·owner는 서버 기본값으로 돌아갑니다. 기존 nested 정의는 이 scalar PUT으로 갱신하지 않습니다. SDK가 현재 설정을 GET하거나 merge하지 않으며 자동 unprotect·rollback도 수행하지 않습니다.

Python proxy도 생략한 namespace를 body에 추가하지만, [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)은 캐시된 Resource의 dirty 값이 없으면 HTTP 없이 같은 객체를 반환할 수 있습니다. Go의 scalar `Update`는 항상 replacement PUT을 보냅니다. Python의 mutable Resource·dirty cache·객체 overload 전체를 같은 계약으로 취급하지 않습니다.

`GetOpts.ResourceType`이 nil이면 query를 생략하고 explicit empty string도 그대로 `resource_type=`로 보냅니다. valid UTF-8/control-free literal 값만 허용합니다. property prefix 적용은 [서버 show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L261-L340)가 수행하며 SDK는 응답을 로컬에서 재작성하지 않습니다.

## 목록과 reverse prefix

기본 ListOpts는 모든 query를 생략하고 `MaxItems=0`으로 cap 없이, `SinglePage=false`로 광고된 다음 page를 읽습니다. `Limit *int`는 nil이면 생략, 0이면 wire 0, 음수이면 오류입니다. Glance의 [limit 검사](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L552-L603)는 0을 허용하고 생략값에는 서버의 configured default/clamp를 적용합니다. Go가 기본 page 크기를 정하거나 cap을 limit hint로 바꾸지 않습니다.

Marker empty는 생략하고 nonempty는 namespace identity 규칙을 검사합니다. Visibility empty는 생략하며 public/private, SortDir empty는 생략하며 asc/desc를 허용합니다. SortKey는 literal UTF-8/control-free 문자열로 보내고 DB whitelist를 추측하지 않습니다. ResourceTypes는 CSV 문자열 그대로 전송합니다. 서버가 comma로 나누며 SDK가 split·trim·정렬·로컬 filter를 하지 않습니다. wire query는 limit·marker·visibility·sort_key·sort_dir·resource_types 6개만 제공합니다.

`List`는 iterator 생성 시 HTTP를 실행하지 않습니다. iteration마다 독립적으로 callback·source·header·query를 준비합니다. required nonnull `namespaces` array 안의 행은 소비할 때만 decode합니다. `MaxItems>0` 또는 caller break는 나머지 행과 next를 검사하기 전에 멈추고, SinglePage는 첫 page의 소비 뒤 next 검사만 생략합니다. 성공한 빈 page는 종료합니다. 성공한 빈 `All`은 nonnil empty slice이며 오류에서는 부분 slice를 반환하지 않습니다. iterator로 이미 소비한 행은 caller에게 남습니다.

continuation은 canonical body `next`만 사용하며 absent/null/empty이면 종료합니다. [서버 serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L647-L661)가 만드는 정확한 root-relative `/v2/metadefs/namespaces`는 캡처한 서비스 collection path에 대응시켜 reverse prefix를 유지합니다. 원래 Body의 next 값은 바꾸지 않습니다. 같은 origin·collection의 absolute/relative 경로도 검증하며 캡처 당시와 정확히 같은 encoded prefix는 유지합니다. 초기 query key/value는 유지하고 singleton nonempty marker만 진행할 수 있습니다. foreign origin·authority·userinfo·fragment·opaque URL·다른 path·다른 escaped alias·dot segment·query drift·반복 URL/marker를 다음 HTTP 전에 거부합니다. first·schema·self·links·HTTP Link는 따라가지 않고 marker fallback을 합성하지 않습니다.

## Canonical 모델·옵션·오류 증거

`Namespace`의 Namespace·DisplayName·Description·Visibility·Owner·Self·Schema는 optional string pointer, IsProtected는 canonical `protected` bool pointer입니다. missing/null은 nil, explicit empty/false는 present이며 잘못된 nonnull type은 오류입니다. 전체 응답은 valid UTF-8의 nonnull object이고 canonical key는 정확한 대소문자만 읽습니다. CreatedAt·UpdatedAt은 nullable literal string이며 시간 parsing이나 ID·Name·Links 합성을 하지 않습니다. nested 정의·links·unknown 값은 shape를 제한하지 않고 `Metadata.Body`의 독립적인 raw JSON으로 보존합니다. 큰 숫자의 precision도 그대로 남깁니다. 응답 namespace·self·schema는 이후 요청을 retarget하지 않습니다.

5개 concrete 옵션 family의 `With…Opts`는 전체 설정을 교체하고 개별 helper는 해당 값이나 일반 header를 추가합니다. 입력 map·pointer·option slice와 callback config를 독립적으로 복사하고 callback은 요청 또는 iteration마다 한 번 실행합니다. nil/error callback·보호된 header·invalid query/input은 HTTP 전에 오류입니다. 임의 body map·builder·query extension·microversion override는 제공하지 않습니다.

실제 Create201/Get·List·Update200만 decode합니다. accepted read·Close·UTF-8·JSON·model·context·source 또는 continuation 오류는 typed 결과를 반환하지 않고 `resource.ResponseError`에 전체 실제 Body·Header·StatusCode와 원인을 보존합니다. Delete의 실제 204는 opaque `Acknowledgement`에 선택한 Namespace와 실제 증거를 남기며, accepted handling 실패에서도 acknowledgement와 오류를 함께 반환합니다.

Delete의 `IgnoreMissing *bool`은 nil이면 true입니다. true는 SDK가 직접 소유한 physical 404의 read·Close·context가 모두 성공했을 때만 nil/nil로 끝내고, 그 404에서만 native RetryFunc를 우회합니다. false는 native 오류·retry 정책을 유지합니다. 서버가 권한 밖 namespace를 404로 숨길 수 있으므로 조용한 404는 실제 부재의 증거가 아닙니다. protected namespace 삭제 정책도 서버가 판단합니다.

source client/provider/type/endpoint/base/microversion/일반 header를 callback 전에 캡처하고 callback 뒤·accepted body 뒤·소비하는 행과 page 전에 안정성과 context를 검사합니다. 원래 provider의 live auth를 사용합니다. configured native pre-body retry·reauth·backoff·동일한 target redirect 정책을 유지하고 method·origin·path·query 또는 owned body를 바꾸는 callback은 차단합니다. body는 한 번 닫고 accepted 실패 뒤 replay하지 않습니다. 원래 read·Close·transport 원인과 `ctx.Err()`·custom `context.Cause`를 보존합니다. native reauth wrapper는 기존 정책을 유지하며 `ErrOriginal`·`ErrReauth` 필드에서 원인을 확인합니다.

비교 기준은 openstacksdk commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, Glance commit `57f7dd9e76ef24e1e9013eceaa703bd442469a24`, native Gophercloud `v2.15.0`입니다. native에는 namespace CRUD/list API가 없어 이 leaf를 SDK에서 소유합니다. Python의 넓은 query/Body filter·coercion·cache/session·nested POST·tag/child resource API와 전체 parity를 주장하지 않으며 discovery gate·Wait·일반 Resources API는 제공하지 않습니다.

실제 wire·raw 모델·advertised paging·옵션·오류 경계는 [외부 HTTP 테스트](contracts_test.go), [core 증거 테스트](core_test.go), [옵션 소유권 테스트](options_test.go)에서 확인합니다. [Connection 테스트](../../../connection_image_metadef_namespaces_test.go)는 shared client와 기본 paging을, [생성기 테스트](../../../internal/cmd/sdkgen/glance_metadef_namespaces_test.go)는 SDK 소유 registry·literal identity·concrete 옵션과 기존 native API의 유지를 검증합니다.
