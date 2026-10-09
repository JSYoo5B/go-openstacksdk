# 외부 Go 프로젝트에서 사용하기

2026-10-09 최신 Glance owned 전체·store 삭제 및 native Delete: 정확한 문서·판정 revision `cdddf468c1df9375684acf21575b23a45deb3efa`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261009000252-cdddf468c1df`·get/build exit0이며 삭제/설치 main **2개**를 빌드했습니다. 원격 Go source2,109개 SHA256 `26712dbb6f9aca5a28fc41c870421b05aa9dfbf596030dceae3a9e170d793983`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-delete-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는284/3,362이며 공통 Resource/cache/session 목표는 계속 추적합니다. 기본 설치 명령은 이 revision을 사용합니다.

2026-10-09 앞선 Glance owned MemberRecord 조회·추가·수정·삭제: 정확한 문서·판정 revision `4aa85ff43194756fcbe31c6beffe5d1c6707cc2a`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008232408-4aa85ff43194`·get/build exit0이며 멤버 CRUD/설치 main **2개**를 빌드했습니다. 원격 Go source2,101개 SHA256 `0f37f5bd403d6f253c7088d9b339d239d9962045303a06acafff41be973c55d3`가 최종 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-member-record-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는282/3,362이며 공통 Resource/cache/session 목표는 계속 추적합니다. 당시 기본 설치 명령은 이 revision을 사용했습니다.

Go 1.25 이상에서 공개 모듈 `github.com/JSYoo5B/go-openstacksdk`를 사용합니다. 전체 SDK는 개발 중이며, 지원 범위는 [구현 현황](implementation-plan.md)에서 확인합니다. `v0.1.0-alpha.1` tag는 아직 배포하지 않았습니다.

2026-10-09 앞선 owned ImageRecord 수정: 정확한 문서·판정 revision `c068cacecf76dc4ab40254b4324233aa4cfcc86e`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008225520-c068cacecf76`·get/build exit0이며 owned 수정/설치 main **2개**를 빌드했습니다. 원격 Go source2,095개 SHA256 `2cfee5fda3252931e10f6c93da3cbe7a7dbe1e4493879b9f3b6c863dbdfaa346`가 집중/전체 gate·재생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-owned-update-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python/jsonpatch 호출은 실행하지 않았습니다. 완료 집계는279/3,362이며 공통 Resource/cache/session 목표는 계속 추적합니다.

2026-10-09 이름·라이선스 최종 확인: Gophercloud fixture를 포함·결합하는 테스트 파일 18개에 원저작권·원본 범위·로컬 변경 고지를 보완한 revision `1ec3b909bd3588df74f2e84ed38baa96d99cc577`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008222501-1ec3b909bd35`이며 get/build exit0·문서 main **2개** 빌드를 확인했습니다. 원격 Go source2,089개 SHA256 `fdd94cb1618b66c86fd59ee79bde0c27120f68d61c0a0634f47f643431db5659`가 최종 `make check`의 **44개 테스트 package**·race·vet·판정·포맷 검사를 통과한 소스와 같고, 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. 당시 기본 설치 명령은 이 revision을 사용했습니다. 근거는 `/private/tmp/go-openstacksdk-renaming-licensing-final-remote-consumer.json`이며 실제 인증·OpenStack 호출은 실행하지 않았습니다. [이름 변경 안내](renaming.md)와 [라이선스 적용 범위](licensing.md)를 참고하세요.

2026-10-09 앞선 Glance native Update·owned 수정 raw Body 기반: 정확한 문서·판정 revision `862eef0b0d0b0ec08a072ac8937aae5b8916b776`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008220600-862eef0b0d0b`·get/build exit0이며 native 수정/설치 main **2개**를 빌드했습니다. 원격 Go source2,089개 SHA256 `0bbe5d809249c129a9b4355dcdcbb92e7519bd3c2b8f3a77655b0f67df0bbfa3`가 최종 집중/전체 gate·반복 생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-update-raw-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python/jsonpatch 호출은 실행하지 않았습니다. 완료 집계는278/3,362이며 Python owned update/Resource/cache/session 목표는 계속 추적합니다.

2026-10-09 앞선 ImageRecord 태그 추가·삭제: 정확한 문서·판정 revision `72cf240b6a9508cbbbf4404ae882ef9576a306ce`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008214624-72cf240b6a95`·get/build exit0이며 태그/설치 main **2개**를 빌드했습니다. 원격 Go source2,087개 SHA256 `b449631fcefd58d9a3f240ddce214040a3a60bf190d5d20c9a8c497c1166eba4`가 최종 집중/전체 gate·반복 생성의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-tags-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는277/3,362이며 공통 Resource/cache/session 목표는 계속 추적합니다.

2026-10-09 앞선 ImageRecord 상태·삭제 대기: 정확한 문서·판정 revision `180b672ba90518981c21913086af9bb060b7d544`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008210659-180b672ba905`·get/build exit0이며 대기/설치 main **2개**를 빌드했습니다. 원격 Go source2,082개 SHA256 `377666ca351152533e11139af3b2dce6e24c3e46ff728b1b49967cf091746085`가 최종 집중/전체 gate의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-record-waits-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는275/3,362이며 공통 Resource/cache/session 목표는 계속 추적합니다.

2026-10-09 앞선 Glance Image owned 조회·목록·검색: 정확한 문서·판정 revision `16b2873a9ef203b5f07162bba277df84b5a26a16`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008202819-16b2873a9ef2`·get/build exit0이며 Image/설치 main **2개**를 빌드했습니다. 원격 Go source2,074개 SHA256 `3cc9702bc1924492bbb534fb6d94640aa739e90543a5268c488b0249fd4c9b91`가 최종 집중/전체 gate의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-image-records-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는273/3,362이며 공통 mutable Resource·캐시·JMESPath/session 계약은 계속 추적합니다.

2026-10-09 앞선 Glance import·기본 store records: 정확한 문서·판정 revision `95a7c6de03b24b76ffbe49fa6f17efdad553abc6`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008193028-95a7c6de03b2`·get/build exit0이며 discovery/설치 main **2개**를 빌드했습니다. 원격 Go source2,062개 SHA256 `5edf83e69d402ecd8b22677afb70d3d7ca5f8af692534fdd8b6273ea46fb022c`가 최종 집중/전체 gate의 소스와 같고 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-serviceinfo-records-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는270/3,362이며 상세 store는 아직 unresolved입니다.

2026-10-09 앞선 Glance member owned 목록·검색: 정확한 문서·판정 revision `0647ce86adc0e5f0cdbfbe8f3dc7c100c7655433`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008190535-0647ce86adc0`·get/build exit0이며 member/설치 main **2개**를 빌드했습니다. 원격 Go source2,054개 SHA256 `aaf23a88846b8ed282cd9097a9e384df83beca6fa6150fad2c6877fa5f964aad`가 최종 집중/전체 gate의 소스와 같고, 라이선스·고지 **14개 파일**도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-member-records-remote-consumer.json`에 기록했습니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다. 완료 집계는269/3,362입니다.

2026-10-09 이름·라이선스 재검증: revision `478f9b4feff6ea5b71a036a5db0c83aae45e8cbc`을 새 모듈 경로로 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008183000-478f9b4feff6`이며 설치·목록 예제 main 2개의 get/build가 통과했습니다. 보완한 YAML libyaml MIT 원문을 포함해 라이선스·고지 **14개 파일**이 원격 모듈과 로컬에서 byte-identical입니다. 전체 `make check`의 43개 테스트 package·vet·판정·포맷 검사도 통과했습니다. 아래 과거 revision의 13개 파일 검증은 당시의 기록입니다. [이름 변경 안내](renaming.md)와 [라이선스 적용 범위](licensing.md)를 참고하세요.

2026-10-09 앞선 Glance object·namespace owned 목록: 정확한 문서·판정 revision `73f54853810614b582e2e5513a4812e6cd8a97e2`을 별도 외부 module에서 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전 `v0.0.0-20261008181840-73f548538106`·get/build exit0이며 목록/설치 main **2개**를 빌드했습니다. 원격 Go source2,044개 SHA256 `c0ea7d4a6716e99c59fd707e53c15bd770409bb0cc3c7cca1d40386936934d29`가 집중 race·전체43 package gate·반복 생성 drift0의 source와 같고 라이선스/고지13개 파일도 로컬과 byte-identical입니다. `/private/tmp/go-openstacksdk-metadata-lists-remote-consumer.json`에 기록했습니다. 완료 집계는267/3,362이며 실제 인증·OpenStack 호출은 실행하지 않았습니다.

앞선 2026-10-09 Glance resource type association owned 생성·삭제 revision `1f9c1d7a5ede8f0b2c12142e033ce0af5d75e435`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008173702-1f9c1d7a5ede`이며 [생성삭제 main](../image/metadef-resource-type-association-mutations.md)·[목록 main](../image/metadef-resource-types-records.md)·당시 설치 main **3개**의 get/build exit0입니다. 원격 Go source2,033개 SHA256 `0907dfca1b916d7803d4098933f38753ee9f2a83804db323e20baa1e7bb6a452`가 집중45그룹753사례·전체43 package race/vet·최종265 metadata gate·반복 생성 drift0와 같습니다. 라이선스·고지13개 파일도 로컬과 byte-identical이고 완료 집계는265/3,362입니다. `/private/tmp/go-openstacksdk-association-remote-consumer.json`에 기록합니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다.

앞선 2026-10-09 Glance property owned 생성·수정 revision `7b07fe93e433b3694852c685d979d4fbffa4cb35`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008171128-7b07fe93e433`이며 [생성수정 main](../image/metadef-property-record-write.md)·[조회 main](../image/metadef-property-records.md)·[목록 main](../image/metadef-property-record-list.md)·당시 설치 main **4개**의 get/build exit0입니다. 원격 Go source2,029개 SHA256 `71d742e983a57a99a6fdba10b86318c7b7ac87bb1e3e29e298fd0eed07ef03e0`가 집중42그룹477사례·전체43 package race/vet·최종263 metadata gate·반복 생성 drift0와 같습니다. 라이선스·고지13개 파일도 로컬과 byte-identical이고 완료 집계는263/3,362입니다. `/private/tmp/go-openstacksdk-property-write-remote-receipt.json`에 기록합니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다.

앞선 2026-10-09 Glance property owned 목록·삭제 revision `50b4302520ee2d05a1ced1119e40f919e857bc01`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008163807-50b4302520ee`이며 [목록 main](../image/metadef-property-record-list.md)·[조회 main](../image/metadef-property-records.md)·당시 설치 main3개와 [삭제 package example](../image/v2/metadefproperties/README.md#owned-목록과-삭제)1개가 get/build exit0입니다. 원격 Go source2,024개 SHA256 `76ad963e062c1d685aba025872b49ec057391d3748931458ee27080cfc93d21a`가 집중33그룹274사례·전체43 package race/vet·최종261 metadata gate·반복 생성 drift0와 같습니다. 라이선스·고지13개 파일도 로컬과 byte-identical이며 당시 완료 집계는261/3,362입니다. 검증 근거는 `/private/tmp/go-openstacksdk-property-list-remote-receipt.json`입니다. 실제 인증·OpenStack/Python 호출은 실행하지 않았습니다.

앞선 2026-10-09 Glance owned property 조회 revision `1662a6743e2543bbb76ba3f15ed30511d1a4632a`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008160331-1662a6743e25`이며 [owned property main](../image/metadef-property-records.md)·[기존 raw 조회 main](../image/metadef-property.md)·당시 설치 안내 main **3개**의 get/build exit0입니다. 원격 Go source2,018개 SHA256 `bb6e1a27ce2cb84e913fdc425eb7c3d34bc143447c4ff7fa144ef013da581894`가 집중13그룹92사례·전체43 package race/vet·최종 metadata gate·반복 생성 drift0와 같습니다. 라이선스/고지13개 파일은 원격 module cache와 로컬에서 byte-identical입니다. 당시 완료 집계는258/3,362이며 owned property 유한 목록은 위 최신 단위에서 완료했습니다. 실제 인증·OpenStack/Python 호출·alpha tag 배포는 수행하지 않았습니다.

2026-10-09 Glance owned 목록 구현 revision `b4a0f6ab71ec34b698cf12ab670f0516112411dd`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008153156-b4a0f6ab71ec`이며 [resource type·association main](../image/metadef-resource-types-records.md)·[property 조회 main](../image/metadef-property.md)·당시 설치 안내 main **3개**의 get/build exit0입니다. 원격 Go source2,013개 SHA256 `19e37601b49bd38dc5a4a60a24c3a1d43e85747c9e7b96c576f1f3922bd9401a`가 집중 race·전체43 package `make check`·반복 생성 drift0와 같습니다. 루트·제3자 라이선스/고지 13개 파일은 원격 module cache와 로컬 내용이 byte-identical입니다. 완료 집계는257/3,362이며 당시 property 조회의 seed/default/fetch는 미해결이었습니다. 실제 인증·OpenStack/Python 호출·alpha tag 배포는 수행하지 않았습니다.

2026-10-08 이름 변경 후 revision `7cec3a4df3a46838d23f6a4c381034eb6e27b947`을 별도 외부 module에 `GOWORK=off`·replace 없이 설치했습니다. 실제 버전은 `v0.0.0-20261008061758-7cec3a4df3a4`이며 당시 설치 안내 main·Glance schema main·별칭 없는 `openstack` import main **3개**의 get/build exit0입니다. 원격 module cache에 루트 LICENSE·NOTICE·통합 고지와 제3자 원문이 포함되어 있습니다. Go source **2,005개**, SHA256 `7c4805edb03341a4f8aa06b16c49ce0eab5479e916f865feb74ee59e7fede33a`가 로컬 전체 검사와 원격 cache에서 같습니다. 전체 `make check` **43개 실제 test package**, vet·고정 parity·progress·gofmt와 반복 생성 drift0가 PASS했습니다. 이동한 디렉토리의 `make smoke`도 **5개 흐름·9개 기존 그룹**이 PASS했습니다. 실제 인증/OpenStack 호출과 alpha tag 배포는 포함하지 않습니다.

새 프로젝트에서 아래처럼 설치합니다. 기존 Go 프로젝트에서는 `go mod init`을 생략합니다. 이 커밋은 원격 설치·빌드와 라이선스·고지 14개 파일의 포함을 확인한 revision입니다.

```sh
go mod init example.com/mycloud
GOWORK=off go get github.com/JSYoo5B/go-openstacksdk@cdddf468c1df9375684acf21575b23a45deb3efa
```

외부 소비자 검증에는 아래 main을 그대로 사용합니다. 공개 root·서비스·leaf·generic 옵션을 컴파일하며, 인증이나 HTTP 요청을 실행하지 않습니다. `CreateRecordOpts`의 공개 alias를 통해 concrete 속성과 SDK 소유 옵션을 사용할 수 있습니다. builder interface 구현은 필요하지 않습니다.

```go
package main

import (
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    _ = (*sdk.Connection).DeactivateImageRecord
    _ = (*sdk.Connection).ReactivateImageRecord
    _ = (*image.Service).DeactivateImageRecord
    _ = (*image.Service).ReactivateImageRecord
    _ = image.ImageRecordActionRequest{ID: "literal-image-id"}
    _ = image.WithImageRecordActionOpts(image.ImageRecordActionOpts{})
    _ = image.WithImageRecordActionHeader("X-Trace", "record-action")
    _ = (*sdk.Connection).DeleteImageRecord
    _ = (*image.Service).DeleteImageRecord
    _ = (*images.API).Delete
    _ = image.ImageRecordDeleteRequest{ID: "literal-image-id"}
    _ = image.WithImageRecordDeleteStoreID("store-id")
    _ = image.WithImageRecordDeleteIgnoreMissing(false)
    _ = (*sdk.Connection).GetImageMemberRecord
    _ = (*sdk.Connection).AddImageMemberRecord
    _ = (*sdk.Connection).UpdateImageMemberRecord
    _ = (*sdk.Connection).RemoveImageMemberRecord
    _ = (*image.Service).GetImageMemberRecord
    _ = (*image.Service).AddImageMemberRecord
    _ = (*image.Service).UpdateImageMemberRecord
    _ = (*image.Service).RemoveImageMemberRecord
    _ = image.ImageMemberRecordRequest{ID: "literal-member-id"}
    _ = image.ImageMemberRecordWriteOpts{Attributes: []resource.ListOption{resource.WithFilter("status", "accepted")}}
    _ = image.WithImageMemberRecordMemberID("recipient-project-id")
    _ = image.WithImageMemberRecordStatus("accepted")
    _ = (*sdk.Connection).UpdateImageRecord
    _ = (*image.Service).UpdateImageRecord
    _ = image.ImageRecordUpdateRequest{ID: "literal-image-id", Attributes: map[string]any{"name": "new"}}
    _ = (*image.Service).WaitForImageRecordStatus
    _ = (*image.Service).WaitForImageRecordDelete
    _ = (*image.Service).AddImageRecordTag
    _ = (*image.Service).RemoveImageRecordTag
    _ = image.ImageRecordTagRequest{ID: "literal-image-id"}
    _ = image.WithImageRecordWaitUnlimited()
    _ = image.WithImageRecordWaitAttribute("status")
    _ = (*image.Service).GetImageRecord
    _ = (*image.Service).ListImageRecords
    _ = (*image.Service).AllImageRecords
    _ = (*image.Service).FindImageRecord
    _ = image.ImageRecordRequest{ID: "image-id"}
    _ = image.WithImageRecordAttribute("properties", map[string]any{"owner_hint": "project"})
    _ = image.WithImageRecordListLimit(0)
    _ = image.WithImageRecordListFilter("is_hidden", true)
    _ = image.WithFindImageRecordIgnoreMissing(false)
    _ = (*metadefnamespaces.API).ListRecords
    _ = (*metadefnamespaces.API).AllRecords
    _ = metadefnamespaces.WithRecordListLimit(0)
    _ = metadefnamespaces.WithRecordListFilter("is_protected", false)
    _ = (*metadefobjects.NamespaceScope).ListRecords
    _ = (*metadefobjects.NamespaceScope).AllRecords
    _ = metadefobjects.WithRecordListFilter("properties", map[string]any{"type": "string"})
    _ = sdk.Connect
    _ = (*metadefresourcetypes.NamespaceScope).CreateRecord
    _ = (*metadefresourcetypes.NamespaceScope).DeleteRecord
    _ = metadefresourcetypes.WithRecordCreateAttributes(map[string]any{"name": "OS::Nova::Server"})
    _ = metadefresourcetypes.RecordRequest{ID: "OS::Nova::Server"}
    _ = (*sdk.Connection).AttachVolume
    _ = (*metadefproperties.NamespaceScope).CreateRecord
    _ = (*metadefproperties.NamespaceScope).UpdateRecord
    _ = metadefproperties.WithRecordCreateAttributes(map[string]any{"name": "example", "type": "string"})
    _ = metadefproperties.WithRecordUpdateAttribute("min_length", 0)
    _ = (*metadefproperties.NamespaceScope).ListRecords
    _ = (*metadefproperties.NamespaceScope).DeleteRecord
    _ = (*metadefproperties.NamespaceScope).DeleteAllRecords
    _ = metadefproperties.WithRecordListFilter("max_items", 3)
    _ = metadefproperties.WithRecordListMaxItems(20)
    var _ *compute.Service
    var _ *network.FloatingIPs
    _ = resource.ID("server-id")
    _ = (*containers.API).CreateRecord
    _ = (*orders.API).CreateRecord
    _ = (*secrets.API).CreateRecord
    _ = containers.WithCreateRecordAttributes(map[string]any{"type": "generic"})
    _ = orders.WithCreateRecordAttribute("meta", map[string]any{"algorithm": "aes"})
    _ = secrets.WithCreateRecordAttribute("name", "example")
    var option containers.CreateRecordOption = func(config *request.Config[containers.CreateRecordOpts]) error {
        config.Options.Attributes = map[string]any{"name": "example"}
        return nil
    }
    _ = option
}
```

main을 `main.go`로 저장한 뒤 빌드합니다.

```sh
GOWORK=off go build -mod=readonly ./...
```

실제 호출 예제와 Python 비교는 [전체 README](../README.md), [Compute](../compute/README.md), [Network](../network/README.md), [Barbican 생성](../keymanager/v1/metadata-create.md)에 있습니다. HTTP 동작은 기존 Gophercloud fixture 기반 계약 테스트와 `make smoke`로 검증합니다.

## 이름 변경 전 검증 기록

이 절의 revision·버전·source SHA는 당시 `github.com/JSYoo5B/gophercloudsdk` 모듈과 예제로 검증했습니다. 현재 예제의 새 import 경로를 이 과거 revision에 조합하면 안 됩니다.

2026-10-08 최신 schema Source revision `3729f70c2b150d19a719d6c8abf2a1eae8a4f666`을 새 외부 module에 replace 없이 설치해 당시 설치 main과 [Glance schema main](../image/schema-records.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·버전 `v0.0.0-20261008051710-3729f70c2b15`입니다. 원격 Go source2,005개 SHA256 `e4e94fb52a3729a7da9003665b580fb934a6d5d6b7bc583fb5ee018908e4cb0c`가 최종 전체43 package gate와 같으며 SDK와 소비자에 replace가 없습니다. 이번 단위의 완료 집계는255/3,362입니다. 인증·OpenStack/Python 호출은 실행하지 않았습니다.

2026-10-08에 source revision `2488a769b27401fab85115e989cf511c01a86cfa`을 별도 외부 module에 replace 없이 설치하고 위 설치 main과 [Availability Zone main](../compute/availability-zones.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008045149-2488a769b274`이며 module-cache Go source2,001개 SHA256 `915a3a9d4f67131b4aaba489232d613643347134ff634b51fc43d670bb9d294d`가 집중41그룹(기존38재사용)·전체43 package gate와 같습니다. 새3행12계약 중 완료2개·Proxy부분1개 판정으로 완료243/3,362이며 catalog/source pins와 기존547 reviews를 보존했습니다. 같은 최종 Go의 JSON/prose에는 전체 Go gate를 재실행하지 않습니다. 인증/OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `f22a803d095233c151a4d47206b49e8a74d8b92d`을 별도 외부 module에 replace 없이 설치하고 위 설치 main과 [Cloud Flavor main](../compute/flavor-cloud.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008042836-f22a803d0952`이며 module-cache Go source1,998개 SHA256 `c6b80797734918b28faaca111130f30d9dd2a154a8f16f91d0c4ca3eede1dc08`가 집중60그룹(기존56재사용)·전체43 package gate와 같습니다. 새 Source3행9계약 판정으로 완료241/3,362이며 catalog/source pins와 기존544 reviews를 보존했습니다. 같은 최종 Go의 JSON/prose에는 전체 Go gate를 재실행하지 않습니다. 인증/OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `037bd1535f699c20d09aedef6dea11e963f022a4`을 별도 외부 module에 replace 없이 설치하고 위 설치 main과 [Flavor 목록·검색 main](../compute/flavor-records.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008035547-037bd1535f69`이며 module-cache Go source1,996개 SHA256 `cc3404ffe583ef697d0eb7040d71e3d7bf177b738f150ae529bbbce1493d2350`가 집중53그룹(기존41재사용)·전체43 package gate와 같습니다. Source2개 판정으로 현재 완료 수는238/3,362이고 catalog/source pins·다른542 reviews·기존 Find8계약을 보존했습니다. 같은 최종 Go의 JSON/prose에는 전체 Go gate를 재실행하지 않습니다. 인증/OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `3d446f604fb57fefe5be31595679c24ff9ffd796`을 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Flavor property/native main](../compute/flavor-property-and-native.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008031535-3d446f604fb5`이며 module-cache Go source1,983개 SHA256 `bccbeed5b2b97484735d5d6f5d90532313c2f6d1379c3f4c6d844ed1f7f9ce9d`가 집중35그룹(기존28재사용)·전체42 package gate와 같습니다. 새5행15계약으로 현재 완료 수는236/3,362이며 기존538 reviews/catalog/source pins를 보존했습니다. 같은 최종 Go의 JSON/prose 갱신에는 전체 Go gate를 재실행하지 않습니다. 인증·OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `df3afaea603f3eb627f9948be30948a87ec29991`을 새 외부 module에 replace 없이 설치하고 위 설치 main, [Flavor extra-specs main](../compute/flavor-extra-specs.md), [Cloud keypair main](../compute/keypairs-cloud.md)을 빌드했습니다. GOWORK=off·get/build exit0·실제 버전 `v0.0.0-20261008024734-df3afaea603f`이며 module-cache Go source1,979개 SHA256 `47121553d380a5b4b7ed6e34b1dcbfc1fa5a7144a92910e43773c034959d1774`가 집중46그룹·전체42 package gate와 같습니다. 새7행24계약으로 현재 완료 수는231/3,362이며 기존531 reviews/catalog/source pins를 보존했습니다. 같은 최종 Go의 JSON/prose 갱신에는 전체 Go gate를 재실행하지 않습니다. 인증·OpenStack/Python 실행·alpha tag 배포는 포함하지 않습니다.

2026-10-08에 source revision `2490e4813a564b5d8163f15091eff5448817398f`을 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Keypair 목록·검색 main](../compute/keypairs-list-find.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·실제 버전 `v0.0.0-20261008020804-2490e4813a56`이며 module-cache Go source1,973개 SHA256 `bbb1c68d0a0a67e5a50a531247d85c0f13b71ca2b8ebf3d530ffbb3ffb3ef9b3`가 집중98그룹·전체42 package gate와 같습니다. 새3행12계약으로 현재 완료 수는224/3,362이며 기존528 reviews/catalog/source pins를 보존했습니다. 소비자-only local replace 예제 빌드와 이 원격 설치는 별개로 확인했습니다. 인증·실제 OpenStack/Python 호출·alpha tag 배포는 수행하지 않았습니다.

2026-10-08에 source revision `40dbb0eaa8f97a19eb0c7aca417bc4251db955e9`을 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Console token 조회 main](../compute/console-auth-token.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·실제 버전 `v0.0.0-20261008013006-40dbb0eaa8f9`이며 module-cache Go source1,968개 SHA256 `1b22fdecda0348eb93f1fc99c17b4a6caf4e79b1808eff1406d66dbadc901e31`가 집중12그룹·전체42 package gate와 같습니다. catalog/source pins/기존527 reviews를 보존한 새1행5계약으로 현재 API 완료 수는221/3,362입니다. SDK와 원격 소비자에 replace가 없고 인증·OpenStack/Python 호출은 실행하지 않았습니다.

2026-10-08에 최종 source revision `466f28d7f3f4a6f7f44d75cf9a4e4c1a96bea2f2`를 새 외부 module에 replace 없이 설치하고 위 설치 main과 [Console 자동 선택 main](../compute/console-selection.md)을 빌드했습니다. `GOWORK=off`, get/build exit0·실제 버전 `v0.0.0-20261008010620-466f28d7f3f4`이며 module-cache Go source1,965개 SHA256 `7543a1f9eaea5ab349f95f4bc6a9d3027137111e26ffc70acadc00d78f2bbc99`가 최종 전체42 package gate와 같습니다. 새 composition1개를 검토한 현재 API 완료 수는220/3,362입니다. SDK와 소비자에 replace가 없고 인증·OpenStack/Python 호출은 실행하지 않았습니다.

2026-10-08에 push한 `0851b89cb637`을 새 외부 module에서 replace 없이 설치하고, 위 설치 main과 [Keypair·legacy console main](../compute/keypairs-console.md)을 함께 빌드했습니다. `GOWORK=off`, `go get`·`go build -mod=readonly` exit0이며 실제 버전은 `v0.0.0-20261008003103-0851b89cb637`입니다. module-cache Go source1,960개 SHA256 `65af6c636b84bb427a832758738be65d823a78ac0e00dc4efaa31632b470d1b2`가 로컬 전체41 package gate의 최종 소스와 같습니다. SDK와 소비자에 replace가 없고 인증·OpenStack 호출은 실행하지 않았습니다. 당시 같은 Go 소스에서 user 작업2개와 native Create1개를 개별 판정하여 API 완료 수는219개였습니다.

앞선 `1d159655cb11`에서도 설치 main과 [Compute 작업 main](../compute/user-actions.md)의 원격 빌드가 PASS했습니다. 당시 버전은 `v0.0.0-20261008001048-1d159655cb11`이며 로컬 전체 gate와 원격 Go SHA가 같았습니다.

앞선 `f32680a5cb51`에서도 설치 main과 [Compute 조회 main](../compute/user-read-apis.md)의 원격 빌드가 PASS했습니다. 당시 버전은 `v0.0.0-20261007234227-f32680a5cb51`이며 로컬 전체 gate와 원격 Go SHA가 같았습니다.

앞선 `36e16d08dc5f`에서도 설치 main과 [Keystone native/owned 두 main](../identity/v3/users/memberships.md)의 원격 빌드가 PASS했습니다. 당시 버전은 `v0.0.0-20261007231141-36e16d08dc5f`이며 로컬 전체 gate와 원격 Go SHA가 같았습니다.

아래는 namespace 도입 당시의 검증 이력입니다. checkout 밖의 독립 소비자 2개로 위 main을 빌드했습니다. local-replace 검증과 원격 설치 검증을 구분하며, 원격 소비자는 `GOWORK=off`이고 replace가 없습니다. 실제 설치 버전은 `v0.0.0-20261007215549-3f0240534253`이며 module cache에서 빌드했습니다. source SHA256은 `137a479b9613464c549cfe734a66b91a32a99bcb201ea886d3a068a183128d9d`입니다. 이 설치 단위 검증 당시 API 완료 수는188개였으며, 해당 직전 판정 구간에서196개까지 증가했습니다. 최신 개수는 [구현 현황](implementation-plan.md)에서 확인합니다.

| 검사 | 상태 |
|---|---|
| 고정 소스·기존 판정 보존 | PASS:namespace 단위의3,362개 fingerprint·513개 판정·당시188개 완료 유지 |
| SDK 재생성 일치 | PASS:생성기 race test·재생성 drift0 |
| 기존 전체 계약 검사·핵심 smoke | PASS:40개 test package·5흐름/9그룹; Go source SHA 보존 |
| 외부 module의 local-replace 빌드 | PASS:위 main 그대로 별도 module에서 빌드; consumer에만 replace |
| push된 정확한 커밋의 replace 없는 설치·빌드 | PASS:`3f0240534253`; `go get`·`go build -mod=readonly` exit0, Replace 없음 |
| Keystone 두 owned 목록 추가 후 외부 설치·빌드 | PASS:`36e16d08dc5f`, no replace; 설치 main+native/owned membership main3개와 원격/로컬 Go SHA 일치 |
| Compute 조회 추가 후 외부 설치·빌드 | PASS:`f32680a5cb51`, no replace; 설치 main+조회 main2개와 원격/로컬 Go SHA 일치 |
| Compute user 작업 추가 후 외부 설치·빌드 | PASS:`1d159655cb11`, no replace; 설치 main+action main2개와 원격/로컬 Go SHA 일치 |
| Keypair 생성·legacy console 추가 후 외부 설치·빌드 | PASS:`0851b89cb637`, no replace; 설치 main+keypair/console main2개와 원격/로컬 Go SHA 일치 |
| Console 자동 선택 추가 후 외부 설치·빌드 | PASS:`466f28d7f3f4`, no replace; 설치 main+자동 선택 main2개와 최종 원격/로컬 Go SHA 일치 |
| Console token 조회 추가 후 외부 설치·빌드 | PASS:`40dbb0eaa8f9`, no replace; 설치 main+조회 main2개와 최종 원격/로컬 Go SHA 일치 |
