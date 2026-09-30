# Trove databases

`service, err := conn.DatabaseV1(ctx)`로 준비한 서비스에서 `service.Databases.InInstance(ctx, instanceRef)`를 사용합니다. Scope는 instance를 한 번 해석한 뒤 같은 instance 안에서 데이터베이스를 조회·생성·삭제합니다. instance 이름은 정확하게 검색하며, `resource.ID("instance-id")`는 부모 사전 조회를 생략합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.database.databases(instance)` | `scope.List(ctx)` 또는 `scope.All(ctx)` |
| `conn.database.find_database(name, instance, ignore_missing=False)` | `scope.Find(ctx, resource.Name(name))` |
| `conn.database.create_database(instance, name=name, character_set=charset)` | `scope.Create(ctx, databases.CreateOpts{Name: name, CharSet: charset})` |
| 여러 `create_database` 호출 | `scope.CreateBatch(ctx, databases.BatchCreateOpts{...})` |
| `conn.database.delete_database(name, instance=instance)` | `scope.Delete(ctx, resource.ID(name))` |
| `delete_database(..., ignore_missing=False)` | 삭제 호출에 `resource.WithMissingError()` 추가 |

핀으로 고정한 Python `Database` Resource는 `allow_fetch=False`이며 Gophercloud v2.15.0도 데이터베이스별 fetch를 제공하지 않습니다. Go의 `scope.Get(ctx, name)`은 instance 데이터베이스 목록을 끝까지 읽어 정확한 이름으로 찾습니다. `/instances/{id}/databases/{name}`에 GET을 보내지 않습니다. 데이터베이스에는 UUID가 없으며 `Name`이 식별자입니다.

## 조회와 생성

아래 Go 조각은 `service`, `ctx`가 준비된 오류 반환 함수 안에서 사용하며, `databases`, `resource`, `fmt`를 import합니다.

```go
scope, err := service.Databases.InInstance(ctx, resource.Name("db-server"))
if err != nil { return err }

database, err := scope.Find(ctx, resource.Name("app"))
if err != nil { return err }
fmt.Println(database.Name, database.CharSet, database.Collate)

if err := scope.Create(ctx, databases.CreateOpts{
    Name: "logs",
    CharSet: "utf8mb4",
    Collate: "utf8mb4_unicode_ci",
}); err != nil { return err }
```

`Find`는 이름을 대소문자까지 정확하게 비교하며, 중복 결과가 있으면 `resource.ErrAmbiguous`를 반환합니다. 기본 누락 정책은 오류이며 `resource.WithIgnoreMissing()`을 추가하면 `nil, nil`을 반환합니다. server query로 이름을 추측하지 않고 페이지를 따라가므로 prefix 결과나 다음 페이지의 중복 이름도 임의로 선택하지 않습니다. 목록의 `character_set`은 모델의 `CharSet`에 보존합니다.

```go
if err := scope.CreateBatch(ctx, databases.BatchCreateOpts{
    {Name: "app"},
    {Name: "logs", CharSet: "utf8mb4"},
}); err != nil { return err }
```

단일 생성도 Trove의 `{"databases": [{...}]}` 배열 본문을 사용합니다. 생성 API는 비동기 처리하며 응답에 데이터베이스 객체가 없으므로 두 메서드는 `error`만 반환합니다. 빈 batch, 중복 이름, 빈 이름과 64바이트를 넘는 이름은 POST 전에 오류로 처리합니다. charset과 collate를 생략하면 datastore의 기본값을 사용합니다. 여러 항목의 원자적 생성 여부와 완료 시점은 Trove가 결정합니다.

기존 `WithCreateField`는 요청 본문의 추가 확장 필드를 제공하며 builder는 SDK가 구현합니다. `WithCreateOptions`는 batch 입력을 교체하지만 단일 `Create`에서는 결과 batch가 한 항목이어야 합니다. 이름·charset·collate는 typed `CreateOpts`로 지정합니다.

## 삭제와 지원 범위

```go
if err := scope.Delete(ctx, resource.ID("app")); err != nil { return err }
if err := scope.WaitDeleted(ctx, resource.ID("app")); err != nil { return err }
```

삭제는 기본적으로 404를 무시하며 다른 HTTP 오류는 보존합니다. `resource.Name`으로 삭제하면 정확한 조회를 먼저 수행하고, `resource.ID`에는 데이터베이스의 이름을 그대로 전달합니다. 공백·`?`·`#`가 포함된 이름도 SDK가 URL path를 escape합니다. `WaitDeleted`는 같은 instance의 목록을 반복 조회하여 이름이 사라질 때까지 대기합니다.

데이터베이스 모델에 상태가 없으므로 `Wait(ctx, ref, "ACTIVE")`는 `resource.ErrUnsupported`입니다. 목록 API에 노출되지 않은 query나 page-size 옵션도 HTTP 전에 같은 오류로 처리합니다. 서비스가 제공한 pagination 링크는 자동으로 따라갑니다.

이 scope는 데이터베이스 생성·조회·삭제에 한정합니다. 사용자 credential 갱신·접근 권한 관리와 데이터베이스 이름 변경은 여기서 제공하지 않습니다. [Database 서비스 README](../README.md)와 [Trove HTTP 계약 테스트](../../../api/trove_databases_contracts_test.go)를 참고하세요.
