# Nova quota 초기화 query

고정한 [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)의 `revert_quota_set(project, **query)`는 `Proxy._delete(QuotaSet, ignore_missing=False, params=query)`를 거쳐 `DELETE os-quota-sets/{project}`를 보냅니다. 이때 [`Resource._prepare_request`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)가 `urllib.parse.urlencode(params)`로 query를 붙입니다. Go에서는 `ProjectQuotaScope.Reset`과 `UserQuotaScope.Reset`에 `WithResetQuery`를 주어 같은 요청을 보냅니다.

| Python | Go |
|---|---|
| `revert_quota_set(project)` | `scope.Reset(ctx)` |
| `revert_quota_set(project, force="True")` | `scope.Reset(ctx, quotasets.WithResetQuery("force", "True"))` |
| `revert_quota_set(project, user_id=user, force="True")` | `userScope.Reset(ctx, quotasets.WithResetQuery("force", "True"))` |

## 동작

- `WithResetQuery(key, value)`는 query 키 하나를 더하고, 같은 키를 다시 주면 마지막 값이 남습니다. Python의 `urlencode`는 값에 `str()`을 적용하므로 Go는 서버가 받을 문자열을 그대로 받습니다. 그래서 `True`는 `"True"`로, 목록은 Python 표기 문자열로 넘겨야 같은 요청이 됩니다.
- `user_id`는 `InUser`로 고정한 scope가 맡기 때문에 두 scope 모두 `WithResetQuery("user_id", ...)`를 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다. 빈 키, 제어 문자가 들어간 키, UTF-8이 아닌 값과 nil option도 같은 방식으로 거부합니다.
- query가 없으면 project Reset은 이전과 같이 Gophercloud의 본문 없는 DELETE를 보냅니다. query가 있으면 user Reset과 같은 고정 대상 guard를 거쳐 DELETE를 보내므로, redirect가 경로나 query를 바꾸면 두 번째 요청을 보내지 않습니다.
- Python처럼 기본으로 404를 오류로 돌려주며 이 오류는 `resource.ErrNotFound`이기도 합니다. `WithResetIgnoreMissing(true)`를 주면 404만 `nil, nil`이 됩니다. 그 밖의 오류는 Gophercloud 상태 오류를 `resource.OperationError`(Operation `Reset`)로 감쌉니다.

## Python과 다른 점

- Python은 400 미만의 모든 상태를 받고 Go는 Nova의 202와 204만 받습니다.
- Python은 min(QuotaSet `_max_microversion` 2.56, 서버 최대값)을 보내고 Go는 client microversion을 보냅니다.
- Python은 dict 순서대로 query를 붙이고 Go는 키 이름 순서로 붙입니다.
- Python은 `None`을 돌려주고 Go는 고정한 project와 user, 응답 헤더를 담은 `ResetResponse`를 돌려줍니다.
