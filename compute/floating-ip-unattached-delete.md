# 연결되지 않은 Floating IP 정리

`Connection.DeleteUnattachedFloatingIPs`와 `compute.Service.DeleteUnattachedFloatingIPs`는 configured Neutron cloud의 Floating IP 목록을 조회하고, `port_id`가 falsey인 항목을 순서대로 삭제합니다. 서버·port·network builder를 애플리케이션이 구현할 필요 없이 SDK가 목록 조회, 삭제 retry와 결과 확인을 관리합니다.

```go
result, err := conn.DeleteUnattachedFloatingIPs(ctx,
	compute.WithFloatingIPDeleteRetries(1),
	compute.WithFloatingIPDeleteTimeout(time.Minute),
)
```

고정 소스는 다른 프로세스가 IP를 먼저 만들고 나중에 attach하는 두 단계 작업을 진행하는 동안 이 정리를 실행하면 그 사이의 IP를 지울 수 있다고 설명합니다. 이 함수는 조회한 목록의 `port_id`를 기준으로 처리하며, 목록 조회와 삭제를 하나의 transaction으로 묶지 않습니다.

## 선택과 처리 순서

1. configured source가 `FloatingIPNeutron`이고 Network 서비스를 사용할 수 있어야 정리 대상 cloud입니다. Nova·None 또는 정상적인 Network endpoint 부재이면 `Eligible=false`, `Count=0`, `AllDeleted=true`로 끝나며 Compute·목록·삭제를 요청하지 않습니다. endpoint 조회의 source/context/기타 실패가 함께 있으면 오류를 보존합니다.
2. 필터 없는 [일반 ListFloatingIPs](floating-ip-queries.md)로 전체 목록을 먼저 확보합니다. 페이지 조회나 모델 처리에서 오류가 발생하면 앞 페이지에 candidate가 있어도 삭제를 시작하지 않습니다.
3. 조회 Resource view의 `port_id`에 Python JSON truthiness를 적용합니다. missing의 기본 null, null·empty string·false·0·빈 array/object는 candidate이고, nonempty string·nonzero number·nonempty container는 연결된 항목으로 건너뜁니다. 상태·owner·pool·주소 가족을 추가 선택 조건으로 넣지 않습니다.
4. candidate마다 [DeleteFloatingIP](floating-ip-delete.md)의 같은 준비된 옵션을 사용합니다. 목록 순서와 중복 ID를 유지하며, 삭제 사이에 candidate 목록을 새로 조회하지 않습니다.
5. `Deletion.Deleted=false`이면 뒤의 candidate를 계속 처리합니다. 오류가 발생하면 즉시 중단하고 앞선 결과와 현재 항목의 부분 결과를 반환합니다.

`Count`는 오류 없이 `Deletion.Deleted=true`가 된 항목의 수입니다. 접수한 HTTP DELETE 수가 아닙니다. `AllDeleted`는 반복이 끝났고 모든 candidate의 최종 bool이 true일 때만 true입니다. 하위 Delete가 true로 완료된 뒤 상위 source/context guard에서 오류가 나도 이미 완료된 Count는 유지합니다. 전체 성공 판단에는 반환 error와 AllDeleted를 함께 확인합니다.

| 항목별 결과 | Count | AllDeleted | 다음 항목 |
|---|---|---|---|
| candidate 없음 | 0 | true | 완료 |
| 모두 true | candidate 수 | true | 완료 |
| true, false, true | 2 | false | 모두 처리 |
| true, error, 이후 항목 | 1 | false | error 항목에서 중단 |

오류가 없는 `AllDeleted=true`는 Python의 정수 반환 `Count`에 대응하고, `AllDeleted=false`는 Python의 boolean `False`에 대응합니다. Python의 `all([])`가 true이므로 빈 목록과 정리 대상이 아닌 cloud는 정수0입니다. `Count=0`만 보고 정리 실패로 판단하지 않습니다.

## 삭제 옵션과 기본값

이 함수는 standalone Delete의 concrete 옵션을 재사용합니다. 옵션은 목록과 모든 항목을 포함한 전체 작업에서 각각 한 번 준비하며, 항목마다 caller 옵션 함수를 다시 실행하지 않습니다.

| 옵션 | 동작 |
|---|---|
| `WithFloatingIPDeleteRetries(n)` | 기본1; 각 IP의 최초 DELETE에 추가할 횟수. 0은 후속 검증 생략, 음수는 DELETE1회와 검증 |
| `WithFloatingIPDeleteSource(source)` | 이번 작업의 source override; Nova/None이면 정리 gate에서 skip |
| `WithFloatingIPDeleteLocation(location)` | 목록·검증 조회의 owned cloud/project location |
| `WithFloatingIPDeleteStrict(bool)` | Nova fallback 조회 정규화의 strict 모드 |
| `WithFloatingIPDeleteDirectGet(bool)` | 항목 삭제 뒤 UUID형 ID의 검증을 직접 GET으로 선택. 최초 inventory는 일반 List |
| `WithFloatingIPDeleteTimeout(duration)` | 양수인 전체 SDK cap; backend 선택·모든 inventory 페이지·모든 DELETE와 검증에 하나의 budget |
| `WithUnlimitedFloatingIPDeleteTimeout()` | 앞선 SDK cap 제거; caller context 유지 |
| `WithFloatingIPDeleteOptions(compute.FloatingIPDeleteOpts{...})` | 전체 옵션을 owned 복사본으로 교체 |

기본은 `Retries=1`, 추가 SDK deadline 없음, `Strict=false`, `DirectGet=false`입니다. bulk의 `FloatingIPDeleteOpts{}`는 `Retries=0`도 명시하므로 기본 retry를 유지하려면 `Retries: 1`을 넣거나 뒤에 retry setter를 추가합니다. 뒤의 옵션이 앞의 값을 덮어씁니다. nil 옵션·invalid source·invalid 전체 timeout은 HTTP 전에 오류입니다. `PrepareFloatingIPDeleteOptions(ctx, options...)`로 HTTP 없이 같은 옵션을 준비할 수 있습니다.

기본 retry1이면 각 성공한 DELETE 뒤 일반 Get을 수행하고, IP가 없거나 조회 view의 exact `DOWN`이면 해당 항목은 true입니다. 여전히 존재하는 다른 상태이면 최대 한 번 더 DELETE합니다. 정상 DELETE404는 false이며 검증을 수행하지 않습니다. false 항목도 `Items`에 기록하고 다음 candidate를 처리합니다. 실제 부재는 `Deletion.Absent`, 존재하는 DOWN 정책 성공은 `Deletion.Down`으로 확인합니다. retry0의 true는 접수만 확인한 결과입니다.

## inventory backend와 삭제 backend

정리 gate는 configured Neutron과 Network 서비스로 판단합니다. 최초 unfiltered Neutron 목록404가 일반 List 계약에 따라 Nova 목록으로 fallback하면 `Inventory.Backend`는 Nova일 수 있습니다. 뒤의 DELETE는 각 row의 Backend가 아니라 **configured Neutron**으로 선택합니다. 정상 Neutron DELETE404를 Nova DELETE로 재시도하지 않습니다.

`Inventory.FloatingIPs[i].Backend`, `NormalizationSource`, actual `Wire`와 logical `Resource`는 다른 정보를 담을 수 있습니다. Nova wire의 합성 ACTIVE나 `attached`를 candidate 선택에 대신 쓰지 않으며, public Resource의 `port_id`를 봅니다. configured Neutron fallback의 nonstrict view는 null `port_id` alias를 제공할 수 있습니다. strict Nova view에서 그 속성이 없으면 원본의 attribute 접근 실패에 대응하는 오류로 중단하고, 앞서 처리한 항목을 보존합니다.

선택한 Compute microversion이2.36 이상이면 legacy Nova inventory/검증이 필요할 때 unsupported입니다. SDK는 selected 버전을 낮추지 않습니다. [legacy Nova 가이드](server-nova-floating-ip.md)의 호환 endpoint/version 범위를 함께 참고하세요.

## owned 결과와 부분 진행

반환 결과는 다음을 구분합니다.

| 필드 | 의미 |
|---|---|
| `Eligible` | 최초 configured Neutron/Network gate가 정리를 허용했음 |
| `Inventory` | 전체 일반 목록 조회의 rows·Pages·fallback·실패 증거. 조회 오류가 있어도 확보한 페이지가 남을 수 있음 |
| `Items` | 순서대로 처리한 candidate의 기록. 연결된 항목은 포함하지 않음 |
| `Items[i].FloatingIP` | 선택 당시 inventory Resource/Wire의 owned 복사본 |
| `Items[i].ID` | 해당 삭제를 위해 소비한 ID |
| `Items[i].Deletion` | standalone Delete의 bool·attempts·검증·실제 응답·부분 실패 기록 |
| `Items[i].Error` | 이 항목의 ID 처리·삭제·source/context 오류. DELETE 전 ID 오류이면 Deletion은 nil이어도 선택한 Resource/Wire가 남음 |
| `Count` | 개별 Delete가 오류 없이 최종 Deleted=true로 완료한 항목 수. 이후 aggregate guard 오류나 false가 있어도 완료된 수를 유지함 |
| `AllDeleted` | 오류 없이 전체 반복이 끝났고 모든 항목의 bool이 true임. 빈 결과도 true |

candidate의 executable ID는 해당 순서에서 소비합니다. 앞의 성공 뒤 늦은 unsafe/missing ID가 발견되면 그 DELETE 전에 중단하며 앞선 Count와 Items를 유지합니다. truthy port로 건너뛴 row의 ID는 삭제를 위해 소비하지 않습니다.

`Items[i].FloatingIP`의 Resource/Wire를 바꿔도 `Inventory`나 다른 항목을 바꾸지 않도록 소유한 결과를 제공합니다. 개별 Delete의 응답 처리·검증에서 read/Close/source/context 오류가 발생하면 receipt와 가능한 모델을 남기고 Count를 올리지 않습니다. 개별 Delete가 이미 완료된 뒤 aggregate guard가 실패한 경우에는 완료한 Count를 유지하므로, Item.Error와 Deletion.Deleted를 함께 확인합니다. 전체 caller context 또는 SDK cap이 끝나면 이후 항목을 처리하지 않습니다.

`Envelope`는 실제 response의 passive bytes이므로 항상 유효한 JSON은 아닙니다. 전체 result의 `json.Marshal` 대신 아래 예제처럼 scalar summary와 counts를 출력하면 malformed response가 있는 부분 결과도 확인할 수 있습니다.

## 독립 Go 예제

이 프로그램은 선택한 cloud의 연결되지 않은 IP를 실제로 삭제합니다. 기본은 항목별 retry1과 전체 정리1분 cap이며, Connect를 포함한 부모2분 context를 사용합니다. `-timeout 0`은 추가 SDK cap을 제거합니다. `-source nova`/`none`은 정리 gate에서 skip입니다. 필요한 inventory/검증 Nova fallback의 호환 microversion으로 Compute2.35를 명시합니다.

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	retry := flag.Int("retry", 1, "additional deletes per IP; zero skips verification")
	source := flag.String("source", "", "optional neutron, nova, or none override")
	strict := flag.Bool("strict", false, "strict Nova fallback view")
	direct := flag.Bool("direct", false, "direct UUID verification GET")
	timeout := flag.Duration("timeout", time.Minute, "whole cleanup cap; zero removes SDK cap")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(*cloud),
		sdk.WithMicroversion(sdk.Compute, "2.35"),
	)
	if err != nil {
		return err
	}
	opts := []compute.FloatingIPDeleteOption{
		compute.WithFloatingIPDeleteRetries(*retry),
		compute.WithFloatingIPDeleteStrict(*strict),
		compute.WithFloatingIPDeleteDirectGet(*direct),
	}
	if *source != "" {
		opts = append(opts, compute.WithFloatingIPDeleteSource(compute.FloatingIPSource(*source)))
	}
	if *timeout == 0 {
		opts = append(opts, compute.WithUnlimitedFloatingIPDeleteTimeout())
	} else {
		opts = append(opts, compute.WithFloatingIPDeleteTimeout(*timeout))
	}
	result, cleanupErr := conn.DeleteUnattachedFloatingIPs(ctx, opts...)
	var summary map[string]any
	if result != nil {
		summary = map[string]any{
			"eligible": result.Eligible, "count": result.Count,
			"all_deleted": result.AllDeleted, "items": len(result.Items),
		}
		if result.Inventory != nil {
			summary["inventory_backend"] = result.Inventory.Backend
			summary["inventory_pages"] = len(result.Inventory.Pages)
			summary["inventory_rows"] = len(result.Inventory.FloatingIPs)
		}
		items := make([]map[string]any, 0, len(result.Items))
		for _, item := range result.Items {
			entry := map[string]any{"id": item.ID}
			if item.Error != nil {
				entry["error"] = item.Error.Error()
			}
			if deletion := item.Deletion; deletion != nil {
				entry["backend"] = deletion.Backend
				entry["deleted"] = deletion.Deleted
				entry["absent"] = deletion.Absent
				entry["down"] = deletion.Down
				entry["attempts"] = len(deletion.Attempts)
			}
			items = append(items, entry)
		}
		summary["item_results"] = items
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return errors.Join(cleanupErr, encoder.Encode(summary))
}
```

## openstacksdk 비교와 선언 범위

```python
import openstack

conn = openstack.connect(cloud="dev")
deleted = conn.delete_unattached_floating_ips(retry=1)
if deleted is False:
    print("Some candidate deletions returned False")
else:
    print("Completed:", deleted)  # 0 is a successful empty result
```

비교 pin은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`입니다. [정리 함수와 두 단계 create/attach 설명](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1048-L1078), [일반 목록과 fallback](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L511-L566), [항목별 Delete retry/검증](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L977-L1046)을 기준으로 합니다.

Python은 전체 성공이면 정수 count, 하나라도 false이면 boolean False, 오류이면 exception입니다. Go는 `Count`·`AllDeleted`·error와 실제 inventory/항목별 부분 결과를 함께 제공하므로, false 또는 error 뒤에도 앞선 성공 수와 접수 기록을 확인할 수 있습니다. 빈 결과와 Nova/None skip는 `Count=0, AllDeleted=true`로 표현합니다.

이 named operation은 최초 Neutron eligibility, unfiltered complete inventory, Resource `port_id` truthiness와 source-order Delete 반복을 다룹니다. Go의 typed executable ID·owned JSON·source/context guard·전체 cap은 명시적인 API 선택입니다. 일반 목록/검증에서의 strict envelope/status 정책과 missing endpoint의 정확한 판정은 각 Go 가이드의 경계를 따릅니다. mutable Python Resource/session 전체, 다른 CRUD·attach/detach·revision-aware Network Proxy 선언은 개별 SDK 범위로 추적합니다.

