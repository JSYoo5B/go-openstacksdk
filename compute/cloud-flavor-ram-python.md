# Cloud Flavor Ram Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/get_flavor_by_ram`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AllFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavorByRAM`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMInclude`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMGetExtra`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMGetExtraValue`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListSourceRecords`

- AllFlavors reproduces the list_flavors(get_extra=True) requests that get_flavor_by_ram starts with.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`.
- SDK eagerly consumes every detailed page and required extra-specs fetch before stable ascending RAM selection; default get_extra is true, inline truthy specs skip fetch, tied RAM preserves original page/row order, minimum/include are not pushed down. A late extra/page failure prevents otherwise valid early selection.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMStableSortAllPagesDefaultEnrichmentAndActualProof`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMExtraTruthinessInlineSkipAndLateFailureBeforeSelection`.
- Source RAM descriptor and threshold comparison preserve null failure timing, missing0, signed-string-to0, raw bool, float-to-int binary truncation and exact mixed arbitrary-int/loaded-float ordering. Empty inventory raises completed no-match without prematurely validating unused comparison types.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMThresholdJSONNumbersDescriptorsAndNoMatchOrder`.
- Include is a case-sensitive substring for normal string names, not fnmatch. Falsey JSON skips membership; array names use Python-loaded equality and object names use key membership. Wrong types only fail when reached after RAM comparison; null/noniterable below-threshold names do not fail early.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMIncludeSubstringTruthinessContainersAndShortCircuit`.
- Concrete library-owned options snapshot JSON bytes/DTOs/query-option slices and isolate retained callback pointers. Existing flavor headers/version/extra read options are reused; explicit RAM GetExtra is final override, no filters argument is invented. Nil malformed options/context, selected API mutation and custom cancellation fail before HTTP with native cause preserved.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMOwnsOptionsAndInheritedReadTransportOptions`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMPreflightSelectedBindingCancellationAndUnsupportedOptions`.
- Flavor and Inventory records are separately cloned; Source Resource/default/location differs from actual Wire row. FlavorRecord.Envelope is the decoded flat row (unchanged for UTF8 input), page Header/Status203 remain actual, and extra receipt201 keeps whole physical extra response and numeric vendor precision. No-match is ErrNoFlavorByRAM, not fabricated HTTP404; local errors reference consumed row evidence.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMStableSortAllPagesDefaultEnrichmentAndActualProof`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMThresholdJSONNumbersDescriptorsAndNoMatchOrder`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMExtraTruthinessInlineSkipAndLateFailureBeforeSelection`.
- Existing guarded reader preserves native401 actual failed body, physical Read/Close error chain, custom cancel cause and transient/restored client-source invalidation without returning completed selection. Clean503 retry is exercised with original list route. Existing guard/retry invariants remain; the RAM-only Source status policy is covered by the dedicated contract below.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMPhysicalFaultsNative401AndCleanRetry`.
- RAM-only Source collection and conditional enrichment accept100..399 through ListSourceRecords. Existing AllFlavors/Search/Get/ListRecords/standalone FetchExtraSpecs keep200..399. Actual status/header/body are retained; Requests charset decoding is not implied by this policy.
  Tests: `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMSourceStatusListAndEnrichment`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMSourceStatusPreservesExistingReaders`.
- RAM-only Source-status list and conditional extra-specs normalize explicit Latin1 and UTF16/32 BOM or LE/BE bytes before JSON decode. Physical list PageEnvelope and extra Envelope/header/status remain original; rows use decoded UTF8 JSON. Selected/inventory proofs are independently cloned. Decode, later-page and semantic failures preserve actual physical response bytes without overwriting a later HTTP rejection; legacy readers keep their decoder policy. Unknown codec and malformed headers fail explicitly under the finite Go response profile.
  Tests: `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMExplicitCharsetsListAndEnrichment`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMCharsetFailuresKeepPhysicalEvidence`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMCharsetLaterPageAndSemanticFailuresKeepActualProof`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMCharsetDoesNotChangeLegacyReaders`.

Python/Go differences:

- Go returns owned FlavorRAMResult with selected Flavor and actual consumed Inventory, instead of only mutable Python Flavor. Resource/default/location, Wire, per-row list Envelope and separate extra-specs whole response evidence remain distinct; local no-match uses errors.Is(ErrNoFlavorByRAM), comparison/membership errors use ErrInvalidOption.
- Public any inputs are SDK-owned JSON-domain primitive/container snapshots, not required application builder interfaces. RawMessage preserves numeric spelling. Python arbitrary classes/operators, NaN/Infinity runtime float values, configurable huge-integer string conversion limits and custom runtime callback timing are not reproduced. Safe Go IDs/finite explicit-charset JSON/header/selected-source policy and existing foundation SDK-R1/SDK-C1/SDK-S1 remain explicit; supported increment0.
- Actual imported pinned Source under Python3.14.8 ran49 cloud→NovaProxy→Flavor/Resource cases, plus2 status199 list/extra cases. Only Adapter Response/connection were substituted. Mock/Go/race/main proof is not actual authenticated backend/role/deployment acceptance.
- Existing AllFlavors default false remains unchanged while this helper owns default true. Root Connection forwarding is not added; conn.Compute(ctx) returns the service. Existing native and generic flavor reader policies are preserved. Existing RAM/status tests plus new explicit-charset integration/root validation are distinct from actual Python Source execution.
- Explicit response profile now supports Latin1 and UTF16/32 BOM or LE/BE on RAM Source-status reads. Unmarked charset sniffing, BOMless generic UTF16/32 native-endian decoding, arbitrary Requests codecs and nonfinite/permissive JSON remain documented Go format differences rather than claims of universal Requests decoding. PageEnvelope is physical whole-page bytes while Envelope is the decoded flat row; extra Envelope remains physical response bytes.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--ram", default="4096", help="minimum RAM as JSON number")
parser.add_argument("--include", default="null", help="name substring as JSON string, or null")
parser.add_argument("--get-extra", action=argparse.BooleanOptionalAction, default=True)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_api_version="2.55") as conn:
    result = conn.get_flavor_by_ram(
        json.loads(args.ram),
        include=json.loads(args.include),
        get_extra=args.get_extra,
    )
    print(json.dumps(result.to_dict(), indent=2))
```

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	ram := flag.String("ram", "4096", "minimum RAM as JSON number")
	include := flag.String("include", "null", "name substring as JSON string, or null")
	extra := flag.Bool("get-extra", true, "fetch missing extra specs before selection")
	flag.Parse()
	if !json.Valid([]byte(*ram)) || !json.Valid([]byte(*include)) {
		log.Fatal("ram and include must be complete JSON")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	result, readErr := service.GetFlavorByRAM(ctx, json.RawMessage(*ram),
		compute.WithFlavorRAMInclude(json.RawMessage(*include)),
		compute.WithFlavorRAMGetExtra(*extra))
	var selected any
	inventoryCount := 0
	if result != nil {
		inventoryCount = len(result.Inventory)
		if row := result.Flavor; row != nil {
            var enrichment any
            if e := row.Enrichment; e != nil {
                enrichment = map[string]any{
                    "resource": e.Resource, "wire": e.Wire, "extra_specs": e.ExtraSpecs,
                    "envelope_bytes": []byte(e.Envelope), "header": e.Header, "status_code": e.StatusCode,
                }
            }
			selected = map[string]any{
				"resource": row.Resource, "wire": row.Wire,
				"envelope": row.Envelope, "page_envelope_bytes": []byte(row.PageEnvelope), "header": row.Header,
				"status_code": row.StatusCode, "enrichment": enrichment,
			}
		}
	}
	data, printErr := json.MarshalIndent(map[string]any{
		"selected": selected, "inventory_count": inventoryCount,
	}, "", "  ")
	if printErr == nil {
		fmt.Println(string(data))
	}
	if err := errors.Join(readErr, printErr); err != nil {
		log.Fatal(err)
	}
}
```

## References

- [Install](../docs/install.md)
