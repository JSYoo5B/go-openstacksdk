# Third-party notices and provenance

Original contributions to **go-openstacksdk** are licensed under
[Apache-2.0](LICENSE), copyright 2026 JaeSang Yoo and the go-openstacksdk
contributors. Third-party material retains the licenses and notices below.
The project license does not relicense the CPython port, Unicode data, or
other upstream material. The [NOTICE](NOTICE) and [license originals](licenses/README.md)
are part of this distribution.

go-openstacksdk is an independent project. It is not affiliated with or
endorsed by the OpenInfra Foundation, OpenStackSDK, or Gophercloud projects.
Upstream names are used to identify dependencies and compatibility references;
the included software licenses do not grant trademark approval for this name.

## Gophercloud v2.15.0: dependency, generated declarations and test source fixtures

- Source: [github.com/gophercloud/gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/tree/v2.15.0).
- License: Apache-2.0; [exact upstream LICENSE](licenses/gophercloud-v2.15.0-LICENSE).
- Copyright: 2012–2013 Rackspace, Inc.; Gophercloud authors.
- Scope: the direct `github.com/gophercloud/gophercloud/v2` dependency,
  generated service type aliases and API wrappers, imported public test helpers,
  and pinned source excerpts in generator tests.

`internal/cmd/sdkgen` reads public Go declarations to generate this SDK's
facades and concrete option adapters. Gophercloud implementations are called
through the dependency rather than copied wholesale into those facades.
Generated headers identify the source version. `internal/testcloud` imports
the public Gophercloud `testhelper` package; it does not vendor that helper's
implementation.

Generator tests under `internal/cmd/sdkgen` also include Gophercloud source
excerpts as fixture strings, including result extraction implementations and
service declarations in `qos_policy_filters_test.go`, `network_filters_test.go`
and `subnet_pool_filters_test.go`. These excerpts retain the Gophercloud
Apache-2.0 attribution above. The tests may combine or alter the excerpts and
add local declarations/stubs to exercise generator behavior; they are test
fixtures rather than a replacement runtime implementation.

## yaml.v2 v2.4.0: dependency

- Source: [go-yaml/yaml v2.4.0](https://github.com/go-yaml/yaml/tree/v2.4.0).
- Licenses: Apache-2.0 for the Go implementation, with MIT terms retained for
  the libyaml port; [exact upstream LICENSE](licenses/yaml-v2.4.0-LICENSE) and
  [exact upstream LICENSE.libyaml](licenses/yaml-v2.4.0-LICENSE.libyaml).
- Copyright: 2011–2016 Canonical Ltd.; libyaml port: (c) 2006 Kirill Simonov.
- Scope: the direct `gopkg.in/yaml.v2` dependency used for cloud configuration.
- Upstream attribution: [exact upstream NOTICE](licenses/yaml-v2.4.0-NOTICE).

The implementation is imported as a Go module, not vendored into this repository.
Its original NOTICE and libyaml license are included for distributions that
incorporate the dependency. Upstream `LICENSE.libyaml` identifies eight files
ported from libyaml C sources: `apic.go`, `emitterc.go`, `parserc.go`, `readerc.go`,
`scannerc.go`, `writerc.go`, `yamlh.go`, and `yamlprivateh.go`. Those files retain
the original MIT copyright and permission terms; the Apache project license does
not replace them.

## go-jmespath v0.4.0: adapted source fork

- Source: [jmespath/go-jmespath v0.4.0](https://github.com/jmespath/go-jmespath/tree/v0.4.0).
- License: Apache-2.0; [original copyright/license notice](licenses/go-jmespath-v0.4.0-LICENSE)
  and the [full Apache license](LICENSE).
- Copyright: 2015 James Saryerwinnie.
- Scope: the adapted engine in `internal/jmespath`.

The upstream API, Pratt parser, lexer, AST, interpreter, functions and generated
token names are retained and modified. The eight adapted source files retain
their origin, copyright and modification notices. The original short upstream
`LICENSE` is also preserved in [internal/jmespath/LICENSE](internal/jmespath/LICENSE);
it is not a substitute for the full license text supplied at the repository root.

The [engine README](internal/jmespath/README.md) describes the local changes:
exact JSON decimal handling, explicit finite arithmetic boundaries, Unicode
string operations, stable nonmutating sorting, copied compiled literals,
deterministic object traversal, and parser/interpreter/lexer error and grammar
corrections. Newly written precision helpers and tests accompany this fork.

## OpenStackSDK: extracted reference metadata and behavioral reference

- Source pin: [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe).
- License: Apache-2.0; [exact pinned LICENSE](licenses/openstacksdk-LICENSE).
- Scope: `api/openstacksdk` public API inventories and resource/filter manifests,
  generated query/default mappings, and the reference for Python-compatible
  SDK behavior and differential tests.

The inventory and manifest tools extract public names, signatures, defaults,
resource declarations, query mappings and source/AST fingerprints. They do not
bundle the Python SDK implementation or execute it during AST extraction.
Individual inventories and manifests identify their pinned source files and
positions. Handwritten Go adapters implement the reviewed contracts; those
reference records do not assert that every Python behavior has been implemented.

The Cloud filtering reference
[`openstack/cloud/_utils.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
retains the source attribution: Copyright (c) 2015 Hewlett-Packard Development
Company, L.P. Other upstream files retain their original owners and notices.
This attribution is not a claim of project ownership over OpenStackSDK material.

## CPython 3.14.8: glob parsing port and test reference

- Source: [CPython 3.14.8 `Lib/fnmatch.py`](https://github.com/python/cpython/blob/v3.14.8/Lib/fnmatch.py).
- License: Python's PSF and historical license agreements;
  [complete audited runtime LICENSE](licenses/python-3.14.8-LICENSE).
- Copyright: (c) 2001 Python Software Foundation; All Rights Reserved.
- Scope: `internal/cloudfilter/glob.go` bracket parsing and CPython behavior
  used in local differential reference data.

**Changes to the port:** `compileGlob` and `compileClass` adapt `fnmatch`'s
bracket construction, including repeated-star compression, negation/closing-bracket
scanning, hyphen chunk splitting and invalid descending-range removal. Go rune
tokens and a dynamic full-string matcher replace Python's generated regular
expression and atomic-regexp syntax. The implementation adds context/source
guard checks and uses explicit UTF-8 input validation. Filesystem separators
and backslash escaping are not given separate path semantics.

The port retains the Python license conditions; the project's Apache license
does not replace them. Python itself and its standard library are not bundled
as an executable dependency of this SDK. The full upstream license is retained
rather than selecting only one agreement from Python's historical license chain.

## Unicode 16.0.0: digit-property and lowercase data

- Reference: [Unicode Character Database 16.0.0](https://www.unicode.org/Public/16.0.0/ucd/ReadMe.txt).
- License: Unicode-3.0; [official Unicode License V3](licenses/unicode-LICENSE).
- Data attribution: Copyright 2024 Unicode, Inc.
- Scope: `descriptorDecimalRanges` and `descriptorNondecimalRanges` in
  `internal/jsonfilter/descriptor_integer.go`, shared by integer conversion helpers.

These compact property ranges are reproducible with the audited CPython 3.14.8
runtime, whose `unicodedata.unidata_version` is `16.0.0`. A scan of all Unicode
code points using `str.isdecimal()` produces the 71 stored decimal ranges;
`str.isdigit() and not str.isdecimal()` produces the 20 stored nondecimal digit
ranges. Both results were verified to match the Go tables exactly. This records
a verified reproduction method; it does not assert an undocumented historical
extraction directly from `UnicodeData.txt`.

The Image record waiter's `internal/cloudfilter/python_lower_data.go` also
contains Unicode 16.0.0 full lowercase mappings and Cased/Case_Ignorable ranges
extracted from the official [UnicodeData.txt](https://www.unicode.org/Public/16.0.0/ucd/UnicodeData.txt),
[SpecialCasing.txt](https://www.unicode.org/Public/16.0.0/ucd/SpecialCasing.txt)
and [DerivedCoreProperties.txt](https://www.unicode.org/Public/16.0.0/ucd/DerivedCoreProperties.txt).
The generated file records each exact source SHA-256 and Unicode-3.0 attribution.
The original Go `PythonLower` algorithm applies language-neutral full mappings
and original-string Final_Sigma context without relying on Go's Unicode version.
These Unicode data retain their own terms; the surrounding original Go algorithm
uses the project's Apache-2.0 license. No Python executable or extra module is bundled.

The data is represented as inclusive Go rune ranges, separate from the original
conversion algorithms. Unicode's copyright and permission terms continue to
apply to that data. The official license text is preserved without changing its
current copyright year range; the UCD 16.0.0 data attribution above is recorded
separately.

## jmespath.py 1.0.1: test reference

- Source: [jmespath/jmespath.py 1.0.1](https://github.com/jmespath/jmespath.py/tree/1.0.1).
- License: MIT; [exact upstream LICENSE.txt](licenses/jmespath-py-v1.0.1-LICENSE).
- Copyright: (c) 2013 Amazon.com, Inc. or its affiliates. All Rights Reserved.
- Scope: comparison runtime used for `internal/jmespath/engine_reference_cases.json`
  and cloud-filter differential reference data.

This Python package is an independently selected comparison reference satisfying
the pinned OpenStackSDK dependency range. Its source is not vendored into this
SDK and it is not a Go runtime dependency. Its MIT license is distinct from the
Apache-2.0 license of the adapted **Go** JMESPath implementation.

## Test reference records

`internal/cloudfilter/testdata/python_cloud_filter_reference.json` records the
audited OpenStackSDK, CPython and JMESPath.py source fingerprints.
`internal/jmespath/engine_reference_cases.json` records the selected Python
runtime's expected values for representative expressions. These are comparison
records, not copies of the referenced Python library implementations. Component
notices above identify their provenance without changing the licenses of any
upstream code or data incorporated by a port.

When adding or changing copied code, translated algorithms, generated source
metadata or external data, preserve the source/version, original notices,
applicable license and a summary of changes alongside the affected component.
