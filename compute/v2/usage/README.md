# 프로젝트 사용량 native 호출

`usage.New(client)`(또는 `service.Usage`)의 generated `SingleTenant(ctx, tenantID, opts, options...)`는 Gophercloud `v2.15.0`의 [usage 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/usage/requests.go)을 바꾸지 않고 `GET os-simple-tenant-usage/{tenantID}`를 보냅니다. 전체 프로젝트를 조회하는 `AllTenants`는 관리자 호출이라 별도 단계에서 다룹니다.

`SingleTenantOpts`의 `start`, `end`, `limit`, `marker`와 `WithSingleTenantQuery` 확장 query를 보냅니다. `start`와 `end`는 `2006-01-02T15:04:05.999999` 형식으로 보내는데 offset을 붙이지 않고 시각 값의 wall clock을 그대로 씁니다. Nova는 이 값을 UTC로 해석하므로 호출자가 먼저 `.UTC()`로 바꿔 넘겨야 합니다.

결과는 페이지마다 `tenant_usage` 객체 하나를 내보내는 stream입니다. 응답의 `tenant_usage_links`에 `rel=next`가 있으면 그 href를 받은 그대로 따라갑니다. 응답에 `tenant_usage`가 없으면 빈 페이지로 보고 아무 값도 내보내지 않지만, 빈 객체 `{}`는 값 하나로 내보냅니다. native pager의 성공 status는 200, 204, 300이며 본문 없는 204는 `io.EOF` 오류 하나로 끝납니다. 목록 오류는 operation 문맥 없이 전달되고 nil 옵션은 HTTP 전에 operation 문맥과 함께 거부됩니다.

`start`, `stop`, 서버별 `started_at`, `ended_at`은 zone 없는 Nova 시각을 UTC로 해석하며 null과 빈 문자열은 zero 시각입니다. Python `conn.compute.get_usage`의 Resource 모델과 cloud `get_compute_usage`의 프로젝트 해석은 이 native 호출에 포함되지 않습니다.
