# Flavor 조회와 extra specs

`conn.Compute(ctx)`의 `service.API.Flavors`는 native flavor API와 SDK가 소유하는 extra specs 조회를 제공합니다. 이름/ID 검색은 `service.Flavors` collection에서 사용합니다. [Compute 조회 가이드](../../user-read-apis.md)에 일반 Get/List/Find와 기존 extra specs 검색 옵션을 설명합니다.

`FetchExtraSpecs(ctx, FlavorExtraSpecsRequest, options...)`는 openstacksdk의 `conn.compute.fetch_flavor_extra_specs(flavor)`에 대응합니다. ID·기존 `Flavor`·기존 `resource.RawResource`를 concrete 입력으로 받으며 inline extra specs가 있어도 `/flavors/{id}/os-extra_specs`를 조회합니다. ID를 얻기 위한 flavor GET/List나 이름 검색을 추가하지 않습니다.

[Flavor extra specs 가이드](../../flavor-extra-specs.md)에 Python 비교와 독립 실행 Go main, 입력 snapshot·route ID, `WithFlavorExtraSpecsOptions`/`Microversion`/`Header`, 선택된 버전 유지·자동2.61 상한·명시 빈 버전을 설명합니다.

반환 `FlavorExtraSpecsRecord`의 Resource는 기존 입력의 소유 복사에 extra_specs만 갱신합니다. 실제 extra_specs 객체와 null은 유지하고, 생략 또는 nonnull 비객체 값은 Python dict descriptor처럼 `{}`로 표현합니다. 별도 ExtraSpecs/Wire/Envelope는 실제 값을 보존하고 record의 Header/StatusCode는 이번 응답을 나타냅니다. 오류에 receipt가 함께 반환되어도 Resource가 nil일 수 있으므로 error를 확인합니다.

기존 `ListExtraSpecs(ctx, id)`는 native200 성공 코드와 `map[string]string` 반환을 유지합니다. owned FetchExtraSpecs의200..399 처리·raw JSON·dict view와 구분하여 필요한 결과를 선택합니다. FetchExtraSpecs는 기존 입력과 공유 client 설정을 변경하지 않습니다.

값 하나는 `GetExtraSpecsProperty(ctx, FlavorExtraSpecsRequest{ID: flavorID}, property)`로 조회합니다. raw `Value`와 `Present`로 missing/null/빈 값·비문자열을 구분하며 supplied flavor를 변경하거나 전체 specs를 추가 조회하지 않습니다. Cloud ID 조회는 `service.GetFlavorByID(ctx, id)`와 `compute.WithFlavorByIDExtraSpecs(true)`를 사용합니다. [단일 property·native 조회 가이드](../../flavor-property-and-native.md)에 독립 main과 기존 `Get`·`GetExtraSpec`·`ListDetail`의 native DTO·상태·query 차이를 설명합니다.
