# 전체 API 파사드

Gophercloud **v2.15.0**에서 서비스 클라이언트를 받는 공개 함수 **1,126개**를 194개 리소스 패키지에 제공합니다. [연산 목록](gophercloud_inventory.json)은 생성기가 같은 소스와 타입 정보로 만듭니다. `make generate`로 재생성하고 `make check`로 검증합니다.

각 패키지의 `New(client)`는 클라이언트를 보관하는 API 객체를 만듭니다. builder 입력은 SDK가 concrete options로 대체하고, 단순 조회의 선택 옵션은 `WithListOptions` 등의 함수로 제공합니다. 목록은 `iter.Seq2`, 일반 응답은 값과 `error`, 다운로드는 닫을 수 있는 스트림을 반환합니다. 여러 입력 builder가 필요한 Nova scheduler hints도 `WithCreateHintOpts`로 전달합니다.

[Keystone v2 인증](../identity/v2/tokens/README.md)은 토큰·사용자·catalog를 함께 반환합니다. [Nova 암호 조회](../compute/v2/servers/README.md)는 암호화된 문자열을 기본값으로 반환하고 복호화를 옵션으로 선택합니다. 이처럼 단일 extractor로 해석할 수 없는 응답도 SDK가 처리합니다.

[Cinder v2](../blockstorage/v2/snapshots/README.md)·[v3 Snapshot](../blockstorage/v3/snapshots/README.md)의 `UpdateMetadata`는 응답의 `metadata` 객체를 `map[string]any`로 반환합니다. 이전 `*Snapshot` 반환형을 사용한 호출자는 metadata map을 받도록 수정하고, Snapshot이 필요하면 `Get`을 명시적으로 호출합니다. [연산 목록](gophercloud_inventory.json)의 `result_policy: sdk_snapshot_metadata_object`는 두 연산의 안전한 결과 해석을 표시합니다. SDK는 원래 native 오류와 `json.Number`를 보존하고 잘못된 metadata envelope는 오류로 반환합니다. [HTTP 계약 테스트](snapshot_metadata_results_test.go)는 실제 PUT·응답·오류·확장 옵션과 별도의 Get을 검증합니다.

[Inspector 시작](../baremetalintrospection/v1/introspection/README.md)의 `StartIntrospection`은 pinned native 함수가 누락하는 query를 SDK helper로 보정합니다. [연산 목록](gophercloud_inventory.json)의 `request_policy: sdk_query_preserving_start`가 이 요청 실행 예외를 표시합니다. 필드가 없는 연산은 기존 native 요청 함수를 호출합니다. [감사된 규칙](../internal/cmd/sdkgen/audited_requests.go)은 해당 native 선언·signature·입력·결과 타입이 바뀌면 생성 오류로 중단해 재검토를 요구합니다.

```go
import (
    "context"
    "gophercloudsdk/network/v2/ports"
)

func example(ctx context.Context, api *ports.API) error {
    port, err := api.Create(ctx,
        ports.CreateOpts{NetworkID: "network-id", Name: "worker"},
        ports.WithCreateField("binding:host_id", "node-1"),
    )
    if err != nil { return err }
    _ = port
    for port, err := range api.List(ctx,
        ports.WithListOptions(ports.ListOpts{Limit: 100}),
        ports.WithListQuery("tags", "worker"),
    ) {
        if err != nil { return err }
        _ = port
    }
    return nil
}
```

`WithCreateField`는 기본 입력 필드 덮어쓰기를 거부하며 JSON 값을 옵션 생성 시 복사합니다. 확장 스키마와 microversion 요구 사항은 OpenStack 서비스가 검증합니다. 기본 필드는 concrete options로 지정합니다. HTTP 응답 오류는 `errors.As`로 원래 Gophercloud 오류까지 접근할 수 있습니다.

이 디렉토리의 완전성은 **고정한 Gophercloud 공개 연산에 대한 API 지원**을 뜻합니다. Python openstacksdk의 리소스 모델, 이름 해석, 복합 작업, 추가 서비스 지원까지 동등하다는 뜻은 아닙니다. API 파사드와 상위 SDK 지원은 별도로 추적합니다. 인증 관련 함수와 URL 도우미도 목록에 포함되므로 연산 수는 HTTP endpoint 수와 같지 않습니다.

계약 테스트는 Neutron 확장 필드와 페이지 순회, Nova의 scheduler hints, Glance의 JSON Patch, Swift의 업로드 및 다운로드 본문을 로컬 HTTP 서버에서 검증합니다. 전체 API 컴파일과 `go vet`도 수행합니다. 모든 연산의 실제 클라우드 동작을 검증한 것은 아닙니다.

[공통 정책 목록](resource_inventory.json)은 일반 Collection과 부모 범위를 구분합니다. `find`는 이름 검색, `delete`는 삭제·삭제 대기, `wait`는 문자열 상태 대기 capability입니다. 이름이 없는 모델도 명시 ID의 Get/Find를 제공할 수 있습니다. `scope`와 `parent`는 library-owned 부모 고정을 기록합니다. `kind: string_set`인 Nova tags는 Collection 대신 집합 API를 제공하므로 삭제 대기·ID 조회 capability로 해석하지 않습니다. Heat의 `kind: compound_identity`는 `Resources()`와 고정 name+ID scope, `compound_child`는 실제 owner와 resource_name을 확인하는 자식 범위, `event_log`는 stack/resource별 읽기 전용 이벤트 범위를 제공합니다. `kind: singleton`은 프로젝트 quota 범위입니다. Nova·Manila는 Get/Defaults/Detail/Update/Reset, Cinder는 Get/Defaults/Usage/Update/Reset, Neutron은 Get/Defaults/Detail/Update/Delete, Octavia는 Get/Update/Reset과 전역 Defaults, Designate는 Get/Update/Reset을 제공합니다. `kind: project_limits`인 Nova·Cinder limits는 일반 Fetch 또는 고정 프로젝트 Get만 제공하는 읽기 전용 singleton입니다. `kind: project_resource_quota`인 Magnum quota는 프로젝트와 resource 이름을 고정해 Create/Get/Update/Delete를 제공합니다. pinned Python에는 quota 선언이 없습니다. `kind: named_singleton`인 Manila quota class는 정확한 class 이름을 고정해 Get/Update를 제공합니다. Manila quota·quota class는 pinned Gophercloud에 없어 `source: sdk_owned`로 구분합니다. 서비스별 경로·성공 코드·기본값은 각 quota README에 설명합니다. SDK의 확장 응답 모델은 `model`, 원래 Gophercloud 모델은 필요한 경우 `upstream_model`에 기록합니다.

`source: sdk_owned`인 [Cyborg](../accelerator/v2/README.md) Device·Deployable·DeviceProfile·Attribute·AcceleratorRequest는 Gophercloud에 없는 구현입니다. 이들의 소스와 HTTP 계약은 Python/Cyborg 문서에 근거하며 native Gophercloud 1,126 연산 집계에 포함하지 않습니다.

[공통 정책 목록](resource_inventory.json)의 `metadata_scope: MetadataIn`은 Cinder v2/v3 Volume·Snapshot의 네 기존 collection에 추가한 SDK 소유 범위입니다. 별도 metadata Collection이나 Backup 경로를 뜻하지 않습니다. Get·POST Merge·PUT Replace·순서별 DeleteKeys는 [metadata 사용법](../blockstorage/metadata/README.md)에 설명하며 native Snapshot UpdateMetadata와 volume image metadata API를 유지합니다. [네 binding HTTP 계약](cinder_metadata_scopes_test.go)과 [공유 Connection 계약](../connection_metadata_test.go)은 요청·응답·부분 성공·토큰·고정 경로를 검증합니다.

API transport 목록과 별도로 [지원 판정](sdk_reviews.json)을 보존하고 [검증기](../internal/cmd/paritycheck/README.md)로 API·테스트·문서 근거와 고정 소스를 확인합니다. 미검토 연산은 `unresolved`이며 생성된 함수 수만으로 SDK 지원을 완료 처리하지 않습니다.

Compute Server·Cinder v2/v3 Volume/Snapshot·Image v2 Image의 `WaitForState/WaitForDelete`는 [서비스별 대기 기본값](../docs/service-waits.md)을 적용합니다. `resource_inventory.json`의 `service_wait`는 감사한 기존 여섯 collection에만 기록하며 native `WaitForStatus`나 공통 `WaitFor/WaitForDeletion`을 교체하지 않습니다.

Glance v2 Task의 `task_wait: WaitForTask`는 기존 ID-only collection의 [Task 대기와 396 재생성](../image/v2/tasks/README.md)을 표시합니다. `WaitForTask/WaitForTaskState`는 전용 concrete 옵션으로 success·failure·120초·2초 기본값을 적용하고, 정확한 396 실패에서 받은 type/input으로만 재생성해 같은 시간 제한으로 새 ID를 조회합니다. 실제 응답과 생성 증거를 오류와 함께 반환하며 native Task Get/Create/List와 공통 waiter는 유지합니다. Task Name·Delete capability를 추가하지 않습니다.

Glance의 [SDK import 제출](../image/v2/imageimport/README.md)은 `ImageImport.ImportImage(ctx, ref, options...)`와 이미 보유한 native Image를 받는 `ImportKnownImage`를 제공합니다. format 사전검증, method·원격 소스·root 저장소 목록과 생략/false 옵션, 확장 필드를 SDK가 조립하고 실제 202 응답을 반환합니다. 단일 저장소는 호환 헤더만 보내 복수 목록과 구분합니다. import는 Collection이 아니므로 기존 inventory의 미적용 행과 native Create/Get을 유지합니다. stage·이미지 생성·완료 대기는 별도 호출입니다.

Glance의 [SDK staging](../image/v2/imagedata/README.md)은 `ImageData.StageImage(ctx, ref, data, options...)`와 `StageKnownImage`를 제공합니다. queued 사전검증·단일 바이너리 전송·후속 metadata 조회와 선택적 Size/Headers를 SDK가 처리합니다. caller Reader를 닫거나 seek하지 않으며 binary PUT의 재전송·재인증·redirect를 막습니다. 실제 PUT204 acknowledgement는 후속 조회 실패에도 남고 GET200 응답·native Image와 독립적으로 보존됩니다. 기존 native Stage/Upload/Download와 미적용 Collection inventory는 유지합니다.

상위 [이미지 생성·import 흐름](../image/create-import.md)은 `conn.Image(ctx)`의 `Service.CreateAndImport`에서 사용합니다. 생성201·staging204·fresh 조회200·import202를 연결하고 remote 소스·단계별 concrete 옵션·선택적 active 대기를 제공합니다. 실제 단계 결과를 보존하는 수동 SDK 기능이며 native 연산 수와 전체 Python 지원 판정은 바꾸지 않습니다.
