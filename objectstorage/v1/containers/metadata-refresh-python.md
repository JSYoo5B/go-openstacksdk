# Swift container metadata: Python 기본 refresh

기준은 Gophercloud v2.15.0과 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. 고정 [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L210-L251)의 두 미완료 행에 대해 stock Swift container POST/HEAD 경로의 실제 호출 순서를 추가합니다. 기존 `SetMetadata`/`DeleteMetadata` ABI와 엄격한 상태 정책은 유지합니다.

| Python | 새 Go API | 기본 HTTP |
|---|---|---|
| `set_container_metadata("container", owner="ops")` | `SetMetadataAndRefresh(ctx, "container", map[string]any{"owner":"ops"})` | bodyless POST → HEAD |
| `delete_container_metadata("container", ["owner"])` | `DeleteMetadataKeys(ctx, "container", []string{"owner"})` | bodyless POST |

[BaseResource.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L73-L92)의 기본 refresh는 true입니다. 빈 setter 입력도 POST와 HEAD를 수행하며, 빈 삭제 keys도 한 번 POST합니다. 삭제 중복은 Python dict comprehension처럼 합쳐집니다. 정확한 `X-Container-Meta-` prefix가 붙은 키는 다시 prefix하지 않습니다. Python의 prefix 검사는 대소문자를 구별하므로 Go도 같은 입력 변환을 적용합니다.

```go
api := containers.New(swiftClient)
result, err := api.SetMetadataAndRefresh(ctx, "container", map[string]any{
    "owner": "ops",
    "enabled": false,
    "X-Container-Meta-External": "value",
})
if err != nil {
    // result.Acknowledgement가 있으면 POST가 적용되었을 수 있다.
    return err
}
observed := result.Refreshed.Metadata.Values["owner"]
_ = observed
```

`MetadataRefreshResult`의 Acknowledgement와 Refreshed는 각각 실제 POST와 HEAD의 원문·헤더·상태입니다. 요청값을 HEAD 결과로 합성하지 않으며, refresh 실패는 이미 성공한 POST를 재전송하거나 rollback하지 않습니다. 쓰기 metadata 헤더는 refresh HEAD로 복사하지 않고 ordinary 옵션/선택한 컨테이너/인증은 유지합니다.

JSON-domain 입력은 공통 `cloudfilter.PythonString`으로 `None`, `True`, `False`, 정확한 JSON 정수 및 리스트/객체 repr로 바꿉니다. 입력과 nested 값은 callback 전에 직렬화·변환해 소유합니다. 임의 Python 객체의 `__str__`, Go custom marshaler 효과, JSON 밖 값, Python float repr/Unicode 버전의 완전 동치는 범위 밖입니다. Go map의 순서가 없으므로 같은 wire 이름으로 충돌하는 alias는 HTTP 전에 거부합니다. 인증/전송 헤더는 기존 metadata 옵션의 SDK 소유 정책을 유지합니다.

새 메서드는 Source `raise_from_response`의 최종 200..399 정책을 실제 callable로 제공합니다. 거부 응답의 native HTTP 원인, accepted/rejected 본문 Read·Close 오류, 취소 cause를 공통 REST로 보존합니다. mutable Container cache, Adapter/session 및 server quota·ACL·복제 내구성은 별도 계약이며 새 callable 완료로 전체 Resource 동치를 주장하지 않습니다.

| operation | catalog Source fingerprint | 회귀 |
|---|---|---|
| `python:object_store/v1/set_container_metadata` | `21e83b033927c89bb1a9bc33a3c02f9568f7641f4c0feb92293a5116439e2569` | `TestSwiftPythonContainerMetadataSetterRefreshesObservedStateAndSourceValues`, `TestSwiftPythonContainerMetadataRefreshFailuresKeepAcknowledgementWithoutReplay` |
| `python:object_store/v1/delete_container_metadata` | `11b38c4d607fa533f5ebb1c6986282857a09d2a7754eae747f7d49beb832b844` | `TestSwiftPythonContainerMetadataEmptyAndPrefixedDeleteFollowSourcePOST` |

공통 회귀는 `TestSwiftPythonContainerMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `TestSwiftPythonContainerMetadataPreflightAndSourceStatusPolicy`입니다. 테스트는 기존 package transport/body와 public Gophercloud testhelper 기반 `internal/testcloud`를 재사용합니다. 새 HTTP 엔진은 없습니다. Source getter `get_container_metadata`는 이미 판정된 행으로 중복 집계하지 않습니다. 구현/회귀는 작성했으며 집중 Go/race 실행과 중앙 review 갱신은 부모 runner가 진행합니다.

컨테이너 setter에는 실제 Source 선택 인자 `refresh`를 `WithMetadataRefresh(false)`로 제공합니다. 기본과 nil은 true이고 나중 scalar 옵션이 앞선 값을 바꿉니다. `WithMetadataRefreshOpts`는 전체 config를 소유 복사하고 교체합니다. setter의 일반 헤더는 `WithMetadataRefreshHeader(s)`, 삭제는 기존 `WithMetadataHeader(s)`를 사용합니다. 옵션 factory와 callback 결과는 pointer/map을 독립 복사해 반복·동시 재사용합니다.

[Container._system_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/container.py#L22-L34)의 실제 8개 alias를 모두 지원합니다: `content_type`, `is_content_type_detected`, `versions_location`, `history_location`, `read_ACL`, `write_ACL`, `sync_to`, `sync_key`. 정확한 소문자 wire 이름도 같은 헤더가 됩니다. 빈 alias 삭제값은 실제 해당 system 헤더를 empty로 POST하며 클라이언트가 성공한 서버 상태를 추론하지 않습니다. 추가 회귀는 `TestSwiftPythonContainerMetadataEverySystemAliasAndPrefixedConvention`과 `TestSwiftPythonContainerMetadataRefreshFalseLaterOptionsAndReusableSnapshots`입니다.
