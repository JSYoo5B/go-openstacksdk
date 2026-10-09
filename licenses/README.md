# Retained upstream licenses

These files preserve upstream license and notice texts verbatim. They do not
replace the component scope descriptions in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
Original project contributions use the [root Apache-2.0 license](../LICENSE);
third-party code and data retain their respective terms.

| Retained file | Version/source | SHA-256 |
|---|---|---|
| [Root LICENSE](../LICENSE) | [Apache official text](https://www.apache.org/licenses/LICENSE-2.0.txt) | `cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30` |
| [gophercloud-v2.15.0-LICENSE](gophercloud-v2.15.0-LICENSE) | [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/LICENSE) | `36208a19f74a7af9ab7342c027bb7acf6f27828137bea2919f13ee73321fa002` |
| [yaml-v2.4.0-LICENSE](yaml-v2.4.0-LICENSE) | [yaml v2.4.0](https://github.com/go-yaml/yaml/blob/v2.4.0/LICENSE) | `b40930bbcf80744c86c46a12bc9da056641d722716c378f5659b9e555ef833e1` |
| [yaml-v2.4.0-NOTICE](yaml-v2.4.0-NOTICE) | [yaml v2.4.0](https://github.com/go-yaml/yaml/blob/v2.4.0/NOTICE) | `f6c2dd3a67b576eafb89b80200b8b1627230bf3821a0c14cb99a22ac19107d00` |
| [yaml-v2.4.0-LICENSE.libyaml](yaml-v2.4.0-LICENSE.libyaml) | [yaml v2.4.0 libyaml port](https://github.com/go-yaml/yaml/blob/v2.4.0/LICENSE.libyaml) | `a94710b55e03b5285f77d048c5ba61bb9d6ee04a06c0eb90e68821e11b0c707a` |
| [openstacksdk-LICENSE](openstacksdk-LICENSE) | [OpenStackSDK ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/LICENSE) | `09e8a9bcec8067104652c168685ab0931e7868f9c8284b66f5ae6edae5f1130b` |
| [go-jmespath-v0.4.0-LICENSE](go-jmespath-v0.4.0-LICENSE) | [go-jmespath v0.4.0](https://github.com/jmespath/go-jmespath/blob/v0.4.0/LICENSE) | `03cfaf331a260694d06227a589cdd2b5fcfab474697b72e250736696c313655a` |
| [python-3.14.8-LICENSE](python-3.14.8-LICENSE) | [CPython v3.14.8](https://github.com/python/cpython/blob/v3.14.8/LICENSE), exact audited installed runtime file | `b0e25a78cffb43f4d92de8b61ccfa1f1f98ecbc22330b54b5251e7b6ba010231` |
| [unicode-LICENSE](unicode-LICENSE) | [Official Unicode License V3](https://www.unicode.org/license.txt), retrieved 2026-10-08 | `e7a93b009565cfce55919a381437ac4db883e9da2126fa28b91d12732bc53d96` |
| [jmespath-py-v1.0.1-LICENSE](jmespath-py-v1.0.1-LICENSE) | [jmespath.py 1.0.1](https://github.com/jmespath/jmespath.py/blob/1.0.1/LICENSE.txt) | `66b313cce80ed0623fc7db3f24863a0c80fd83eb341a46b57864158ae74faa56` |
| [golang-x-crypto-v0.55.0-LICENSE](golang-x-crypto-v0.55.0-LICENSE) | [golang.org/x/crypto v0.55.0](https://cs.opensource.google/go/x/crypto/+/v0.55.0:LICENSE) | `911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad` |
| [golang-x-sys-v0.47.0-LICENSE](golang-x-sys-v0.47.0-LICENSE) | [golang.org/x/sys v0.47.0](https://cs.opensource.google/go/x/sys/+/v0.47.0:LICENSE) | `911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad` |

Gophercloud and yaml originals were copied from the pinned module cache;
OpenStackSDK and go-jmespath originals were copied from their pinned reference
checkouts. The Python file was copied from the audited Python 3.14.8 installation.
Apache, Unicode and JMESPath.py texts were downloaded from the exact URLs above.
The copied files were checked for byte equality with their local originals;
downloaded files were hashed without rewriting whitespace or copyright text.

The go-jmespath upstream `LICENSE` is a copyright and Apache application notice,
not the full Apache license. It is intentionally retained unchanged together
with the full root license and [existing internal notice](../internal/jmespath/LICENSE).
The YAML `NOTICE` is retained in addition to its Apache license. Its separate
`LICENSE.libyaml` preserves the original MIT terms and Kirill Simonov copyright
for the eight files ported from libyaml.

The Unicode license is the official retrieved text, which currently carries
Copyright © 1991–2026 Unicode, Inc. The [UCD 16.0.0 source notice](https://www.unicode.org/Public/16.0.0/ucd/ReadMe.txt)
separately attributes the 2024 data release. The README and NOTICE preserve that
data attribution without editing the upstream license text.

The golang.org/x/crypto and golang.org/x/sys BSD-3-Clause originals were copied
verbatim from the pinned module cache. They cover imported BLAKE2 and its CPU
capability dependency; no implementation source is copied into this SDK.

The pinned Go module-to-license mapping is maintained in
[dependencies.json](dependencies.json). Run `make license-check` to verify the
original hashes, runtime module coverage, adapted JMESPath notice, and required
distribution notices. The check runs offline and is included in `make check`.
A new module or version must be reviewed before updating this mapping. This
checks the recorded distribution files; it does not provide trademark approval
or independently determine legal compliance.
