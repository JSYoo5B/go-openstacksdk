# Cinder service 동작: Python과 Go

`services.API`에는 Gophercloud가 생성한 `List` 외에 SDK가 직접 보내는 service 동작이 있습니다. Gophercloud v2.15.0은 `GET os-services`만 바인딩하므로 아래 PUT 요청은 이 SDK의 owned 코드가 보냅니다. 기준 소스는 [`_proxy.py` Services 절](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3423-L3616)과 [`service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/service.py)입니다.

| Python `conn.block_storage` | Go `services.API` | 요청 |
|---|---|---|
| `enable_service(service)` | `Enable(ctx, target, options...)` | `PUT os-services/enable` |
| `disable_service(service, reason=None)` | `Disable(ctx, target, options...)` | `PUT os-services/disable` 또는 `disable-log-reason` |
| `thaw_service(service)` | `Thaw(ctx, target, options...)` | `PUT os-services/thaw` |
| `freeze_service(service)` | `Freeze(ctx, target, options...)` | `PUT os-services/freeze` |
| `failover_service(service, cluster=None, backend_id=None)` | `Failover(ctx, target, options...)` | `PUT os-services/failover` 또는 `failover_host` |
| `set_service_log_levels(level=..., binary=None, server=None, prefix=None)` | `SetLogLevels(ctx, level, options...)` | `PUT os-services/set-log` |
| `get_service_log_levels(binary=None, server=None, prefix=None)` | `GetLogLevels(ctx, options...)` | `PUT os-services/get-log` |

```python
svc = next(conn.block_storage.services(binary="cinder-volume"))
conn.block_storage.disable_service(svc, reason="maintenance")
for level in conn.block_storage.get_service_log_levels(binary="cinder-api"):
    print(level.host, level.levels)
```

```go
api := services.New(volumeClient)
for svc, err := range api.List(ctx, services.WithListOptions(services.ListOpts{Binary: "cinder-volume"})) {
    if err != nil { return err }
    if _, err := api.Disable(ctx, services.TargetOf(svc), services.WithDisableReason("maintenance")); err != nil { return err }
    break
}
levels, err := api.GetLogLevels(ctx, services.WithLogBinary("cinder-api"))
if err != nil { return err }
for _, level := range levels.Levels { fmt.Println(level.Host, level.Levels) }
```

## 대상 service

Python은 service 인스턴스의 `binary`와 `host`로 본문을 만듭니다. Go는 이 두 값을 `ServiceTarget`으로 받으며, `NewServiceTarget(binary, host)`로 만들거나 `List` 결과에서 `TargetOf(svc)`로 얻습니다. Python에 service를 ID 문자열로 넘기면 그 문자열은 보내지 않고 `binary`와 `host`가 null이 됩니다. Go에서 같은 요청이 필요하면 필드가 nil인 `ServiceTarget{}`을 넘기면 됩니다.

`Enable`은 `{"binary": ..., "host": ...}`를 보냅니다. `Disable`은 같은 본문을 `disable`로 보내고, `WithDisableReason`의 사유가 비어 있지 않으면 Python처럼 `disable-log-reason`으로 바꾸고 `disabled_reason`을 더합니다. `Thaw`와 `Freeze`는 `{"host": ...}`만 보냅니다. `Failover`는 `{"host": ...}`에 `WithFailoverCluster`와 `WithFailoverBackendID` 값이 비어 있지 않을 때만 `cluster`와 `backend_id`를 더합니다.

이 동작들의 결과는 `ServiceRecord`입니다. 대상의 `binary`와 `host` 위에 응답 객체의 선언된 키를 덮어쓰며, Python `Service`에는 `resource_key`가 없으므로 최상위 객체를 그대로 읽습니다. `Body`는 선언된 Body 키 14개를 raw JSON으로 보관하고 `name`은 Python alias처럼 값이 없으면 `binary`를 씁니다. `Wire`는 실제 응답 객체, `Envelope`는 실제 응답 본문입니다.

## Microversion

`Enable`, `Disable`, `Thaw`, `Freeze`, `Failover`의 PUT은 Python `Service._action`처럼 `session.default_microversion`, 즉 client에 설정된 microversion만 보내고 설정이 없으면 microversion 헤더를 보내지 않습니다. `Failover`는 그 전에 Python `utils.supports_microversion(3.26)`처럼 Cinder version 문서를 한 번 읽습니다. 서버 범위가 3.26을 포함하고, client microversion이 설정되어 있다면 그 값도 3.26 이상일 때 `failover`를 쓰고, 아니면 `failover_host`를 씁니다. 버전 문서가 없으면 `failover_host`입니다.

`SetLogLevels`와 `GetLogLevels`는 Python `_assert_microversion_for(3.32)`를 따릅니다. client에 설정된 microversion이 있으면 discovery 없이 그 값을 쓰고, 없으면 3.32와 서버 최대값 중 작은 값을 고릅니다. 고른 값이 없거나 3.32보다 낮으면 요청 없이 `resource.ErrUnsupported`로 실패합니다.

## Log level

두 메서드는 `binary`가 없으면 Python처럼 빈 문자열을 보내고, `server`와 `prefix`가 없으면 null을 보냅니다. `WithLogServer("")`처럼 빈 문자열을 주면 빈 문자열이 그대로 나갑니다. `SetLogLevels`는 `level`을 더해 `set-log`로 보내며, Python이 None을 돌려주는 자리에 실제 응답을 담은 `ActionReceipt`를 돌려줍니다.

`GetLogLevels`는 응답 객체의 `log_levels` 배열을 읽습니다. Python이 `entry['binary']`처럼 키를 직접 읽으므로 Go도 `log_levels`가 없거나 배열이 아니거나 항목에 `binary`, `host`, `levels` 중 하나가 없으면 실제 응답을 담은 `resource.ResponseError`로 실패합니다. `LogLevel`의 `Binary`, `Host`, `Levels`는 JSON 타입이 맞을 때 채워지고 `Body`에 세 값의 raw JSON이 남습니다.

## 상태와 오류

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 비어 있거나 JSON이 아닌 동작 응답은 대상 값을 그대로 둡니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고 모든 오류는 `resource.OperationError`(Resource `services`, Operation `Enable`, `Disable`, `Thaw`, `Freeze`, `Failover`, `SetLogLevels`, `GetLogLevels`)로 감쌉니다.

## Python과 다른 점

Python `get_service_log_levels`는 generator라서 첫 항목을 꺼낼 때 요청을 보내지만, Go `GetLogLevels`는 호출할 때 요청을 보내고 결과를 slice로 돌려줍니다. Python `Level`과 `Binary` enum 대신 Go는 문자열을 받습니다. Python이 반환하는 Service 인스턴스는 인자로 받은 인스턴스를 갱신한 것이지만 Go는 새 `ServiceRecord`를 만듭니다. 헤더 옵션은 Go 확장이며 인증, microversion, content 헤더는 SDK가 소유합니다.
