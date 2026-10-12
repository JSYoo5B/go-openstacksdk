# Neutron LBaaS, flavor, metering: pinned Python and Go

이 문서는 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 `conn.network` 메서드 가운데 Neutron LBaaS v2, Neutron flavor와 service profile, metering label과 rule, 그리고 floating IP 별칭인 `find_ip`와 `ips`를 Go SDK와 비교합니다. Go 쪽 기준은 Gophercloud v2.15.0과 이 저장소의 생성 facade입니다.

Python 원본은 [`_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)와 각 resource 파일입니다. LBaaS resource는 [`load_balancer.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/load_balancer.py), [`listener.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/listener.py), [`pool.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/pool.py), [`pool_member.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/pool_member.py), [`health_monitor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/health_monitor.py)에 있습니다. 나머지는 [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/flavor.py), [`service_profile.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/service_profile.py), [`metering_label.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label.py), [`metering_label_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label_rule.py), [`floating_ip.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/floating_ip.py)입니다.

## 판정 요약

58개 메서드 가운데 Go 매핑으로 판정한 메서드는 없습니다. 56개는 Gophercloud v2.15.0에 해당 Neutron route의 binding이 없고, `find_ip`와 `ips`는 Go API가 요청의 일부만 재현합니다.

| Python 메서드 | Go 호출 | 판정 |
| --- | --- | --- |
| `create_load_balancer(**attrs)` | 없음 | 미해결 |
| `delete_load_balancer(lb, ignore_missing=True)` | 없음 | 미해결 |
| `find_load_balancer(name_or_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_load_balancer(lb)` | 없음 | 미해결 |
| `load_balancers(**query)` | 없음 | 미해결 |
| `update_load_balancer(lb, **attrs)` | 없음 | 미해결 |
| `create_listener(**attrs)` | 없음 | 미해결 |
| `delete_listener(listener, ignore_missing=True)` | 없음 | 미해결 |
| `find_listener(name_or_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_listener(listener)` | 없음 | 미해결 |
| `listeners(**query)` | 없음 | 미해결 |
| `update_listener(listener, **attrs)` | 없음 | 미해결 |
| `create_pool(**attrs)` | 없음 | 미해결 |
| `delete_pool(pool, ignore_missing=True)` | 없음 | 미해결 |
| `find_pool(name_or_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_pool(pool, fields=None)` | 없음 | 미해결 |
| `pools(**query)` | 없음 | 미해결 |
| `update_pool(pool, **attrs)` | 없음 | 미해결 |
| `create_pool_member(pool, **attrs)` | 없음 | 미해결 |
| `delete_pool_member(member, pool, ignore_missing=True)` | 없음 | 미해결 |
| `find_pool_member(name_or_id, pool, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_pool_member(member, pool, fields=None)` | 없음 | 미해결 |
| `pool_members(pool, **query)` | 없음 | 미해결 |
| `update_pool_member(member, pool, **attrs)` | 없음 | 미해결 |
| `create_health_monitor(**attrs)` | 없음 | 미해결 |
| `delete_health_monitor(hm, ignore_missing=True)` | 없음 | 미해결 |
| `find_health_monitor(name_or_id, ignore_missing=True, **query)` | 없음 | 미해결 |
| `get_health_monitor(hm)` | 없음 | 미해결 |
| `health_monitors(**query)` | 없음 | 미해결 |
| `update_health_monitor(hm, **attrs)` | 없음 | 미해결 |
| `create_flavor(**attrs)` | 없음 | 대응 |
| `delete_flavor(flavor, ignore_missing=True)` | 없음 | 대응 |
| `find_flavor(name_or_id, ignore_missing=True, **query)` | 없음 | 대응 |
| `get_flavor(flavor, fields=None)` | 없음 | 대응 |
| `flavors(**query)` | 없음 | 대응 |
| `update_flavor(flavor, **attrs)` | 없음 | 대응 |
| `associate_flavor_with_service_profile(flavor, service_profile)` | 없음 | 대응 |
| `disassociate_flavor_from_service_profile(flavor, service_profile)` | 없음 | 대응 |
| `create_service_profile(**attrs)` | 없음 | 대응 |
| `delete_service_profile(sp, ignore_missing=True)` | 없음 | 대응 |
| `find_service_profile(name_or_id, ignore_missing=True, **query)` | 없음 | 대응 |
| `get_service_profile(sp, fields=None)` | 없음 | 대응 |
| `service_profiles(**query)` | 없음 | 대응 |
| `update_service_profile(sp, **attrs)` | 없음 | 대응 |
| `create_metering_label(**attrs)` | 없음 | 대응 |
| `delete_metering_label(label, ignore_missing=True)` | 없음 | 대응 |
| `find_metering_label(name_or_id, ignore_missing=True, **query)` | 없음 | 대응 |
| `get_metering_label(label, fields=None)` | 없음 | 대응 |
| `metering_labels(**query)` | 없음 | 대응 |
| `update_metering_label(label, **attrs)` | 없음 | 대응 |
| `create_metering_label_rule(**attrs)` | 없음 | 대응 |
| `delete_metering_label_rule(rule, ignore_missing=True)` | 없음 | 대응 |
| `find_metering_label_rule(name_or_id, ignore_missing=True, **query)` | 없음 | 대응 |
| `get_metering_label_rule(rule, fields=None)` | 없음 | 대응 |
| `metering_label_rules(**query)` | 없음 | 대응 |
| `update_metering_label_rule(rule, **attrs)` | 없음 | 대응 |
| `find_ip(name_or_id, ignore_missing=True, **query)` | `floatingips.API.Find(ctx, resource.ID(id), resource.WithIgnoreMissing())` (ID 조회만) | 미해결 |
| `ips(**query)` | `floatingips.API.All(ctx, resource.WithQuery(k, v)...)` (서버 query만) | 미해결 |

## 공통 Python 동작

이 범위의 resource는 모두 Python 기본 helper를 그대로 사용합니다. `create_*`는 resource key로 감싼 body를 `POST`하고, `update_*`는 바뀐 속성만 담아 `PUT /{collection}/{id}`를 보냅니다. `get_*`는 `GET /{collection}/{id}`이며, `fields`를 받는 메서드는 Python 속성 이름을 서버 필드 이름으로 바꾼 뒤 `fields` query로 붙입니다. `delete_*`는 `ignore_missing=True`가 기본값이라 404를 `None`으로 바꿉니다. `find_*`는 먼저 `GET /{collection}/{name_or_id}`를 시도하고, 404, 400, 403 중 하나이면 목록 조회로 넘어가 ID 또는 이름이 정확히 같은 항목 하나를 고릅니다. 목록 단계의 `name` query는 `_query_mapping`에 `name`이 있는 resource에서만 붙으므로 health monitor, service profile, metering rule은 이름 query 없이 전체 목록을 읽습니다. 목록 메서드는 `_query_mapping`에 있는 키만 서버로 보내고, 선언된 Body 속성 중 query에 없는 키는 받은 결과를 로컬에서 거르며, 나머지 키는 버립니다. 모든 resource가 `_allow_unknown_attrs_in_body = True`라서 선언되지 않은 속성도 create와 update body에 들어갑니다.

Go SDK가 이 route들을 지원하게 되더라도 위 기본값은 따로 맞춰야 합니다. 예를 들어 delete는 `Remove`나 `resource.IgnoreMissing`으로 404를 숨겨야 하고, find는 ID 조회 뒤 이름 목록 조회로 넘어가는 helper가 필요합니다.

## Neutron LBaaS v2

Python은 load balancer, listener, pool, member, health monitor를 Neutron `network` 서비스 endpoint 아래의 `/v2.0/lbaas/loadbalancers`, `/v2.0/lbaas/listeners`, `/v2.0/lbaas/pools`, `/v2.0/lbaas/pools/{pool_id}/members`, `/v2.0/lbaas/healthmonitors`로 보냅니다. body key는 `loadbalancer`, `listener`, `pool`, `member`, `healthmonitor`이고, `create_pool_member`는 `pool`을 ID로 바꿔 URI에만 넣습니다. 이 resource들은 `NetworkResource`를 상속하지 않으므로 `revision_number`나 `if-match` header를 다루지 않습니다.

Gophercloud v2.15.0은 Neutron LBaaS binding을 제거했습니다. 이 저장소의 `loadbalancer/v2` 패키지는 Octavia API이며, 상대 경로는 `v2.0/lbaas/...`로 같지만 서비스 catalog의 `load-balancer` endpoint에 묶여 있습니다. 그래서 `Connection.LoadBalancerV2`로 얻은 facade는 Python이 사용하는 Neutron endpoint로 요청을 보내지 않습니다. Octavia 모델에는 Neutron pool의 `status`, `status_description`, `vip_id`, `health_monitor_status` 같은 응답 필드도 없습니다. Octavia facade를 network 서비스 client에 억지로 연결하는 방식은 SDK가 계약으로 검증한 경로가 아니라서 매핑으로 인정하지 않았습니다. 같은 이유로 Python 쪽 query 이름 차이도 그대로 남습니다. Health monitor 목록은 `is_admin_state_up`을 서버의 `adminstate_up` query로 보내고, load balancer 목록은 `tenant_id`와 `provider`를 서버 query로 보냅니다.

## Neutron flavor와 service profile

Neutron flavor는 Nova flavor나 Octavia flavor(`loadbalancer/v2/flavors`의 `/lbaas/flavors`)와 다른 API입니다. Python은 `/v2.0/flavors`와 `/v2.0/service_profiles`를 사용하고 body key는 `flavor`, `service_profile`입니다. `associate_flavor_with_service_profile`은 `POST /v2.0/flavors/{flavor_id}/service_profiles`에 `{"service_profile": {"id": ...}}`를 보내고 응답 JSON을 dict 그대로 돌려줍니다. `disassociate_flavor_from_service_profile`은 `DELETE /v2.0/flavors/{flavor_id}/service_profiles/{service_profile_id}`를 보내고 `None`을 돌려주며 `ignore_missing` 인자가 없습니다. 두 메서드 모두 Proxy의 `raise_exc=False` 기본값을 그대로 쓰므로 HTTP 오류 상태를 예외로 바꾸지 않습니다. Gophercloud v2.15.0과 이 저장소에는 이 route들의 binding이 없습니다.

## Metering label과 rule

Python은 `/v2.0/metering/metering-labels`와 `/v2.0/metering/metering-label-rules`를 사용하고 body key는 `metering_label`, `metering_label_rule`입니다. Rule의 `remote_ip_prefix`는 deprecated 속성으로 남아 있으며 `source_ip_prefix`와 `destination_ip_prefix`가 후속 속성입니다. Metering rule은 `name` query를 선언하지 않고 응답에도 이름이 없으므로 `find_metering_label_rule`의 목록 단계는 사실상 ID 일치만 확인합니다. Gophercloud v2.15.0과 이 저장소에는 metering binding이 없습니다.

## Floating IP 별칭 `find_ip`, `ips`

두 메서드는 `FloatingIP` resource를 사용하는 별칭입니다. `FloatingIP`는 `name`을 `floating_ip_address`로 정의하므로 Python `find_ip("203.0.113.5")`는 `GET /v2.0/floatingips/203.0.113.5`가 404일 때 이름 query 없이 `GET /v2.0/floatingips` 전체 페이지를 읽고, ID 또는 `floating_ip_address`가 같은 항목 하나를 고릅니다.

Go의 `floatingips.API.Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`는 첫 단계인 ID 조회와 404를 `nil, nil`로 바꾸는 동작만 재현합니다. ID 조회가 400이나 403으로 실패하면 Python은 목록 단계로 넘어가지만 Go는 오류를 그대로 돌려줍니다. 이 binding은 이름 adapter와 identity find 설정이 없으므로 `Find(ctx, resource.Name(x))`와 `api.Resources.FindIdentity`는 HTTP 요청 없이 `resource.ErrUnsupported`를 돌려줍니다. 테스트는 `network/v2/extensions/layer3/floatingips/python_parity_ips_test.go`의 `TestPythonIPsFindIPOnlyReproducesTheIDLookup`입니다.

`ips()`는 Go `floatingips.API.All(ctx)`와 같은 `GET /v2.0/floatingips`를 보내고 `floatingips_links`의 next 링크를 따라갑니다. `floating_network_id`, `status`처럼 Python `_query_mapping`에 있는 키는 `resource.WithQuery`로 같은 서버 query를 만들 수 있습니다. 다만 Python은 `name`(실제로는 `floating_ip_address`), `dns_name`, `qos_policy_id`처럼 query에 없는 Body 속성을 로컬에서 거르는데, 이 collection에서는 `resource.WithName`과 `resource.WithBodyFilter`가 HTTP 요청 없이 `resource.ErrUnsupported`를 돌려줍니다. Go `resource.WithStatus`는 `status` query를 보내면서 결과를 대소문자 구분 없이 한 번 더 거르므로, Python `status` 인자와 똑같이 서버 필터만 쓰려면 `resource.WithQuery("status", ...)`를 사용합니다. 테스트는 같은 파일의 `TestPythonIPsListServerQueryAndPagesButNoLocalBodyFilters`입니다. `find_floating_ip`, `floating_ips` 같은 본래 이름의 메서드는 이 문서의 범위가 아닙니다.

## 미해결 메서드와 부족한 점

Neutron service flavor·service profile·metering label·metering label rule 메서드는 SDK 소유 package가 추가되어 대응으로 바뀌었습니다. [flavor](extensions/flavors/README.md), [service profile](extensions/serviceprofiles/README.md), [metering label](extensions/metering/labels/README.md), [metering rule](extensions/metering/rules/README.md) README에 요청과 차이를 설명합니다. LBaaS v2 메서드는 서버 API가 폐기되어 구현 대상에서 제외했습니다.

- LBaaS v2 30개 메서드(`*_load_balancer`, `load_balancers`, `*_listener`, `listeners`, `*_pool`, `pools`, `*_pool_member`, `pool_members`, `*_health_monitor`, `health_monitors`): Gophercloud v2.15.0에는 Neutron `network` endpoint의 `/v2.0/lbaas/*` binding이 없습니다. Octavia facade는 `load-balancer` endpoint를 쓰므로 SDK가 소유한 Neutron LBaaS 호출이 따로 필요합니다.
- Flavor 8개 메서드(`*_flavor`, `flavors`, `associate_flavor_with_service_profile`, `disassociate_flavor_from_service_profile`)와 service profile 6개 메서드(`*_service_profile`, `service_profiles`): `/v2.0/flavors`, `/v2.0/flavors/{id}/service_profiles`, `/v2.0/service_profiles` binding이 없어서 SDK가 소유한 호출이 필요합니다.
- Metering 12개 메서드(`*_metering_label`, `metering_labels`, `*_metering_label_rule`, `metering_label_rules`): `/v2.0/metering/metering-labels`와 `/v2.0/metering/metering-label-rules` binding이 없어서 SDK가 소유한 호출이 필요합니다.
- `find_ip`: ID 조회 뒤 `floating_ip_address`나 ID로 전체 목록을 맞춰 보는 fallback helper가 floating IP binding에 없습니다. 400과 403 응답에서 목록 단계로 넘어가는 동작도 없습니다.
- `ips`: query에 없는 선언 Body 속성의 로컬 필터와 알 수 없는 키를 버리는 Python 동작을 floating IP collection이 지원하지 않습니다.
