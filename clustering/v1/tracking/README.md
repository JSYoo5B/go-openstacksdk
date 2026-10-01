# Profile·Policy 변경 추적

`Profiles.Load/Track`과 `Policies.Load/Track`은 SDK가 소유하는 구체 handle을 반환합니다.
앱 개발자가 builder나 lifecycle interface를 구현할 필요가 없습니다. 기존 `UpdateOpts`와
`WithUpdate...` 함수를 `Edit`에서도 사용하며, `Commit`이 변경된 필드만 PATCH합니다.

| 작업 | pinned openstacksdk | Go |
|---|---|---|
| 서버 조회 | `get_profile(id)` / `get_policy(id)` | `API.Load(ctx, resource.ID(id))` |
| 기존 모델 사용 | Resource 객체 | `API.Track(model)`; HTTP 없음 |
| 편집 | 속성 대입 | `tracked.Edit(UpdateOpts{}, options...)` |
| 필드 삭제 | `del resource.name` | `RemoveName()`; profile의 `RemoveMetadata()` |
| 변경 전송 | `update_profile(resource)` / `update_policy(resource)` | `Commit(ctx, options...)` |
| 새 조회·변경 초기화 | `resource.fetch(session)` | `Refresh(ctx)` |
| 캐시·응답 구분 | Resource 내부 관리 | `Value()`는 병합 캐시, `Response()`는 마지막 성공 응답의 필드 |

```python
profile = conn.clustering.get_profile("PROFILE_ID")
profile.name = "renamed-profile"
profile.metadata = {"team": "platform"}
profile = conn.clustering.update_profile(profile)
```

```go
// context.Context ctx, *gophercloudsdk.Connection conn을 사용하는 함수 안에서
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.Edit(profiles.UpdateOpts{},
    profiles.WithUpdateName("renamed-profile"),
    profiles.WithUpdateMetadata(map[string]any{"team": "platform"})); err != nil {
    return err
}
profile, err := tracked.Commit(ctx, profiles.WithUpdateHeader("X-Audit-Tag", "rename"))
if err != nil { return err }
fmt.Println(profile.Name, tracked.Dirty(), tracked.Response().Body)
```

`profiles`는 `gophercloudsdk/clustering/v1/profiles`, `policies`는
`gophercloudsdk/clustering/v1/policies`, `resource`는 `gophercloudsdk/resource`입니다.
기존 stateless `API.Update`는 여전히 명시한 필드를 바로 전송하며 빈 요청을 거부합니다.
tracked handle의 변경 없는 `Commit`은 캐시 snapshot을 반환하고 HTTP를 수행하지 않습니다.

## 값과 변경분의 소유권

`Track`은 모델·Body·nested JSON·헤더를 복사합니다. 응답의 Body가 있으면 그 필드와 ID를
기준으로 삼으며 caller가 typed 모델의 ID만 바꾼 값은 route에 사용하지 않습니다. Body가
없는 수동 모델은 typed 필드로 로컬 캐시를 초기화합니다. 이때 `Response()`도 로컬 seed이며
성공한 HTTP 응답을 의미하지 않습니다. 입력에 없던 HTTP status나 header를 만들지 않습니다.

`Load`의 explicit ID는 최초 GET부터 이후 PATCH·Refresh까지 고정합니다. Name은 정확한 목록
검색 결과를 캐시로 사용하고 그 ID를 고정하며 별도 detail GET은 하지 않습니다. Load는
미존재를 오류로 반환하고 중복 이름을 거부합니다. 후속 응답에 다른 ID나 null·생략 ID가 와도
route를 바꾸지 않습니다. `Value()`와 `Response()`도 독립 snapshot이므로 반환 모델을 변경해도 handle은
편집되지 않습니다. `Edit`은 잘못된 옵션이나 JSON에서 부분 변경을 남기지 않습니다.

dirty 판정은 대입 시의 현재 값과 비교합니다. clean 필드에 같은 값을 대입하면 clean을
유지하지만, `A → B → A`로 바꾼 필드는 dirty로 남아 `A`를 전송합니다. 처음 조회한 값과
다시 같아졌다고 자동으로 변경을 지우지 않습니다. nested JSON은 정확한 숫자를 보존하고
boolean과 숫자의 JSON 타입을 구분합니다. Python의 `True == 1` 비교와 달리 JSON `true`와
`1`은 다른 값입니다. nested 값이나 getter snapshot을 직접 바꾸는 대신 `Edit`을 사용합니다.

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Policies.Load(ctx, resource.Name("existing-policy"))
if err != nil { return err }
current := tracked.Value().Name
if err := tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateName(current)); err != nil {
    return err
}
_, err = tracked.Commit(ctx) // 같은 값이면 PATCH 없음
return err
```

## 삭제·응답 병합·실패

`RemoveName()`은 캐시에서 필드를 제거하고 Commit에 `name:null`을 남깁니다. 없던 필드를
다시 제거하면 새로운 변경을 만들지 않습니다. Profile의 `RemoveMetadata()`도 같은 정책입니다.
`WithUpdateMetadata(nil)`은 캐시에 명시 null을 남기고, 빈 map은 `{}`를 남깁니다. null을
서버가 허용하는지는 서버 schema와 권한 정책이 결정합니다.

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
tracked, err := service.Profiles.Load(ctx, resource.ID("PROFILE_ID"))
if err != nil { return err }
if err := tracked.RemoveMetadata(); err != nil { return err }
_, err = tracked.Commit(ctx)
return err
```

성공한 응답은 field 단위로 캐시에 병합합니다. nested 객체를 재귀적으로 병합하지 않고 해당
객체 전체를 교체합니다. 응답이 편집한 필드를 생략해도 전송한 변경분은 clean으로 바뀌며 캐시에
현재 값이 남습니다. `Response().Body`에는 응답이 실제로 포함한 필드만 남습니다.

HTTP 오류나 잘못된 성공 envelope·JSON에서는 응답 병합이나 dirty 초기화를 수행하지 않습니다. 성공 HTTP 뒤에
`resource.ResponseError`가 발생하면 이미 mutation이 처리되었을 수 있습니다. SDK가 자동으로
재전송하지 않으며 `Refresh`로 상태를 확인한 후 후속 요청을 결정할 수 있습니다. Refresh 성공은
GET 전에 있던 변경분을 초기화하지만 HTTP 처리 중 새로 편집한 값은 유지합니다. Commit도 전송
snapshot 이후의 새 Edit을 덮어쓰거나 clean으로 만들지 않습니다. Commit/Refresh는 handle별로
직렬화하고 편집 revision을 SDK 내부에서 관리합니다.

## 지원 범위와 Go 정책

[공식 PATCH 계약](https://docs.openstack.org/api-ref/clustering/)의 mutable 필드는 Profile의
name·metadata와 Policy의 name입니다. `Edit`은 이 concrete 옵션을 사용하며 readonly 필드와
typed 필드의 대소문자 별칭을 `WithUpdateField`로 덮어쓰는 요청은 거부합니다. unknown vendor
필드의 명시 편집은 Go 확장이며 서버가 schema를 검사합니다. 응답의 unknown 필드는 캐시에
보존합니다. Python의 일반 descriptor/attrs가 모든 알려진 Body 필드에 대입을 허용하는 것과
달리 spec/type/data/소유자 등은 Go core 갱신 필드가 아닙니다.

Commit 옵션은 header만 받습니다. 본문 변경은 Edit에 전달하며 인증·버전·transport 헤더는
SDK가 소유합니다. context·서비스·선택 버전·옵션 검증은 clean Commit에도 적용하며 header만
지정해도 clean 본문에 PATCH를 보내지 않습니다. Python의 header dirty 및 `allow_empty_commit`
처리와 다른 명시적 Go 정책입니다.

성공 응답에는 `profile` 또는 `policy` 객체 envelope가 필요합니다. Python의 flat fallback이나
잘못된 JSON을 무시하는 처리를 따르지 않고 accepted 응답 오류를 증거와 함께 반환합니다.
캐시 snapshot과 실제 응답 필드를 별도로 노출하며 원본 Resource pointer를 변경하지 않습니다.
선택 client의 endpoint·인증·numeric microversion을 사용하고 자동 협상이나 버전 업그레이드는
하지 않습니다. per-call `base_path`, `prepend_key`, `has_body`, `retry_on_conflict`, `microversion`
및 임의 Resource subclass를 받는 Python generic commit 제어는 이 concrete API의 옵션이 아닙니다.

비교 기준은 pinned
[변경 관리자](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L214),
[commit 준비](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1914),
[응답 병합](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1380)입니다.
[dirty 객체 요청](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1231),
[generic update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716),
[Profile update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L226),
[Policy update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1030)도 같은 pin을 사용합니다.
[공통 상태 테스트](../../../internal/senlin/tracked_test.go)와
[실제 HTTP 계약](../../../api/clustering_lifecycle_test.go)은 no-op·sticky dirty·null·snapshot·병합,
고정 route와 실패·HTTP 처리 중 편집 보존을 검증합니다.

주요 회귀 anchor는 `TestClusteringTrackedNoOpStickyDirtyMergeAndSeparateResponse`,
`TestClusteringTrackedDeletionNullAndMetadataPresence`,
`TestClusteringTrackedExplicitLoadKeepsRequestIDAndNameResolvesOnce`,
`TestClusteringTrackedCommitFailureKeepsDirtyAndResponseEvidence`,
`TestClusteringTrackedRefreshMergeResetAndFailurePreservesChanges`,
`TestClusteringTrackedNewEditsSurviveCommitAndRefreshInFlight`입니다.

Cluster·Node도 SDK 소유 Track/Load·Edit·Commit·Refresh를 제공합니다. PATCH202의 Operation과 cached/actual Response, null·pending 버전 gate, 고정 경로는 [Cluster·Node 비동기 변경 추적](async/README.md)에 별도 설명합니다.

[Receiver 변경 추적](receivers/README.md)도 같은 SDK 소유 dirty 상태를 사용합니다. Receiver는 PATCH 200의 동기 모델이고 Action 명령 이름·Channel·Location을 비동기 작업으로 변환하지 않습니다.
