# Swift native 계정·container·object 호출

`accounts.New(client)`, `containers.New(client)`, `objects.New(client)`(또는 `service.Accounts`·`service.Containers`·`service.Objects`)의 generated 메서드는 Gophercloud `v2.15.0`의 [accounts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/accounts/requests.go), [containers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/containers/requests.go), [objects](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError` 문맥만 더합니다. 결과는 대부분 응답 header를 decode한 값이고, SDK가 소유한 metadata·temp URL key·생성/삭제 흐름은 [ObjectStorage v1 사용법](README.md)을 참고합니다. 모든 요청에는 Gophercloud가 `Accept: application/json`을 붙입니다.

## 계정

| 메서드 | 요청 | 결과 | 성공 status |
|---|---|---|---|
| `accounts.Get(ctx, options...)` | `HEAD {endpoint}` | `GetHeader` | 204 |
| `accounts.Update(ctx, opts, options...)` | `POST {endpoint}` | `UpdateHeader` | 201, 202, 204 |

두 호출은 `ResourceBase`가 아니라 client의 정규화된 `Endpoint`(끝에 `/`)로 보냅니다. `GetOpts.Newest`가 true면 `X-Newest: true`를 보내고 false는 생략합니다. `UpdateOpts.Metadata`는 `X-Account-Meta-{key}`, `RemoveMetadata`는 `X-Remove-Account-Meta-{key}: remove`, `ContentType`·`DetectContentType` pointer는 빈 값과 false도 보냅니다. `With...Header`의 확장 header는 typed 입력 header와 겹치거나 형식이 잘못되면 HTTP 전에 거부됩니다.

`GetHeader`는 사용량·container 수·object 수를 숫자 문자열 header에서 읽고, quota header가 없으면 `QuotaBytes`는 nil입니다. 숫자가 아닌 카운터나 RFC1123이 아닌 `Date`는 decode 오류입니다.

## container

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, name, opts, options...)` | `PUT {container}` | 201, 202, 204 |
| `Get(ctx, name, options...)` | `HEAD {container}` | 200, 204 |
| `Update(ctx, name, opts, options...)` | `POST {container}` | 201, 202, 204 |
| `Delete(ctx, name)` | `DELETE {container}` | 202, 204 |
| `List(ctx, options...)` | `GET {endpoint}` | marker paging |
| `BulkDelete(ctx, names)` | `POST {endpoint}?bulk-delete=true` | 200 |

container 이름은 `url.PathEscape`로 escape해 `ResourceBase` 아래 경로 segment가 되며, 비어 있거나 `/`를 포함하면 HTTP 전에 오류입니다. `CreateOpts`의 bool header(`DetectContentType`·`VersionsEnabled`)는 true일 때만 보내고, `UpdateOpts`의 pointer header는 빈 문자열과 false도 보냅니다. metadata는 `X-Container-Meta-{key}`, 삭제는 `X-Remove-Container-Meta-{key}: remove`입니다.

`Get`의 `GetHeader.Read`·`Write`는 ACL header를 쉼표로 나눈 값이라 header가 없으면 `[""]`(빈 문자열 하나)입니다. `X-Versions-Enabled`는 Go `strconv.ParseBool`로 읽어 `True`는 true이고 해석할 수 없는 값은 decode 오류입니다.

`List`는 계정 `Endpoint`에 `Accept`·`Content-Type: application/json` header를 붙여 JSON 목록을 받습니다. Swift marker paging이라 마지막 이름을 `marker` query로 다시 요청하고 빈 페이지에서 멈춥니다. 본문 없는 204도 빈 목록으로 끝납니다. `BulkDelete`는 이름을 path escape해 한 줄씩 보낸 text 본문을 계정 endpoint에 POST하고, 응답의 삭제·미존재 수와 오류 목록을 돌려줍니다. 목록 안 이름 하나라도 잘못되면 요청 전체를 보내지 않습니다.

## object

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, container, object, opts, options...)` | `PUT {container}/{object}` | 201, 202 |
| `Get(ctx, container, object, options...)` | `HEAD {container}/{object}` | 200, 204 |
| `Download(ctx, container, object, options...)` | `GET {container}/{object}` | 200, 206, 304 |
| `Update(ctx, container, object, opts, options...)` | `POST {container}/{object}` | 201, 202 |
| `Delete(ctx, container, object, options...)` | `DELETE {container}/{object}` | 202, 204 |
| `Copy(ctx, container, object, opts, options...)` | `COPY {container}/{object}` | 201 |
| `List(ctx, container, options...)` | `GET {container}` | marker paging |
| `BulkDelete(ctx, container, objects)` | `POST {endpoint}?bulk-delete=true` | 200 |
| `CreateTempURL(ctx, container, object, opts)` | 서명 URL 계산, key가 없으면 HEAD | 없음 |

container와 object 이름은 각각 `url.PathEscape`로 escape하므로 object 이름의 `/`도 `%2F`가 됩니다. 빈 이름과 `/`가 든 container 이름은 HTTP 전에 오류입니다.

`Create`는 `ETag`와 `NoETag`가 모두 비어 있으면 본문 전체를 읽어 MD5를 계산해 `ETag` header로 보냅니다. 명시한 `ETag`는 그대로 보내며, `NoETag`는 소문자 key를 지우는 native 구현이라 명시 `ETag`를 지우지 못합니다. `Content`가 nil이고 ETag 계산이 필요하면 native 구현이 nil reader를 읽다가 **panic**합니다. SDK가 이 경로를 막지 않으므로 이 호출의 판정은 미완료이며, 빈 객체는 빈 reader를 주거나 SDK의 `CreateObject`를 사용합니다.

`Get`과 `Download`의 header 결과는 `X-Static-Large-Object`를 bool로, `X-Delete-At`을 Unix 시각으로 읽습니다. `Download`의 조건부 시각은 RFC1123으로 보내고, 반환값 `*request.Download`는 응답 body를 소유하므로 호출자가 닫아야 합니다. `Update`의 pointer header는 빈 값과 false도 보냅니다.

`Copy`는 `Destination`이 필수이고 `/`로 시작해 container와 object를 구분하는 slash가 있어야 합니다. 앞의 두 slash로만 나눈 각 부분을 path escape하므로 대상 object 이름의 slash는 `%2F`가 됩니다. `List`는 container 경로에 JSON header를 붙여 marker paging으로 읽고, 이름 없는 `subdir` 행은 그 값을 다음 marker로 씁니다. `BulkDelete`는 `{container}/{object}`를 각각 escape한 줄을 계정 endpoint에 보냅니다.

`CreateTempURL`은 `TempURLKey`가 있으면 HTTP 없이 서명합니다. 없으면 container HEAD, 그다음 계정 HEAD에서 key를 찾고, 둘 다 없으면 `ErrTempURLKeyNotFound`입니다. 서명 대상은 escape하지 않은 경로에서 `Split`(기본 `/v1/`) 뒤를 잘라 만든 `{method}\n{expires}\n{path}`이고 반환 URL은 escape된 경로에 `temp_url_sig`·`temp_url_expires`를 붙입니다. digest는 기본 SHA1과 `sha256`·`sha512`만 받습니다. 이 메서드는 옵션 오류만 `OperationError`로 감싸고 key·digest·split 오류는 native 오류 그대로 돌려줍니다.
