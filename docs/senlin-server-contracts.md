# Senlin 16.0.0 server contract audit

이 문서는 2026-10-01에 primary server source를 browser로 읽어 확인한 계약을 기록합니다.
대상은 Senlin `16.0.0`, commit `cef004c05bbc209239e749270ad090063971b4d7`입니다.
공식 API reference와 release 구현이 다른 부분을 구분하며, 실제 OpenStack cloud에서
요청하거나 작업 완료를 확인한 결과는 아닙니다. SDK의 로컬 HTTP fixture도 응답 처리와
전송 계약을 검증할 뿐 배포 환경의 동작을 입증하지 않습니다.

인용은 아래 release 파일과 확인한 함수·route 이름을 기준으로 합니다. 별도로 보존하지
않은 source 줄번호를 추정해서 붙이지 않습니다.

## Success codes and action evidence

| Operation | Published API reference | Senlin 16.0.0 source | SDK consequence |
|---|---|---|---|
| POST `/clusters` | 201 | router의 `cluster_create`가 `success=202`를 지정 | Create는 201과 202를 허용하고 실제 status를 보존 |
| GET `/clusters/{cluster}/attrs/{path}` | 202 | `cluster_collect`에 success override가 없어 WSGI 기본 200 사용 | Collect의 공개 202와 release 200을 별도로 다룸 |
| GET `/clusters/{cluster}/policies` 및 단건 policy binding GET | 200 | `cluster_policy_index` / `cluster_policy_show`에 success override 없음 | 동기 200 조회 계약 |

근거는 [공식 API reference](https://docs.openstack.org/api-ref/clustering/)와
[16.0.0 router.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)의
`cluster_create`, `cluster_collect`, `cluster_policy_index`, `cluster_policy_show` route입니다.
[16.0.0 wsgi.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/common/wsgi.py)의
`Resource.__call__`은 action 인수에서 success 값을 꺼내고 기본 HTTP 200 Response를 만든 뒤,
success가 명시된 경우에만 status를 바꿉니다. 따라서 controller 반환값만으로 202를
가정하지 않고 router와 WSGI를 함께 확인해야 합니다.

Cluster Create의 202는 action Location을 요구하며 `Cluster.Operation`에 action ID와
원래 Location·body·header·status를 보존합니다. 201에서는 Location이 없으면 Operation이
nil이고, 있으면 같은 검증을 거쳐 action 참조를 보존합니다. action 참조가 있어도 SDK가
후속 action GET이나 완료 대기를 자동 수행하지 않습니다. 잘못된 accepted 응답은 원문을
가진 `resource.ResponseError`이며 mutation을 자동 재전송하지 않습니다.

로컬 검증 근거는
[Create status fixtures](../api/clustering_cluster_create_status_test.go)의
`TestClusteringClusterCreatePublishedAndControllerSuccessCodes`,
`TestClusteringClusterCreateAcceptedRequiresValidActionLocation`,
`TestClusteringClusterCreateUndeclaredStatusAndMalformedAcceptedBody`입니다.
201/202의 typed/raw 응답, 202의 필수 Location, accepted 오류 증거와 단일 전송,
허용하지 않은 200의 native HTTP 오류를 검사합니다.

## Attribute collection

[16.0.0 clusters.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py)의
`ClusterController.collect`는 microversion 1.2 이상에서 attribute path를 받습니다.
path를 정리한 뒤 비어 있거나 None이면 HTTP 400으로 거부합니다.
router의 `cluster_collect`와 WSGI의 기본 status를 함께 보면 이 release의 정상 응답은
HTTP 200입니다. [16.0.0 conductor/service.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)의
`cluster_collect`가 수집 경로를 처리합니다. 이 source 증거는 공개 API reference의 202와
차이가 있다는 뜻이며, 모든 Senlin release나 vendor deployment에 같은 status를 일반화하지 않습니다.

## Cluster policy list queries and fixed scope

[16.0.0 cluster_policies.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py)의
`ClusterPolicyController.index` query whitelist는 `enabled`, `policy_name`, `policy_type`,
`sort` 네 필드입니다. 초기 limit/marker나 임의 vendor query를 stock endpoint의 기능으로
주장하지 않습니다. Binding ID와 policy ID를 혼동하여 marker를 생성하지 않습니다.
배포 환경이 명시적으로 제공한 continuation link를 처리하는 Go 기능은 별도 extension입니다.

`enabled=false`는 생략과 구별합니다. scope의 cluster identity, binding의 `id`,
연결된 정책의 `policy_id`, 응답의 `cluster_id`는 서로 다른 역할입니다.
SDK의 고정 부모와 policy route는 반환 모델을 수정하거나 다른 response 필드를 읽었다는
이유로 바뀌어서는 안 됩니다. 세부 Go 범위·query 정책은
[cluster policy bindings README](../clustering/v1/clusterpolicies/README.md)에 기록합니다.

## Metadata mutation and completion

16.0.0 router에는 cluster/node metadata 전용 subresource route가 등록되어 있지 않습니다.
이 release의 router 증거만으로 `/clusters/{id}/metadata` 같은 경로의 CRUD를 만들거나,
Python Resource가 노출한 경로가 실제 서버에서 지원된다고 판정하지 않습니다.
다른 release 또는 vendor route의 존재 여부는 별도 검증 범위입니다.

Cluster의 metadata 변경은 cluster update action의 일부입니다.
[16.0.0 conductor/service.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)의
`cluster_update`는 update action을 queue에 넣는 비동기 경로입니다. Cluster Update의
HTTP 202 수락 계약은 작업의 저장 완료를 뜻하지 않습니다.
[16.0.0 engine/actions/cluster_action.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/engine/actions/cluster_action.py)의
`ClusterAction.do_update`는 inputs에서 metadata를 읽고 None이 아닌 경우 entity.metadata를
그 값으로 대입하여 저장합니다. 객체 key별 병합이 아니라 metadata 전체 객체의 교체입니다.
명시 빈 객체는 replacement 입력이고, 이 코드에서 None은 같은 교체 경로를 실행하지 않습니다.

따라서 PATCH의 JSON null을 metadata 삭제 성공으로 설명하거나, 202 수락을 저장 완료로
취급하지 않습니다. 기존 keys를 유지해야 하는 호출자는 원하는 전체 객체를 명시적으로
구성해야 합니다. 읽기와 쓰기 사이의 경쟁을 없애는 서버 조건부 변경 계약은 이 audit에서
입증하지 않았습니다. 실제 저장 결과와 action 완료는 운영 환경에서 별도로 확인해야 합니다.
