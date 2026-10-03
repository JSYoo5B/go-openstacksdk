# Swift 객체 생성과 stale 확인

`service.Objects.CreateObject`는 explicit bytes, local file 또는 borrowed Reader 하나를 받아 객체를 만듭니다. bytes는 한 번의 PUT이며, file/Reader는 같은 bytes를 소유한 spool로 준비한 뒤 capability와 stale 상태를 확인하고 ordinary·SLO·DLO 업로드를 선택합니다. `IsObjectStale`는 이 비교만 별도로 수행합니다. container는 먼저 존재해야 하며 두 API는 container를 자동으로 만들지 않습니다.

| 입력·작업 | pinned openstacksdk | Go |
|---|---|---|
| present data, 빈 data 포함 | 단일 Resource PUT, mutable Object 반환 | `CreateObjectInput.Data`, 단일 PUT의 실제 증거 반환 |
| explicit filename | hash → capability → stale HEAD → skip 또는 업로드, `None` 반환 | owned spool → capability → stale HEAD → skip 또는 ordinary/SLO/DLO, 각 phase 보존 |
| borrowed Reader | 해당 public parameter 없음 | 현재 cursor부터 한 번 spool하고 file workflow 사용; caller Close·Seek 없음 |
| standalone stale | `is_object_stale(container, name, filename, ...)`, bool 반환 | `IsObjectStale`, nullable decision·local/remote hashes·실제 HEAD 반환 |

다음 예제는 이미 존재하는 `backups` container를 사용합니다. 하나의 인증된 service에서 empty bytes, explicit file, Reader와 별도 stale API를 보여줍니다. 보통 file upload 앞에 별도 stale 호출은 필요하지 않습니다. `CreateObject`가 fresh HEAD를 다시 수행합니다.

```go
package main

import (
    "bytes"
    "context"
    "errors"
    "fmt"

    sdk "gophercloudsdk"
    "gophercloudsdk/objectstorage/v1/objects"
    "gophercloudsdk/resource"
)

func showPhase(name string, phase *objects.ObjectCreatePhaseResult) {
    if phase == nil {
        return
    }
    for _, attempt := range phase.Attempts {
        if attempt.Response != nil {
            fmt.Printf("%s logical=%d physical=%d status=%d response-bytes=%d error=%v\n",
                name, attempt.LogicalAttempt, attempt.PhysicalAttempt,
                attempt.Response.StatusCode, len(attempt.Response.Body), attempt.Error)
        } else {
            fmt.Printf("%s logical=%d physical=%d transport-error=%v\n",
                name, attempt.LogicalAttempt, attempt.PhysicalAttempt, attempt.Error)
        }
    }
}

func showCreate(result *objects.CreateObjectResult) {
    if result == nil {
        return
    }
    fmt.Printf("source=%s mode=%s size=%d skipped=%t prefix=%q manifest-ambiguous=%t\n",
        result.Source, result.Mode, result.Size, result.Skipped,
        result.SegmentPrefix, result.ManifestAmbiguous)
    if result.Capabilities != nil {
        fmt.Printf("segment-size=%d fallback=%t\n",
            result.Capabilities.Size, result.Capabilities.UsedFallback)
    }
    if result.Discovery != nil {
        fmt.Println("stale HEAD", result.Discovery.StatusCode)
    }
    showPhase("ordinary PUT", result.Ordinary)
    for _, segment := range result.Segments {
        fmt.Printf("segment=%q offset=%d size=%d created=%t ambiguous=%t\n",
            segment.Name, segment.Offset, segment.Size, segment.Created, segment.Ambiguous)
        showPhase("segment PUT", segment.Upload)
    }
    showPhase("manifest PUT", result.Manifest)
    for _, cleanup := range result.Cleanup {
        fmt.Println("cleanup target", cleanup.Name)
        showPhase("cleanup DELETE", cleanup.Deletion)
    }
}

func showError(err error) {
    var proof *resource.ResponseError
    if errors.As(err, &proof) {
        fmt.Println("response error", proof.StatusCode, "bytes", len(proof.Body))
    }
    var unconfirmed *objects.ObjectCreateUnconfirmedSegmentError
    if errors.As(err, &unconfirmed) {
        fmt.Println("unconfirmed segment", unconfirmed.Name)
    }
}

func uploadExamples(ctx context.Context, conn *sdk.Connection,
    filename, knownMD5, knownSHA256 string) error {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        return err
    }
    options := []objects.CreateObjectOption{
        objects.WithCreateObjectOpts(objects.CreateObjectOpts{
            Headers: map[string]string{"Content-Type": "application/octet-stream"},
        }),
        objects.WithCreateObjectHeader("X-Trans-Id-Extra", "sdk-upload"),
        objects.WithCreateObjectHeaders(map[string]string{"Cache-Control": "private"}),
        objects.WithCreateObjectMetadata(map[string]string{"owner": "sdk"}),
        objects.WithCreateObjectMetadataValue("empty", ""),
        objects.WithCreateObjectMD5(knownMD5),
        objects.WithCreateObjectSHA256(knownSHA256),
        objects.WithoutCreateObjectSegmentSize(),
        objects.WithoutCreateObjectUseSLO(),
        objects.WithoutCreateObjectGenerateChecksums(),
    }
    result, err := service.Objects.CreateObject(ctx, "backups", "empty ?.bin",
        objects.CreateObjectInput{Data: []byte{}}, options...)
    showCreate(result)
    if err != nil {
        showError(err)
        return err
    }
    stale, err := service.Objects.IsObjectStale(ctx, "backups", "archive ?.tar", filename,
        objects.WithIsObjectStaleOpts(objects.IsObjectStaleOpts{}),
        objects.WithIsObjectStaleHeader("X-Trace", "stale"),
        objects.WithIsObjectStaleHeaders(map[string]string{"X-Trans-Id-Extra": "sdk-stale"}),
        objects.WithIsObjectStaleMD5(knownMD5),
        objects.WithIsObjectStaleSHA256(knownSHA256))
    if stale != nil {
        if stale.Discovery != nil {
            fmt.Println("standalone stale HEAD", stale.Discovery.StatusCode)
        }
        if stale.Stale != nil {
            fmt.Println("observed stale", *stale.Stale)
        }
    }
    if err != nil {
        showError(err)
        return err
    }
    result, err = service.Objects.CreateObject(ctx, "backups", "archive ?.tar",
        objects.CreateObjectInput{Filename: filename}, options...)
    showCreate(result)
    if err != nil {
        showError(err)
        return err
    }
    readerOptions := append([]objects.CreateObjectOption{}, options...)
    readerOptions = append(readerOptions,
        objects.WithCreateObjectMD5(""),
        objects.WithCreateObjectSHA256(""),
        objects.WithCreateObjectSegmentSize(1<<30),
        objects.WithCreateObjectUseSLO(false),
        objects.WithCreateObjectGenerateChecksums(false))
    result, err = service.Objects.CreateObject(ctx, "backups", "stream ?.bin",
        objects.CreateObjectInput{Reader: bytes.NewReader([]byte("reader contents"))},
        readerOptions...)
    showCreate(result)
    if err != nil {
        showError(err)
    }
    return err
}

func main() {}
```

`Data=nil`은 absent, nonnil empty slice는 실제 empty upload입니다. Filename과 Reader도 함께 주거나 모두 생략하면 오류이며 Go는 object 이름을 local filename으로 사용하지 않습니다. Reader는 현재 위치부터 한 번 소비하고 caller를 닫거나 seek하지 않습니다. Reader interface를 담은 input은 동일 stream의 병렬 재사용을 보장하지 않습니다. filename과 Reader는 bounded memory로 private spool을 만들고 실제 size·hash·모든 replay를 같은 bytes에서 얻습니다. local input/spool은 모든 started worker가 끝난 뒤 닫고 제거하며 해당 실패도 유지합니다. 막힌 caller Read의 취소는 caller의 해제 협력이 필요할 수 있습니다.

`GenerateChecksums` nil은 bytes에서 false, file/Reader에서 true입니다. bytes의 명시한 true는 오류이며 MD5·SHA256·SegmentSize·UseSLO는 data branch를 바꾸지 않습니다. file style의 true에서 digest 하나라도 생략되면 둘 다 계산하여 supplied one도 교체합니다. 둘 다 주면 정확하다고 가정합니다. false에서 digest 하나만 있으면 그것만 비교하고, 모두 없으면 existing HEAD 뒤 둘을 계산하지만 comparison-only hash를 upload metadata에 넣지 않습니다. HEAD404에는 이 fallback hashing이 필요하지 않습니다. digest는 32/64자리 ASCII hex만 받으며 대문자 case를 보존합니다. `x-sdk-md5`·`x-sdk-sha256`은 stale 확인용 metadata이며 request ETag와 다릅니다.

stale 비교는 available local digest 하나 이상이 match하고 모든 available digest가 match해야 false입니다. current SDK key는 빈 값이어도 legacy key보다 우선합니다. remote digest는 nullable literal string이며 날짜·ETag·Content-Length는 대체 근거로 사용하지 않습니다. matching stale=false이면 metadata 변경을 포함한 모든 PUT을 생략합니다. standalone `IsObjectStale`도 explicit nonempty filename을 요구하지만 clean HEAD404나 supplied digest가 충분한 existing HEAD에는 파일을 읽지 않습니다. HEAD 뒤 I/O·source·context·projection 오류가 나면 Discovery와 `Stale=nil`을 함께 보존합니다.

file/Reader의 segment size 기본 요청은 1GiB이며 fresh capability의 max-first/min-second 선택과 clean404/412 fallback maximum 2684354561을 사용합니다. explicit0은 유지하고 음수는 input 읽기 전에 거부합니다. empty file은 selected0에도 ordinary PUT이며 nonempty file의 selected size가 0이면 capability 증거와 오류를 반환합니다. file size가 selected size 이하이면 ordinary PUT입니다. 더 크면 기본 SLO, `UseSLO=false`이면 DLO를 사용합니다. false는 작은 file을 강제로 segmented upload하지 않습니다. [기존 capability 계약](../info.md)을 따릅니다.

segmented upload는 object 아래의 unique `.gophercloudsdk-upload-<generation>/` prefix에 저장하고 이를 결과에 반환합니다. index는 최소 6자리이며 필요한 폭을 모두 동일하게 사용해 순서를 유지합니다. 고정 5 workers, segment 최대 2 logical rounds, manifest 최대 3 logical attempts를 사용하며 native physical retry는 별도로 기록합니다. 모든 physical attempt는 처음부터 읽는 독립적인 owned view를 받습니다. accepted 응답의 body·Close·context·source 처리 실패는 재전송하지 않습니다. 성공한 재시도 이전의 오류는 attempt 기록에 남으며 최종 workflow가 성공하면 returned error는 nil입니다. terminal·cleanup·local Close 오류는 반환 오류에 함께 유지합니다.

각 segment PUT은 native retry/redirect alias 이후에도 `If-None-Match: *`를 유지합니다. clean201만 observed created ownership을 줍니다. collision412는 ownership이 없으며 segment202는 실제 acknowledgement와 `ObjectCreateUnconfirmedSegmentError`를 반환하고 manifest publication을 중단합니다. segment ETag는 optional이며 하나의 surrounding quote pair를 logical 값에서 제거합니다. SLO는 literal `/container/segment-name`, exact size와 observed ETag의 ordered JSON array를 보내지만 logical caller Content-Type을 유지하거나 server guess에 맡깁니다. whole-file MD5 또는 manifest JSON MD5를 request ETag로 보내지 않습니다. DLO는 empty body와 이 generation의 slash-terminated prefix를 한 번 quote한 `X-Object-Manifest`를 보냅니다.

ordinary·manifest PUT201/202는 실제 acknowledgement입니다. DLO readable contents, 새 manifest 존재나 cluster completion을 별도로 확정하지 않습니다. terminal 실패의 자동 cleanup은 clean201로 관찰한 `Created && !Ambiguous` exact segment만 대상으로 하고 live context·unchanged source·possible manifest publication 없음이 모두 필요합니다. transport·408·5xx·accepted 처리 오류 등 manifest ambiguity는 전체 native/logical attempts에서 누적되며 나중의 clean rejection으로 지워지지 않습니다. ambiguity·cancellation·source drift이면 prefix와 증거를 남겨 recovery할 수 있게 합니다. prefix listing, collision/unconfirmed 이름 삭제와 background cleanup은 없습니다. cleanup202/204/clean404도 cluster-wide removal을 뜻하지 않습니다.

full options는 전체 설정을 교체합니다. plural Header/Metadata와 singular helper는 canonical 이름으로 순서대로 overlay하고 한 map의 case alias를 거부합니다. metadata는 suffix-only string map으로 empty 값도 보존합니다. file-style supplied/generated digest가 같은 SDK metadata key를 덮어씁니다. auth·framing·ETag·manifest·copy/symlink routing과 raw metadata header는 reserved입니다. upload Headers는 PUT/cleanup에 적용하고 capability/stale에는 captured source headers를 사용합니다. standalone stale Headers는 HEAD에 적용합니다. factory map·pointer·data bytes와 callback 결과는 snapshot하며 callbacks는 한 번 실행합니다. 원래 source·target·context와 live auth를 유지하고 각 attempt의 raw 응답·오류 저장 공간을 분리합니다. `Without`는 nil/default로 복원하며 digest helper의 빈 string은 값을 비웁니다.

[pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L398-L529)는 다음과 같이 사용합니다. container는 따로 생성해야 합니다.

```python
conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "archive.tar", filename=local_filename)
conn.object_store.create_object(
    "backups", "dynamic.tar", filename=local_filename, use_slo=False
)
stale = conn.object_store.is_object_stale("backups", "archive.tar", local_filename)
```

Python의 data path는 Object를 반환하고 file path는 skip·ordinary·SLO·DLO 모두 `None`을 반환합니다. docstring은 missing container 생성과 checksum 기본값 true를 설명하지만 실제 create_object는 container 생성 호출이 없고 data 기본값은 false입니다. `upload_object`는 assignment alias이며 별도 구현이 아닙니다. Python은 filename을 생략하면 name을 local path로 쓰며 caller metadata를 checksum 추가로 변경할 수 있습니다. Go는 explicit source와 owned 설정을 사용합니다.

Python [segment/retry 구현](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L691-L909)에서 사용하는 [FileSegment.seek/reset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L448-L482)은 별도 pos를 초기화하지 않으므로 consumed segment retry가 empty가 될 수 있고 explicit Close도 없습니다. manifest 실패 cleanup은 arbitrary upload의 전체 rollback이 아니라 image-task·autocreated metadata·기본 images container에 제한됩니다. deterministic object/index와 DLO의 넓은 prefix는 기존 segment/neighbor와 섞일 수 있습니다. Go의 immutable spool, unique prefix와 conservative exact-name cleanup은 의도적인 수정입니다. stale legacy fallback도 prefix를 제거한 metadata와 full header key를 비교하는 pinned 경로를 수정합니다.

[Swift PUT API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L153-L299)는 ordinary201, missing container404와 optional ETag mismatch422를 설명합니다. native [Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L173-L281)는 단일 PUT, Content Reader와 lower-level options를 제공하며 기본 ETag 계산이 있습니다. 기존 `Objects.Create`는 그대로 사용할 수 있습니다. [SLO validation/finalization](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1341-L1612)과 [DLO validation](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/dlo.py#L422-L461)은 manifest와 segment의 server-side 의미를 결정합니다. 이 workflow는 heartbeat·wait·Resource dirty state·adapter/session/cache와 cloud image-task cleanup까지 전체 Python parity를 완료한 판정은 아닙니다.

동작 계약은 [Create core](create_core_test.go), [Create options](create_options_test.go), [replay·응답 증거](create_upload_test.go), [Create 외부 계약](create_contracts_test.go), [stale core](stale_core_test.go), [stale options](stale_options_test.go), [stale 외부 계약](stale_contracts_test.go)에서 확인합니다. [Connection](../../../connection_objectstorage_create_object_test.go)과 [generator](../../../internal/cmd/sdkgen/swift_object_create_test.go) 테스트는 shared client와 기존 native/generated 보존을 확인합니다.
