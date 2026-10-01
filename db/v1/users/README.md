# Trove users

`service, err := conn.DatabaseV1(ctx)`로 준비한 서비스에서 `service.Users.InInstance(ctx, instanceRef)`를 사용합니다. Scope는 instance를 한 번 해석하며 명시적인 `resource.ID("instance-id")`는 부모 조회를 생략합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.database.users(instance)` | `scope.List(ctx)` 또는 `scope.All(ctx)` |
| `conn.database.find_user(name, instance, ignore_missing=False)` | `scope.Find(ctx, resource.Name(name))` |
| `conn.database.create_user(instance, name=name, password=password, databases=[...])` | `scope.Create(ctx, users.CreateOpts{...})` |
| 여러 `create_user` 호출 | `scope.CreateBatch(ctx, users.BatchCreateOpts{...})` |
| `conn.database.delete_user(name, instance=instance)` | `scope.Delete(ctx, resource.ID(name))` |
| `delete_user(..., ignore_missing=False)` | 삭제 호출에 `resource.WithMissingError()` 추가 |
| Python User 모델에 host 고정 옵션 없음 | `InInstance(..., users.WithHost(host))` |

핀으로 고정한 Python `User` Resource는 `allow_fetch=False`이며 Gophercloud v2.15.0도 사용자별 fetch를 제공하지 않습니다. `scope.Get(ctx, name)`은 목록에서 이름과 scope의 host를 정확하게 찾으며 사용자별 GET을 추가하지 않습니다. 실제 Trove API의 사용자별 GET 지원 여부와 이 pinned SDK의 fetch 노출 여부는 구분합니다.

Scope 결과는 SDK의 `users.UserResource`입니다. 기존 Gophercloud `User`를 embed하여 `Name`, `Password`, `Databases`를 유지하고, Gophercloud 모델에 없는 `Host`를 응답에서 보존합니다. 중첩 데이터베이스의 `character_set`도 `CharSet`에 보존합니다. 목록이 password를 돌려주지 않으면 `Password`는 빈 문자열이며, 생성 입력으로 응답 값을 채우지 않습니다. 기존 `service.Users.List(ctx, instanceID)`는 native `User` 결과를 그대로 사용합니다.

## 조회와 생성

아래 Go 조각은 `service`, `ctx`, `password`가 준비된 오류 반환 함수 안에서 사용하며, `users`, `databases`, `resource`, `fmt`를 import합니다.

```go
scope, err := service.Users.InInstance(ctx, resource.Name("db-server"))
if err != nil { return err }
user, err := scope.Find(ctx, resource.Name("reader"))
if err != nil { return err }
fmt.Println(user.Name, user.Host)

if err := scope.Create(ctx, users.CreateOpts{
    Name: "writer",
    Password: password,
    Databases: databases.BatchCreateOpts{{Name: "app"}},
}); err != nil { return err }
```

기본 scope의 `List`와 `Find(resource.Name(...))`는 모든 host를 조회합니다. 이름은 대소문자까지 정확하게 비교하며, 다른 페이지나 host에 같은 이름이 있으면 `resource.ErrAmbiguous`를 반환합니다. `Get(name)`과 `Find(resource.ID(name))`는 `%` host의 계정을 조회합니다. 기본 조회 누락 정책은 오류이며 `resource.WithIgnoreMissing()`을 추가하면 `nil, nil`을 반환합니다. 이름과 host는 서버 query가 아닌 전체 pagination 목록에서 비교합니다.

생성은 단일 사용자도 `{"users": [{...}]}` 배열로 제출합니다. Trove가 비동기 처리하고 생성 객체를 반환하지 않으므로 `Create`와 `CreateBatch`는 `error`만 반환합니다. `Databases`는 접근을 부여할 데이터베이스의 이름 입력이며, 이 scope가 데이터베이스를 새로 생성하거나 이름을 추가 조회하지 않습니다.

```go
if err := scope.CreateBatch(ctx, users.BatchCreateOpts{
    {Name: "reader", Password: password, Host: "%"},
    {Name: "reader", Password: password, Host: "192.0.2.10"},
}); err != nil { return err }
```

빈 batch, 빈 이름·password, 예약 이름 `root`, 같은 이름과 host 조합의 중복 계정, 잘못된 중첩 데이터베이스 이름은 전체 batch의 POST 전에 오류로 처리합니다. host를 생략한 입력과 `Host: "%"`는 같은 계정으로 취급합니다. 같은 이름과 서로 다른 host의 생성은 허용하지만, 이를 각각 조회·관리할 때는 아래 host scope를 사용합니다. 여러 항목의 원자적 생성 여부와 완료 시점은 Trove가 결정합니다.

Builder는 SDK가 구현합니다. 기존 `WithCreateField`는 요청 최상위 본문의 추가 확장 필드를 제공하며 `users` 배열을 덮어쓸 수 없습니다. `WithCreateOptions`로 batch를 교체할 수 있지만 단일 `Create`의 결과 입력은 한 항목이어야 합니다.

## host와 삭제

Trove 계정은 이름과 host의 조합입니다. 특정 host를 고정하면 `List`, `Get`, `Find`, `ResolveID`, `Delete`, `WaitDeleted`가 같은 계정을 유지합니다. scope는 생략된 CreateOpts.Host도 고정된 host로 채우며, 상충하는 Host는 POST 전에 거부합니다. 입력 batch를 변경하지 않습니다.

```go
hostScope, err := service.Users.InInstance(ctx, resource.ID("instance-id"),
    users.WithHost("192.0.2.10"))
if err != nil { return err }
id, err := hostScope.ResolveID(ctx, resource.Name("alice@office"))
if err != nil { return err }
user, err := hostScope.Get(ctx, id)
if err != nil { return err }
fmt.Println(user.Name, user.Host)
if err := hostScope.Delete(ctx, resource.ID(id)); err != nil { return err }
if err := hostScope.WaitDeleted(ctx, resource.ID(id)); err != nil { return err }
```

호스트를 고정하지 않은 scope에서 `Delete(ctx, resource.Name(name))`는 유일한 계정을 조회한 뒤 응답의 Host로 삭제합니다. `WaitDeleted`의 이름 조회도 대기를 시작할 때 계정을 한 번 해석하고 같은 host가 사라질 때까지 목록을 반복 조회합니다. 이미 삭제한 계정을 이어서 기다릴 때는 host scope와 명시적인 ID를 사용하여 다른 host의 동명 계정을 선택하지 않게 합니다.

명시적인 `resource.ID(name)`는 사용자 이름을 그대로 받으며 삭제 전 조회를 생략합니다. host scope의 ID는 그 host를 대상으로 하고, 기본 scope의 ID 조회·삭제·삭제 대기는 모두 `%` host를 대상으로 합니다. 기본 scope에서 non-default host 사용자를 이름으로 `ResolveID`하면 `resource.ErrUnsupported`입니다. host를 string ID로 잃지 않도록 `WithHost`가 필요합니다. `ResolveID`로 얻은 이름은 같은 host scope 안에서 ID로 재사용합니다.

SDK는 요청 내부에서 명시적인 `name@host` 식별자를 구성합니다. 이름에 들어 있는 `@`도 literal 이름으로 보존하며, URI escape와 점 escape를 적용합니다. WSGI와 Trove가 경로를 각각 decode하므로 `%`도 두 번의 decode를 거쳐 원래 이름·host로 남도록 보호합니다. 호출자가 이미 합친 `name@host` 문자열을 ID로 보내면 그 전체 문자열을 사용자 이름으로 해석하므로, 이름과 host를 따로 지정해야 합니다. 이 계약은 [Trove 계정 parser](https://github.com/openstack/trove/blob/master/trove/extensions/common/common.py#L76-L85), [WSGI 경로 decode](https://github.com/eventlet/eventlet/blob/master/eventlet/wsgi.py#L789-L791)와 [troveclient의 점 escape](https://github.com/openstack/python-troveclient/blob/master/troveclient/common.py#L32-L38)를 근거로 합니다.

삭제는 기본적으로 404를 무시합니다. `resource.WithMissingError()`는 미존재를 `resource.ErrNotFound`로 반환하며, 다른 HTTP 오류와 context 취소는 원래 cause를 보존합니다. 사용자 상태 필드가 없어 `Wait(ctx, ref, "ACTIVE")`는 `resource.ErrUnsupported`입니다. 목록 API에 노출되지 않은 query와 page-size 옵션도 HTTP 전에 거부합니다.

## 지원 범위

이 scope는 계정 목록·정확한 조회·단일/batch 생성·삭제·삭제 대기를 제공합니다. 기존 사용자의 password 변경, 이름·host 변경, 접근 권한 grant/revoke와 root 활성화는 별도 미구현입니다. `CreateOpts.Databases`는 새 계정의 초기 접근 설정이며 기존 계정의 권한 갱신을 대신하지 않습니다. 전체 Trove SDK parity를 완료했다는 의미가 아닙니다. [Database 서비스 README](../README.md)와 [Trove 사용자 HTTP 계약 테스트](../../../api/trove_users_contracts_test.go)를 참고하세요.
