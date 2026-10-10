# Cinder noauth native client 생성

`noauth` package는 `auth_strategy=noauth`로 실행한 Cinder에 Keystone 없이 접근하는 고정 Gophercloud v2.15.0 [noauth](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/noauth/requests.go) client 생성 함수를 감쌉니다. 서비스 객체에 연결되어 있지 않으므로 `noauth.New(client)`로 직접 만듭니다. 두 메서드 모두 HTTP 요청을 보내지 않습니다.

| 메서드 | 결과 |
|---|---|
| `NewBlockStorageNoAuthV2(ctx, opts, options...)` | `Type`이 `block-storage`인 ServiceClient |
| `NewBlockStorageNoAuthV3(ctx, opts, options...)` | `Type`이 `block-storage`인 ServiceClient |

## 입력과 endpoint

`EndpointOpts.CinderEndpoint`는 필수입니다. 결과 endpoint는 이 주소 끝에 `/`를 맞춘 뒤 ProviderClient token의 project 부분을 붙이고 다시 `/`로 끝냅니다. 예를 들어 token이 `admin:project-1`이고 endpoint가 `http://cinder:8776/v3`이면 `http://cinder:8776/v3/project-1/`입니다. V2와 V3 함수는 같은 동작이며 URL의 버전은 `CinderEndpoint`에 적은 값을 그대로 씁니다.

ProviderClient의 token은 콜론 하나로 나뉜 `user:project` 형식이어야 합니다. native [`noauth.NewClient`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/noauth/requests.go)는 비어 있는 사용자와 tenant 이름을 `admin`으로 채워 이 token을 만듭니다. Keystone 인증으로 받은 token처럼 콜론이 없거나 둘 이상이면 오류이므로, 이 package에 넘기는 client는 `noauth.NewClient`로 만든 ProviderClient를 써야 합니다. 결과 client는 그 ProviderClient를 공유합니다.

endpoint 누락과 token 형식 오류는 native 오류를 operation 문맥 없이 그대로 돌려주고, nil 옵션만 `NewBlockStorageNoAuthV2`/`NewBlockStorageNoAuthV3`과 `noauth` 문맥으로 감쌉니다.
