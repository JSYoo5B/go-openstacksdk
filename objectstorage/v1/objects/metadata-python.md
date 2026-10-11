# Swift object metadata: Python 이름 기반 호출

Gophercloud v2.15.0과 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`를 기준으로, [Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L563-L640)에 object 이름과 container 이름을 전달하는 세 호출을 제공합니다. stock Swift의 기존 HEAD/POST 경로를 사용합니다. `GetMetadata/SetMetadata/DeleteMetadata`의 기존 동작과 ABI는 유지합니다.

| Python | 새 Go API | 기본 HTTP |
|---|---|---|
| `get_object_metadata("file", container="assets")` | `HeadMetadata(ctx, "assets", "file")` | query 없는 HEAD |
| `set_object_metadata("file", container="assets", owner="ops")` | `SetMetadataFromFresh(ctx, "assets", "file", map[string]any{"owner":"ops"})` | HEAD → POST |
| `delete_object_metadata("file", container="assets", keys=["owner"])` | `DeleteMetadataAndRefresh(ctx, "assets", "file", []string{"owner"})` | 실제 삭제 시 HEAD → POST → HEAD |

[Object.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L222-L258)는 새 이름에서 생성한 Resource에 `last_modified_at`이 없어 먼저 HEAD합니다. 빈 setter 입력도 HEAD와 POST를 보냅니다. override의 `refresh=True`는 사용하지 않아 마지막 HEAD가 없습니다. 새 Go setter도 같은 순서를 사용하며 fresh 관측값과 POST 확인만 반환합니다.

[Object.delete_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L260-L320)는 nil/빈 keys를 즉시 끝냅니다. 새 Go 메서드도 HTTP 없이 `NoOp=true`를 반환합니다. nonempty keys는 fresh HEAD에서 읽은 lower-case custom suffix와 정확한 system alias를 비교하며, 없는 항목만 요청하면 HEAD 한 번으로 끝납니다. 실제 삭제가 있으면 전체 나머지 metadata를 POST한 뒤 마지막 HEAD에서 실제 상태를 읽습니다. 중복 일반 key는 첫 삭제 이후 무시합니다. `Owner`나 `X-Object-Meta-Owner`는 조회 후의 `owner`와 같지 않으므로 삭제 no-op입니다.

```go
api := objects.New(swiftClient)
change, err := api.SetMetadataFromFresh(ctx, "assets", "folder/file ?.txt",
    map[string]any{"owner": "ops", "empty": "", "is_content_type_detected": true},
    objects.WithMetadataHeaders(map[string]string{"X-Trans-Id-Extra": "metadata"}),
)
if err != nil {
    // Before와 Acknowledgement에 실제로 완료한 phase의 증거가 남는다.
    return err
}
_ = change.Acknowledgement

removed, err := api.DeleteMetadataAndRefresh(ctx, "assets", "folder/file ?.txt", []string{"owner"})
if err != nil {
    // 마지막 HEAD가 실패해도 Acknowledgement가 있으면 POST가 적용되었을 수 있다.
    return err
}
if !removed.NoOp {
    observed := removed.Refreshed.Metadata.Values
    _ = observed
}
```

setter는 Python처럼 falsey 입력을 필터링합니다. JSON `null`, false, 숫자 0, 빈 문자열/리스트/객체는 기존 값을 지우지 않으며, 기존 관측 값이 있다면 그대로 보냅니다. 삭제에는 deleter를 사용합니다. truthy JSON-domain 값은 공통 `cloudfilter.PythonString`으로 `True`, 정확한 큰 JSON 정수, 리스트/객체 repr로 변환합니다. 정확한 `X-Object-Meta-` prefix와 아래 canonical system alias 또는 정확한 lower-case wire 이름을 받습니다. prefix/alias 입력의 대소문자 검사는 Source와 같습니다.

| system alias | wire header |
|---|---|
| `accept_ranges` | `accept-ranges` |
| `content_disposition` | `content-disposition` |
| `content_encoding` | `content-encoding` |
| `content_type` | `content-type` |
| `delete_after` | `x-delete-after` |
| `delete_at` | `x-delete-at` |
| `is_content_type_detected` | `x-detect-content-type` |
| `manifest` | `x-object-manifest` |
| `access_control_allow_origin` | `access-control-allow-origin` |

관측한 system descriptor는 Source처럼 truthy 값만 보존합니다. `delete_after`는 Source `fields._convert_type`와 같은 공통 `DescriptorIntegerJSON`으로 변환해 zero를 제외합니다. decimal Unicode digit와 큰 정수는 보존하고 signed/공백/nonnumeric 문자열은 zero, `²`처럼 isdigit이지만 int가 받지 않는 문자열은 오류입니다. `is_content_type_detected`는 nonempty 문자열이면 Python `bool(str)`처럼 `True`가 됩니다. literal 문자열 `"False"`도 그 결과가 같습니다. canonical system alias를 입력하면 해당 관측 descriptor보다 우선합니다. wire alias만 입력하면 Source가 canonical 관측 descriptor를 뒤에 추가해 같은 wire header를 덮을 수 있습니다. 예를 들어 관측 `Content-Type=old`에 `content-type=new`를 입력한 POST는 `old`를 보냅니다. `content_type=new`로 입력하면 `new`를 보냅니다. 이 dictionary 병합 순서도 회귀로 검증합니다.

Pinned Source의 특이한 `delete_after` 삭제 분기도 유지합니다. 해당 key는 `delete_at`만 제거하고 실제 삭제 flag를 켜지 않습니다. 단독 `delete_after`는 관측 `delete_at`이 있으면 HEAD 후 no-op이며, 없거나 중복 요청으로 이미 제거했으면 `MetadataKeyError{Key:"delete_at"}`를 반환합니다. 다른 실제 삭제 key와 함께 요청한 경우에는 POST에서 `X-Delete-At`만 빠지고 관측 `X-Delete-After`는 유지됩니다. 이를 backend 예약 삭제 취소로 해석하지 않습니다.

`FreshMetadataChangeResult.Before/Acknowledgement/Refreshed`는 각각 actual HEAD/POST/최종 HEAD의 독립 원문·헤더·상태입니다. 요청한 값을 최종 조회 결과로 합성하지 않고 POST headers는 마지막 HEAD로 복사하지 않습니다. 새 메서드의 Source 최종 성공 상태는 200..399입니다. raw 응답을 읽거나 Close하는 과정에서 실패해도 완료한 phase 증거와 native/context cause를 보존하며 이후 phase를 시작하지 않습니다. accepted 응답이나 전체 read/merge/write를 자동으로 재시작하지 않습니다. 이 경로는 query 없는 HEAD라 Swift의 기본 symlink 처리를 사용하며, 기존 mutation의 `?symlink=get`와 own-link 거부 정책은 별도 API에 그대로 남습니다.

이 Go callable은 context, literal container/object 이름, independent typed observation/phase result를 받습니다. Python `get_object_metadata`는 이름과 mutable `Object` 입력 모두 항상 HEAD합니다. cached `Object` 입력에서 setter/deleter가 fresh HEAD를 생략하는 분기, 같은 Resource 반환/변이, 객체 descriptor 전체 및 Adapter/session은 별도 R1/C1/S1 계약입니다. 임의 Python `__str__`, JSON 밖 값, Python float/Unicode repr의 완전 동치는 지원 범위 밖입니다. 한 incoming Go map에서 같은 wire header를 만드는 alias 충돌은 순서가 없으므로 HTTP 전 오류입니다. 이는 입력 map의 경계이며, HEAD에서 관측한 custom metadata의 충돌은 별도로 아래와 같이 처리합니다. HTTP metadata 토큰/field 값과 응답 단일 값 검사는 Go의 명시적 경계입니다. response의 raw `Header`에는 typed `MetadataInfo` 밖 system descriptor도 남습니다. concurrent read/merge/write는 CAS가 없어 다른 쓰기를 덮을 수 있으며 server quota·ACL·version/SLO·symlink·복제 내구성을 SDK phase 성공으로 주장하지 않습니다.

Go `http.Header`는 응답 헤더의 원래 순서를 보존하지 못합니다. 서로 다른 관측 custom suffix가 같은 최종 wire header를 만드는 경우, Source는 응답 순서를 유지한 dictionary의 마지막 alias 값을 POST하지만 Go는 이를 재현할 수 없습니다. 예를 들어 HEAD가 `X-Object-Meta-Content_type: underscore first` 다음 `X-Object-Meta-Content-Type: hyphen last`를 주면 Source setter는 `Content-Type: hyphen last`를 보냅니다. 이전 Go의 suffix 정렬은 `underscore first`를 선택할 수 있었습니다. 이제 setter와 실제 삭제 경로는 POST 준비 시 남아 있는 관측 suffix끼리의 충돌을 `resource.ErrInvalidOption`으로 거부하며 `Before`와 `resource.ResponseError`에 성공한 HEAD의 실제 헤더·본문·상태를 보존합니다. 이는 Source와 다른 명시적 Go 경계이며, Source가 충돌을 오류로 처리한다는 뜻은 아닙니다. 조회, nil/빈 삭제, 없는 key의 HEAD-only no-op는 이 검사로 거부하지 않습니다. 삭제가 colliding suffix 하나를 제거해 충돌이 해소되면 POST와 최종 HEAD를 진행합니다. incoming/system descriptor 사이의 이미 결정된 병합 우선순위는 유지합니다.

Source는 JSON 변환과 custom `json.Marshaler` 실행 전에 고정하고 변환 직후, 옵션 완료, 각 응답 처리 후에도 다시 검사합니다. 입력과 nested 값은 callback 전에 소유하며 기존 concrete 옵션의 snapshot/마지막 값/병렬 재사용 동작을 재사용합니다. 새 HTTP 엔진이나 별도 서버 harness를 만들지 않고 공통 REST 및 public Gophercloud testhelper 기반 `internal/testcloud`와 기존 fault body/transport를 사용합니다.

| operation | catalog Source fingerprint | 회귀 |
|---|---|---|
| `python:object_store/v1/get_object_metadata` | `d94a0c23e14275ea5e351778b2213e010a08f8faace3f8d8628079cb5d42262f` | `TestSwiftPythonObjectMetadataHeadOwnsQueryFreeResponseAndSourceStatuses`, `TestSwiftPythonObjectMetadataObservedAliasesPreserveReadsAndDeleteNoOps` |
| `python:object_store/v1/set_object_metadata` | `41e6a3e10d05a0e595545f726865dd1f74de19bbf577d7b918ed8a0893f98102` | `TestSwiftPythonObjectMetadataSetterFreshMergeFalseyAliasesAndNoFinalHEAD`, `TestSwiftPythonObjectMetadataSetterEverySystemAliasAndEmptyInput`, `TestSwiftPythonObjectMetadataObservedAliasesRejectOnlyBeforePOST` |
| `python:object_store/v1/delete_object_metadata` | `2f18ff01dbaaade5e5347a3c6487640464c9777a72e0ce9297fb3e490abbb701` | `TestSwiftPythonObjectMetadataDeleteNoOpsAndObservedFinalHEAD`, `TestSwiftPythonObjectMetadataDeleteAfterPreservesSourceBranch`, `TestSwiftPythonObjectMetadataObservedAliasesRejectOnlyBeforePOST`, `TestSwiftPythonObjectMetadataObservedAliasesPreserveReadsAndDeleteNoOps` |

공통 회귀는 `TestSwiftPythonObjectMetadataPhysicalFailuresKeepEachCompletedPhase`, `TestSwiftPythonObjectMetadataOwnedInputsCallbacksConcurrentReuseAndPreflight`, `TestSwiftPythonObjectMetadataCapturesSourceBeforeCustomJSONMarshaler`입니다. 집중 race 실행과 중앙 review 판정을 완료했습니다. 전체 검증 근거는 [지원 판정대장](../../../docs/sdk-support-ledger.md#핵심-credentialneutronswiftnova-34개-마감)에 기록합니다.
