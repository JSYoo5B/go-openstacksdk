# Nova 이전 action 호출

`API.EvacuateWithOptions`, `API.MigrateWithOptions`, `API.LiveMigrateWithOptions`는 openstacksdk의 [`evacuate_server`, `migrate_server`, `live_migrate_server`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)가 보내는 기본 요청을 그대로 보내도록 SDK가 직접 구현한 호출입니다. 생성된 `Evacuate`는 Gophercloud v2.15.0 `EvacuateOpts.OnSharedStorage`에 `omitempty`가 없어 항상 `onSharedStorage`를 보내는데, Nova는 2.14부터 이 키를 거부합니다. 생성된 `Migrate`는 option을 받지 않아 host를 보낼 수 없고, 생성된 `LiveMigrate`는 `block_migration`을 `*bool`로만 받아 `"auto"`를 보낼 수 없습니다.

세 호출 모두 [`Server._action`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)처럼 `POST servers/{id}/action`에 빈 `Accept` 헤더를 붙여 보냅니다. 400 미만의 모든 상태를 성공으로 받고, 400 이상은 Gophercloud 상태 오류를 원본 본문과 함께 `resource.OperationError`(Resource `servers`, Operation은 메서드 이름)로 감쌉니다. 서버 상태를 기다리거나 서버를 다시 읽지 않습니다.

## 요청

| Python 호출 | Go 호출 | 본문 |
|---|---|---|
| `evacuate_server(server)` | `EvacuateWithOptions(ctx, id)` | `{"evacuate": {}}` |
| `evacuate_server(server, host="h", admin_pass="", force=False)` | `WithEvacuateServerHost("h")`, `WithEvacuateServerAdminPass("")`, `WithEvacuateServerForce(false)` | `{"evacuate": {"host": "h", "adminPass": "", "force": false}}` |
| `evacuate_server(server, on_shared_storage=True)` | `WithEvacuateServerOnSharedStorage(true)` | `{"evacuate": {"onSharedStorage": true}}` |
| `migrate_server(server)` | `MigrateWithOptions(ctx, id)` | `{"migrate": null}` |
| `migrate_server(server, host="h")` | `WithMigrateServerHost("h")` | `{"migrate": {"host": "h"}}` |
| `live_migrate_server(server)` | `LiveMigrateWithOptions(ctx, id)` | 아래 표 |

`EvacuateServerOpts`는 nil 필드를 보내지 않고, 지정한 빈 문자열과 false는 그대로 보냅니다. Python과 마찬가지로 microversion에 따라 키를 막지 않으므로 Nova가 각 키를 검증합니다. `MigrateServerOpts.Host`가 빈 문자열이면 Python의 truthiness 검사처럼 host를 보내지 않습니다. host를 주었는데 선택한 microversion이 2.56보다 낮으면 HTTP 전에 `resource.ErrUnsupported`로 실패합니다.

`live_migrate_server`의 본문은 microversion에 따라 달라집니다. Go는 client에 선택한 microversion으로 본문을 고르며, `latest`는 2.30 이상으로 봅니다.

| 선택한 microversion | 요청 microversion | 기본 본문 | 규칙 |
|---|---|---|---|
| 2.30 이상 | 2.30 | `{"host": null, "block_migration": "auto"}` | host가 있을 때만 `force: true`를 더합니다. |
| 2.25 이상 2.30 미만 | 2.25 | `{"host": null, "block_migration": "auto"}` | host를 주면 force가 필요하고 force 키는 보내지 않습니다. |
| 2.25 미만 또는 미지정 | client 값 | `{"host": null, "block_migration": false, "disk_over_commit": false}` | host를 주면 force가 필요하고 `"auto"`는 거부합니다. |

`WithLiveMigrateServerBlockMigration(bool)`은 boolean 값을 보내고, `WithLiveMigrateServerAutoBlockMigration()`은 `"auto"`를 명시합니다. 둘은 서로를 지웁니다. `LiveMigrateServerOpts`에 `AutoBlockMigration`과 `BlockMigration`을 함께 지정하면 HTTP 전에 거부합니다. `WithLiveMigrateServerDiskOverCommit`은 2.25 미만 본문에서만 쓰이며 Python처럼 그 밖의 본문에서는 무시합니다. Python이 `ValueError`를 내는 경우(2.30 미만에서 force 없는 host, 2.25 미만의 `"auto"`)는 HTTP 전에 `resource.ErrInvalidOption`으로 실패합니다.

2.30과 2.25 본문은 Python처럼 해당 microversion을 요청 헤더로 보냅니다. 그래서 `WithLiveMigrateServerHeader`로 다른 값의 `X-OpenStack-Nova-API-Version`을 주면 HTTP 전에 거부합니다.

세 호출의 `With...Field`는 action 객체 안에 확장 키를 넣습니다. typed option이 가진 키는 확장으로 넣을 수 없고, 빈 서버 ID와 nil option, nil API도 HTTP 전에 실패합니다.

## Python과 다른 점

- Python은 서버마다 min(Server `_max_microversion` 2.100, 서버 최대값)을 협상해 보냅니다. Go는 client에 설정했거나 협상한 microversion 하나를 보냅니다.
- Python은 서버가 지원하는 microversion(session 기본값이 있으면 그 값)으로 live migration 본문과 migrate host 지원 여부를 판단합니다. Go는 client에 선택한 microversion으로 판단합니다.
- Python은 Server Resource나 ID를 받고 Go는 서버 ID 문자열을 받습니다.
- `With...Field`로 Python이 보낼 수 없는 확장 키를 보낼 수 있습니다.
