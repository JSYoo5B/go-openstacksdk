# Licensing and attribution

Project code is licensed under [Apache-2.0](../LICENSE). Included third-party code, ports, and data retain their original conditions. Distributions must preserve [NOTICE](../NOTICE), [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md), [license texts](../licenses/README.md), and file-level notices.

## Components

- `internal/jmespath` is a modified go-jmespath v0.4.0 fork. Preserve James Saryerwinnie's copyright, Apache notice, its directory LICENSE, and the complete root license. Its README documents modifications.
- Bracket translation in `internal/cloudfilter/glob.go` derives from CPython 3.14.8 `Lib/fnmatch.py`. Preserve Python license texts and the port notice.
- `internal/jsonfilter/descriptor_integer.go` fixes Unicode 16.0.0 numeric properties. `internal/cloudfilter/python_lower_data.go` derives full lowercase/context tables from official Unicode 16 data. Preserve source hashes, Unicode-3.0 attribution, copyright, and conditions.
- Gophercloud v2.15.0 and yaml.v2 v2.4.0 are module dependencies. Generator test excerpts/adaptations retain upstream scope, copyright, and modification notices. Preserve YAML's upstream NOTICE; the Go implementation uses Apache-2.0 and eight libyaml-derived files retain Kirill Simonov's MIT conditions and LICENSE.libyaml.
- `golang.org/x/crypto` v0.55.0 provides BLAKE2b/BLAKE2s; indirect `golang.org/x/sys` v0.47.0 supplies CPU support. Preserve The Go Authors' BSD-3-Clause texts. Implementations remain module dependencies.
- `github.com/klauspost/compress` v1.18.0 provides Swift zstd decoding. Preserve its complete module LICENSE, vendored xxhash MIT and Snappy reference BSD texts; distinguish other Apache/MIT package scopes in the notices.
- `golang.org/x/text` v0.41.0 provides explicit encoding/transform support. Preserve The Go Authors' BSD-3-Clause license. Statistical Python charset detection remains unimplemented.
- `internal/cloudvendors/data` is an unmodified copy of the pinned OpenStackSDK `openstack/config/vendors` profile files (Apache-2.0), embedded for built-in vendor profile resolution. Preserve the files unchanged with their README hash list and notice.
- Pinned OpenStackSDK provides public declarations/resource metadata for comparison. Python and jmespath.py are reference tools, not SDK runtime dependencies. Pins, source attribution, and comparison scope are specified in the integrated notices.

See [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) for exact component boundaries and original terms. Dependencies do not change the Go 1.25 module minimum.

## Names and contributions

This independent SDK is developed by JaeSang Yoo and contributors. It does not claim official affiliation, endorsement, or sponsorship by OpenStack, OpenInfra Foundation, or Gophercloud. Code licensing does not grant trademark rights; the [OpenInfra trademark policy](https://www.openinfra.dev/legal/trademark-policy/) applies separately.

New project contributions use Apache-2.0. Changes incorporating external code/data must preserve original copyrights, licenses, versions, sources, and modification notices. Do not replace upstream attribution with project attribution.

## Automated checks

`make license-check` verifies hashes for sixteen stored license texts, mappings for the six go.mod dependencies, fork notices, and distribution notices without network access. It is included in `make check`. Review original terms and update [dependencies.json](../licenses/dependencies.json) when dependencies change. File-integrity checks do not replace legal interpretation or trademark permission.
