# Built-in vendor profiles

`data/` holds the vendor profile files of
[OpenStackSDK `openstack/config/vendors`](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/vendors)
at the pinned source commit `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`,
copied byte for byte without modification. OpenStackSDK is licensed under
Apache-2.0; the exact pinned license text is
[`licenses/openstacksdk-LICENSE`](../../licenses/openstacksdk-LICENSE). The files
carry no file-level copyright notice and remain the work of their upstream
OpenStackSDK contributors. See [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md).

`vendors.go` is new project code (Apache-2.0) that reads these files the way
`openstack.config.vendors.get_profile` and the profile branch of
`OpenStackConfig._expand_vendor_profile` do: YAML files first, then JSON; a
nested `profile` key replaces the outer settings when it names another built-in
profile; `status: shutdown` is an error; `status` and `message` are removed.
Remote profiles given as URLs are not fetched (`ErrRemoteProfile`), and
deprecated profiles produce no warning.

When the source pin changes, replace the files and the hashes below together;
`vendors_test.go` checks them.

| File | SHA-256 |
| --- | --- |
| `auro.json` | `f96b793f5295bfb98e6b1f74aa7091ca7a2bb009bdd3f226dd292273db087ba7` |
| `betacloud.json` | `3ff7459c4313b964145e73fd366c87b1575664a5f17578d44d18bfa0577e48ae` |
| `binero.json` | `6798be65bf8574a9dbca92d71c0f69825c3748731cef51df452b882c6824bbdf` |
| `bluebox.json` | `9931fb24e2133232a4895a66cfe062565e7429c2d751997ecca15577cfbd0a1b` |
| `catalyst.json` | `9ad7cc777ccf79d924deaf67396ab09fc0aea4f9231676275894674ef44af6c3` |
| `citycloud.json` | `0016be0790b2d78667e959df2612551c97e3e052214d17a9a1f3c109eb54c4fc` |
| `conoha.json` | `9a669563a93954e10a76d5b88bdb481086c725a74e3ef79ca2e88e900113b2df` |
| `dreamcompute.json` | `3584ab4c5642e77d68879d37fefdbfbfba24ca40d4d9be3a1cc5dd4f7fbd7715` |
| `elastx.json` | `4bd11241025777a00ca749eb2aa8ec25ccd3aa7315d8c2fb40de6a961b4c6d83` |
| `entercloudsuite.json` | `43d91e57572dc41761f3fe77c05a2dba878880666d76899ff871a1a26cdf3197` |
| `fuga.json` | `dcdcfb7c945290a3bd9763911d4b13f1ba8418c118317ee1b0a58e06e2473fcc` |
| `ibmcloud.json` | `df192f2f6a3a0ccabcd360000e9964204831817649ed6b0fbf321c2ec5cb16d2` |
| `internap.json` | `267b01ef466d88f37f996f6758d9f43f4eddbb081a92d7c30fba07aefe00889e` |
| `limestonenetworks.yaml` | `96456c8b80c8b6427ffaa5a66fe38fd04658be72011c9791e30d7cd5f2617064` |
| `otc-swiss.json` | `28fb221a19086588054cdbfb9cbc103fce7143fec69cb07de5727b108ef6fdff` |
| `otc.json` | `38bee16a5f80d60324aab71e5a1e60a3ba53a1f5332d8b85e520af78cba22c8d` |
| `ovh-us.json` | `ecc3c4405b4fb045bb5e934fa87fb16eab6fe66946aa351ceaf66fb9d2462bf8` |
| `ovh.json` | `5f47a90fe9e0b53612f64cc624f5d981e9b89582281ff1b6ca2459b306ec30f2` |
| `rackspace.json` | `ef9bda10a374eacb140244ebc8d1e2d5aa34bb7f20d2f8fd22e2594e5a6c70c1` |
| `switchcloud.json` | `80b1fc106e498fe7f7b03eea0e9cd968e02e4ed88ab909e893da068230254166` |
| `switchengines.json` | `88115d5f99004554d852a9ba478a62df92c83961205d1d588c06182a5d6cca5b` |
| `ultimum.json` | `997b9bfc374f2145b129d322352bf44910228f924f0b812683b18e96c42f30f1` |
| `unitedstack.json` | `210d21017376ab5ddc630ea5886815325996dc28f5639c3115cf4abc14e25468` |
| `vexxhost.json` | `19b49293ca04a10affacd2816769935799efaa1d53e732d8229faa73caf02be7` |
| `zetta.json` | `4c9352b4530333bc8cf53ae214c16d755885f910847525fc9d91e24b42f12868` |
