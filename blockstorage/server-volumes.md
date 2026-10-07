# 서버에 연결된 볼륨 목록

`conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})`는 Cinder v3의 상세 볼륨 목록을 모두 읽고, 각 attachment의 서버 값이 입력과 일치하는 볼륨을 반환합니다. 서버 ID는 로컬 비교 값입니다. 이름·UUID·경로를 검증하거나 Nova에서 서버를 조회하지 않으며, 빈 문자열·공백·대소문자도 그대로 비교합니다.

이 helper는 `GET /volumes/detail`에서 시작합니다. 이름·상태·서버 필터나 all-projects query를 만들지 않습니다. 목록의 EOF까지 materialize한 뒤 association을 선택하므로, 앞 페이지에 일치 항목이 있어도 실제로 읽는 뒤 페이지 실패를 먼저 반환합니다. Canonical `volumes: []` 페이지는 즉시 EOF이며 그 페이지의 advertised next 링크는 읽지 않습니다. 별도의 per-volume GET, device 조회, refresh나 상태 대기는 없습니다. 시간 제한은 caller context로 설정합니다.

## Go 호출과 결과 처리

다음 함수들은 독립적인 사용 예입니다. `conn`과 client는 [전체 README](../README.md)처럼 준비합니다.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/blockstorage"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func getForServer(ctx context.Context, conn *sdk.Connection, serverID string) error {
    result, err := conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})
    return inspectResult(result, err)
}

func getWithClient(ctx context.Context, cinder *gophercloud.ServiceClient, serverID string) (*blockstorage.GetVolumesResult, error) {
    return blockstorage.GetVolumes(ctx, cinder,
        blockstorage.GetVolumesRequest{ServerID: serverID})
}

func getForRawServer(ctx context.Context, conn *sdk.Connection) (*blockstorage.GetVolumesResult, error) {
    // An explicit null ID can match a null attachment server_id.
    server := map[string]json.RawMessage{"id": json.RawMessage(`null`)}
    return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
        blockstorage.WithGetVolumesServerFields(server))
}

func prepareServerFields(ctx context.Context, server map[string]json.RawMessage) (blockstorage.GetVolumesOpts, error) {
    // Runs options locally once and owns their maps/raw bytes; no HTTP.
    return blockstorage.PrepareGetVolumesOptions(ctx,
        blockstorage.WithGetVolumesServerFields(server))
}

func getWithPreparedOptions(ctx context.Context, conn *sdk.Connection, prepared blockstorage.GetVolumesOpts) (*blockstorage.GetVolumesResult, error) {
    return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
        blockstorage.WithGetVolumesOptions(prepared))
}

func inspectResult(result *blockstorage.GetVolumesResult, callErr error) error {
    if callErr != nil {
        if result != nil {
            // Volumes is nil on every workflow error. Pages is actual response evidence.
            for i, page := range result.Pages {
                fmt.Printf("accepted page %d: status=%d request=%q bytes=%d\n",
                    i, page.StatusCode, page.Header.Get("X-Openstack-Request-Id"), len(page.Body))
            }
        }
        return callErr
    }
    for i, volume := range result.Volumes {
        fmt.Printf("match %d: raw id=%s raw status=%s\n", i,
            volume.Body["id"], volume.Body["status"])

        var projected blockstorage.VolumeInfo
        if err := volume.Decode(&projected); err != nil {
            // Selection succeeded; this optional stricter typed projection failed.
            fmt.Printf("typed projection %d: %v\n", i, err)
        } else {
            fmt.Printf("typed projection %d: status=%d request=%q\n", i,
                projected.StatusCode, projected.Header.Get("X-Openstack-Request-Id"))
        }

        privateCopy := volume.Clone()
        privateCopy.Body["name"] = json.RawMessage(`"private-copy"`)
        fmt.Printf("copy %d name=%s; original name=%s\n", i,
            privateCopy.Body["name"], volume.Body["name"])
    }
    return nil
}

func independentMatches(result *blockstorage.GetVolumesResult) []*resource.RawResource {
    copies := make([]*resource.RawResource, len(result.Volumes))
    for i, volume := range result.Volumes {
        copies[i] = volume.Clone()
    }
    return copies
}
```

`GetVolumesResult.Volumes`는 성공 시 새 nonnil slice입니다. 일치 없음도 nonnil 빈 slice이며, 오류가 하나라도 있으면 nil입니다. `Pages`는 실제 접수한 200 응답의 body·header·status를 보관합니다. 예제의 오류 처리는 페이지 증거를 확인하고 원래 오류를 반환합니다. 페이지 일부를 완성된 볼륨 목록으로 취급하지 않습니다.

## openstacksdk와 비교

실제 대응은 cloud helper `conn.get_volumes(server)`입니다. Python은 전달한 서버의 `server['id']`를 읽으며, 서버와 볼륨 상태를 새로 확인하는 별도 요청을 만들지 않습니다.

```python
server = {"id": "server-id"}
volumes = conn.get_volumes(server)
for volume in volumes:
    print(volume["id"])
```

고정 Python 구현은 `list_volumes()`의 반환 목록을 완성한 뒤 모든 attachment를 비교합니다. `cache` 인자는 deprecated warning만 발생시키며 실제 목록 조회·캐시에 영향을 주지 않습니다. Go에는 이 no-op 인자와 warning을 추가하지 않았습니다.

## raw 결과와 선택적인 typed projection

반환 원소는 `*resource.RawResource`입니다. 전체 UTF-8 JSON object와 원래 필드를 보관하며, id·status·timestamp·device 등 알려진 필드를 일괄적으로 typed VolumeInfo로 decode하지 않습니다. Python helper가 읽지 않는 device나 다른 필드 때문에 association 결과를 잃지 않도록 한 표현입니다. JSON 숫자와 extension 값도 `Body`의 RawMessage로 유지합니다.

필요할 때 `raw.Decode(&typed)`로 명시적인 projection을 수행합니다. target은 nonnil pointer여야 하며, 현재 `Body`를 fresh target에 decode한 후 성공 시에만 caller target에 반영합니다. 실패하면 이전 target을 유지합니다. target의 기존 custom decoder와 알려진 타입 검사는 그대로 적용하므로, raw 선택이 성공했어도 `VolumeInfo` projection은 잘못된 device·timestamp·다른 알려진 필드 때문에 실패할 수 있습니다. 직접 선언한 exported exact `Metadata` 필드가 있는 SDK typed target에는 원래 header·status 등의 owned metadata를 전달합니다. 이 local 변환이 HTTP 오류를 새로 만들지는 않습니다.

`RawResource`를 JSON으로 marshal하면 현재 `Body`를 출력합니다. `Clone()`은 Body·raw bytes·header·nullable metadata를 독립적으로 복사합니다. 같은 볼륨의 여러 attachment가 일치하면 `Volumes`에는 **같은 RawResource pointer**가 여러 번 들어갑니다. 한 entry를 수정하면 같은 pointer의 다른 entry에서도 그 수정이 보입니다. 각 occurrence를 독립적으로 수정하려면 예제의 `independentMatches`처럼 각각 clone합니다. ID가 같더라도 서로 다른 원본 목록 row는 별도 resource입니다.

## association과 서버 필드의 읽는 순서

볼륨 순서와 attachment 순서를 유지하며 **모든** attachment를 방문합니다. 일치할 때마다 볼륨을 추가하고 break나 dedup을 하지 않습니다. 앞 attachment가 일치했어도 뒤 attachment가 malformed이면 전체 호출이 실패하며 최종 `Volumes`는 nil입니다. Device는 누락·null·빈 문자열·false·object여도 읽거나 검사하지 않습니다. available/in-use 같은 상태 조건도 없습니다.

Cinder v3의 실제 Volume Resource는 `attachments`에 `type=list` descriptor를 사용합니다. Array는 그대로 순회하며 nonnull nonarray 값은 하나의 row로 감쌉니다. 따라서 `attachments`가 `{"server_id":"server-id"}` object이면 한 attachment로 비교할 수 있고, 빈 object나 string은 object/key 조건에서 실패합니다. 누락 또는 null attachments도 오류입니다. 이는 [순수 device accessor](volume-attachment-device.md)의 raw plain-mapping 빈 iterable 처리와 입력 모델이 다릅니다.

각 row는 object여야 하며 정확한 `server_id` key를 먼저 읽습니다. 그다음 제공한 서버의 정확한 `id`를 읽습니다. 같은 row에서 attachment `server_id`와 서버 `id`가 모두 없으면 attachment 오류가 먼저입니다. 볼륨이 없거나 모든 attachment array가 비어 있으면 서버 필드를 읽지 않습니다.

기본 `GetVolumesRequest.ServerID`는 JSON string으로 비교합니다. `WithGetVolumesServerFields`는 실제 JSON 서버 object의 `id`를 표현할 때 사용합니다. `GetVolumesOpts.ServerFields == nil`은 기본 문자열 입력이고, nonnil pointer가 nil map을 가리키면 명시적인 서버 필드 부재입니다. Nonnil map에 `id`가 있고 그 RawMessage가 nil이면 명시적인 JSON null입니다. 잘못된 JSON·UTF-8 또는 missing `id`는 실제 row가 서버 값을 읽을 때만 local 입력 오류가 됩니다. 옵션 preparation이나 목록 조회 전에 미리 거부하지 않습니다.

JSON string·null·bool·number·array·object를 비교하며 object key 순서는 비교에 영향을 주지 않습니다. Python의 `True == 1`, `False == 0`은 nested array/object 값에서도 적용합니다. 숫자는 float64로 바꾸지 않고 exact decimal 값으로 비교합니다. 이 규칙은 이 workflow의 Python식 비교에만 사용하며 기존 collection filter의 JSON equality 규칙은 바꾸지 않습니다. Python floating-point rounding, 사용자 정의 equality/mapping/iterable와 JSON에 없는 객체는 별도의 표현 경계입니다.

## 옵션·클라이언트·페이지 증거

옵션은 `WithGetVolumesServerFields`와 전체 replacement인 `WithGetVolumesOptions` 두 factory를 제공합니다. Pointer·map·raw bytes를 복사해 callback과 실행 정책을 분리하며 원래 옵션은 순서대로 한 번 적용합니다. `PrepareGetVolumesOptions`는 context를 검사하고 HTTP 없이 owned 정책을 반환합니다. Connection은 원래 옵션을 cached Cinder v3 선택 전에 준비하고, 직접 package 함수는 Cinder source를 capture한 뒤 옵션을 적용합니다. Nova client는 선택하지 않습니다.

Canonical `volumes: []`는 실제 pinned `Resource.list`의 EOF 규칙처럼 즉시 종료합니다. 이 빈 페이지의 next 링크는 malformed이거나 다른 host를 가리켜도 읽거나 요청하지 않습니다. 수집기는 native `IsEmpty`나 typed `ExtractVolumes`를 호출하지 않고 raw canonical array 길이로 판단합니다.

Nonempty 페이지의 continuation은 실제 native `VolumePage.NextPageURL`의 canonical `volumes_links`를 사용합니다. 상대 href는 현재 페이지 URL에 맞춰 해석하며 최초 상세 collection의 escaped path와 origin·authority를 유지합니다. 다른 path·host, userinfo·fragment·opaque URL, malformed query와 반복 URL은 다음 요청 전에 거부합니다. 원래 링크의 query key는 허용하지만 초기 요청에 임의 필터를 추가하거나 synthetic marker, HTTP Link header, 다른 envelope로 대체하지 않습니다.

접수한 각 200의 body·header·status는 read/Close·decode·context/source 오류가 생겨도 독립적인 `Pages` 증거로 남습니다. Rejected HTTP·transport·native 오류에는 성공한 페이지를 만들어 붙이지 않습니다. 응답 row/attachment decode 오류는 해당 row가 나온 실제 page의 증거를 사용하고, 제공한 서버 필드 오류는 `resource.ErrInvalidOption`으로서 HTTP response를 만들지 않습니다. 나중 request의 거부·취소·source 변경에 이전 200 page를 해당 실패 응답인 것처럼 빌려 쓰지 않습니다.

클라이언트의 ordinary header는 capture한 값을 사용하고 provider 인증 토큰은 현재 native 인증 흐름을 따릅니다. Source의 provider·endpoint·resource base·type·microversion 변경은 오류로 끝냅니다. SDK가 provider 인증 retry를 대신 제거하거나 부분 결과를 성공으로 바꾸지 않습니다.

이 문서는 direct `get_volumes` helper의 Go mapping입니다. 별도 `list_volumes`, 단일 get/id/exists/search, native/Proxy API와 inherited Resource/session/cache는 각 선언의 지원 리뷰를 유지합니다. [서비스 README](README.md)와 [전체 README](../README.md)를 함께 참고하세요.

원본은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [get_volumes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L323-L346), [list_volumes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L52-L64), [v3 volumes 기본값](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L890-L921), [attachments descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)와 [list conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111), [Resource.list의 빈 목록 종료](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2340-L2359)를 확인했습니다. 페이지 링크는 Gophercloud `v2.15.0`의 [VolumePage.NextPageURL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L126-L135)을 기준으로 합니다.
