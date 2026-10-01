# Heat stacks

`service, err := conn.OrchestrationV1(ctx)`로 서비스에 접근합니다. 기존 `service.Stacks.Get(ctx, name, id)` 등의 native API는 유지하며, `service.Stacks.Resources()`와 stack scope가 SDK 조회·대기 정책을 제공합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.orchestration.stacks()` | `service.Stacks.Resources().List(ctx)` |
| `find_stack(name, ignore_missing=False)` | `Resources().Find(ctx, resource.Name(name))` |
| `get_stack(id, resolve_outputs=False)` | `Resources().Get(ctx, id, stacks.WithResolveOutputs(false))` |
| name/id를 가진 Stack Resource | `stacks.StackIdentity{Name: name, ID: id}`와 `ForStack(identity)` |
| 문자열로 stack을 지정하는 작업 | `InStack(ctx, resource.Name(name))` 또는 `resource.ID(id)` |
| `delete_stack(stack, ignore_missing=True)` | `scope.Delete(ctx)` |
| `update_stack(stack, **attrs)` | `scope.Update(ctx, stacks.UpdateOpts{...}, ...WithUpdateField)` |
| 상태 대기 | `scope.Wait(ctx, "CREATE_COMPLETE", ...resource.WaitOption)` |
| 삭제 대기 | `scope.WaitDeleted(ctx, ...resource.WaitOption)` |

`Find(Name)`는 정확한 이름의 목록 summary를 반환합니다. Python Stack.find가 GET을 통해 상세 값을 반환하는 것과 다른 Go 매핑입니다. 상세 값이 필요하면 같은 identity를 고정한 scope에서 `Get`을 호출합니다. `Find(ID)`는 상세 GET 결과를 반환합니다. 조회 누락은 기본 오류이며 `resource.WithIgnoreMissing()`을 추가하면 `nil, nil`을 반환합니다.

## 복합 identity와 모델

Heat의 stack endpoint에는 이름과 canonical ID가 모두 필요합니다. ID만으로 이름을 추측하거나 `name/id` 문자열을 UUID 대신 저장하지 않습니다. `StackIdentity`는 두 값을 별도 필드로 보존합니다. `ForStack`은 알려진 identity를 검증하고 HTTP 조회를 생략합니다. `InStack`은 이름을 전체 목록에서 정확하게 해석하거나, 명시적인 ID를 Heat의 `/stacks/{id}` 조회와 canonical redirect로 해석합니다. 동일 이름이 여러 페이지에 있으면 `resource.ErrAmbiguous`로 거부합니다.

아래 Go 조각은 `service`, `ctx`가 준비된 오류 반환 함수 안에서 사용하며 `stacks`, `resource`, `fmt`를 import합니다.

```go
scope, err := service.Stacks.InStack(ctx, resource.Name("app"))
if err != nil { return err }
identity := scope.Identity()
fmt.Println(identity.Name, identity.ID)

stack, err := scope.Get(ctx, stacks.WithResolveOutputs(false))
if err != nil { return err }
fmt.Println(stack.Name, stack.ID, stack.Status, stack.Detailed)
```

`StackResource`는 native `RetrievedStack`의 필드를 embed합니다. 목록의 `ListedStack`은 이름·ID·상태·설명·시간·tags 같은 summary 필드만 공통 모델로 옮기며 `Detailed=false`입니다. 이때 빈 Outputs/Parameters 등은 값이 없다는 판정이 아닙니다. GET은 `Detailed=true`이며 실제 native 응답의 outputs·parameters·template description·rollback·timeout 필드를 보존합니다. 목록 항목마다 추가 GET을 보내지 않습니다.

GET의 `resolve_outputs` 기본값은 Python과 같은 true입니다. false는 `?resolve_outputs=False`를 URL에 직접 붙이며, redirect 중 query를 재적용하지 않습니다. 고정 scope의 GET 응답에서 이름이나 ID가 바뀌면 오류로 반환하여 다른 stack을 대상으로 진행하지 않습니다. ID 조회도 요청 ID와 응답의 canonical ID가 다르면 실패합니다. HTTP 404·403 등과 decode/context 오류는 원래 cause를 보존합니다.

## 알려진 stack의 변경과 대기

```go
scope, err := service.Stacks.ForStack(stacks.StackIdentity{
    Name: "app",
    ID: "stack-uuid",
})
if err != nil { return err }
if err := scope.UpdatePatch(ctx, stacks.UpdateOpts{
    Parameters: map[string]any{"size": "large"},
    Tags: []string{"ops", "test"},
}); err != nil { return err }
if _, err := scope.Wait(ctx, "UPDATE_COMPLETE"); err != nil { return err }
```

`Update`는 native PUT 계약상 template이 필요하고 `UpdatePatch`는 생략할 수 있습니다. concrete options와 기존 `WithUpdateField`/`WithUpdatePatchField`를 그대로 사용하며 builder는 SDK가 구현합니다. `scope.Abandon(ctx)`도 같은 name/ID를 사용합니다. 원래 `Create`, `Adopt`, `Preview`의 native 입력과 반환 모델은 변경하지 않습니다. 생성 결과가 최소 ID/links만 제공하므로 상세 StackResource를 만들어 채우지 않습니다.

```go
if err := scope.Delete(ctx); err != nil { return err }
if err := scope.WaitDeleted(ctx); err != nil { return err }
```

삭제는 기본 404 무시이며 `resource.WithMissingError()`로 엄격하게 처리합니다. scope의 상태·삭제 대기는 같은 name/ID endpoint를 반복 조회합니다. 기본 대기 시간은 공통 정책의 5분, polling은 2초이며 `resource.WithTimeout`/`WithPollInterval`로 변경합니다. `*_FAILED`와 `ERROR`는 즉시 `resource.ErrFailedState`로 반환합니다. 삭제 대기는 404 또는 `DELETE_COMPLETE`를 완료로 처리하고 인증 오류 등을 삭제 완료로 간주하지 않습니다.

정상 GET은 `DELETE_COMPLETE`와 `ADOPT_COMPLETE`의 상태·데이터를 그대로 반환합니다. Python Stack.fetch가 두 상태를 NotFound로 바꾸는 정책과 다릅니다. 삭제 판정은 `WaitDeleted`가 담당하며 `ADOPT_COMPLETE`를 삭제 완료로 추정하지 않습니다.

## 페이지와 남은 범위

SDK 목록은 Heat/Python의 top-level next link를 따릅니다. 상대 링크도 현재 page URL에서 해석합니다. `resource.WithPageSize(n)`처럼 양의 limit을 보냈는데 다음 링크가 없으면 마지막 canonical ID를 marker로 사용하여 빈 page까지 진행합니다. 이는 pinned Python Resource의 marker/limit fallback을 따른 매핑입니다. 반복 링크·반복 marker는 공통 pagination guard로 중단하고, iterator를 break하면 다음 요청을 보내지 않습니다. 이름 검색은 서버의 prefix 필터에 맡기지 않습니다.

이번 범위는 stack identity·summary/detail 조회·기존 변경 호출의 scope·상태와 삭제 대기입니다. StackResource의 dirty/cache state, Python의 전체 추가 응답 필드와 template/environment/file 보조 작업, stack resources/events의 부모 범위와 개별 action은 별도 비교·구현 단위입니다. 이 구현을 전체 Heat SDK parity 완료로 판정하지 않습니다. [서비스 README](../README.md), [HTTP 계약 테스트](../../../api/heat_stacks_contracts_test.go)와 [SDK 지원 판정대장](../../../docs/sdk-support-ledger.md)을 참고하세요.
