# Block Storage

Cinder v3 볼륨의 조회, iterator, 삭제, 상태 대기를 제공합니다. `conn`과 `ctx`는 [전체 README](../README.md)처럼 준비합니다. Go 조각은 `fmt`, `time`, `resource` 등을 import한 오류 반환 함수 안에서 사용합니다.

## openstacksdk 대응

| openstacksdk | gophercloudsdk |
|---|---|
| `conn.block_storage.get_volume(id)` | `service.Volumes.Get(ctx, id)` |
| `conn.block_storage.find_volume(name, ignore_missing=False)` | `service.Volumes.Find(ctx, resource.Name(name))` |
| `conn.block_storage.volumes(status="available")` | `service.Volumes.List(ctx, resource.WithStatus("available"))` |
| `conn.block_storage.wait_for_status(volume, status="available", wait=300)` | `service.Volumes.Wait(ctx, resource.ID(volume.ID), "available", resource.WithTimeout(5*time.Minute))` |
| `conn.block_storage.delete_volume(id)` | `service.Volumes.Delete(ctx, resource.ID(id))` |

SDK에서 볼륨 목록은 상세 목록 API를 사용합니다. Python의 `force`·`cascade` 삭제 옵션은 현재 상위 계층에 노출하지 않았습니다. [공식 Block Storage API](https://docs.openstack.org/openstacksdk/latest/user/proxies/block_storage_v3.html)

## 조회와 목록

Python:

```python
volume = conn.block_storage.find_volume("data", ignore_missing=False)
for volume in conn.block_storage.volumes(status="available"):
    print(volume.id, volume.size)
```

Go:

```go
service, err := conn.BlockStorage(ctx)
if err != nil { return err }
volume, err := service.Volumes.Find(ctx, resource.Name("data"))
if err != nil { return err }
fmt.Println(volume.ID, volume.Size)

for volume, err := range service.Volumes.List(ctx,
    resource.WithStatus("available"), resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(volume.ID, volume.Size)
}
```

query 확장은 `resource.WithQuery("key", "value")`를 사용합니다. 이름 검색은 정확한 일치를 확인하며 모든 페이지에 걸쳐 중복을 검사합니다. 이름 조회는 현재 클라이언트가 볼 수 있는 기본 조회 범위를 사용합니다.

## 대기와 삭제

```go
ready, err := service.Volumes.Wait(ctx, resource.ID(volume.ID), "available",
    resource.WithTimeout(5*time.Minute))
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Volumes.Delete(ctx, resource.ID(volume.ID)); err != nil {
    return err
}
```

`error`, `error_deleting`처럼 `error`로 시작하는 상태는 즉시 실패로 처리합니다. Delete는 삭제 요청의 성공을 반환하고 실제 삭제 완료까지 기다리지 않습니다.

microversion이 필요하면 연결 시 `sdk.WithMicroversion(sdk.BlockStorage, "3.60")`을 지정합니다. 명시 버전을 전송하며 서버 지원 범위를 자동 협상하지는 않습니다. 직접 endpoint를 설정한다면 `/v3/<project-id>/`가 포함된 URL을 사용합니다.

## 미지원 범위와 테스트

볼륨 생성·수정·크기 변경·attachment·snapshot·backup·volume type 관리와 서버와 볼륨을 연결하는 상위 작업은 아직 없습니다.

[blockstorage_test.go](blockstorage_test.go)는 Cinder의 실패 상태 패턴과 microversion 헤더를, [전체 통합 테스트](../collections_test.go)는 서비스 공통 정책을 검증합니다.
