# Changelog

## [1.15.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.15.1...v1.15.2) (2025-05-13)


### Bug Fixes

* update dependencies to latest ([#680](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/680)) ([8962c17](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/8962c17aabc71eaf73ec058beb356c9327d3bbce))

## [1.15.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.15.0...v1.15.1) (2025-04-14)


### Bug Fixes

* configure Cloud Monitoring client correctly ([#673](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/673)) ([91d86af](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/91d86aff0496dfd58a46e894c402b581d4211d5e))
* shut down the internal exporter only once ([#671](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/671)) ([16a6782](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/16a67829b4e86e5d752e9b00220d06faedb4bbbd)), closes [#776](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/776)

## [1.15.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.14.1...v1.15.0) (2025-03-11)


### Features

* add support for Go 1.24 and drop Go 1.22 ([#658](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/658)) ([e9611ac](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/e9611ace582c264f513e5175cf50aa5e3a144b69))
* allow for disabling built-in metrics ([#665](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/665)) ([3478f8f](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/3478f8f6ff827cff1792533c97e1dcb8c25bcdf4))

## [1.14.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.14.0...v1.14.1) (2025-02-11)


### Bug Fixes

* update dependencies to latest ([#653](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/653)) ([b787980](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b787980feb5115303800a302d2aa121968e304b6))

## [1.14.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.13.2...v1.14.0) (2025-01-14)


### Features

* match official Go version support policy ([#647](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/647)) ([a84b1d4](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/a84b1d43024dc6de92a0e55211b10870571cd5f4))

## [1.13.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.13.1...v1.13.2) (2024-12-10)


### Bug Fixes

* update dependencies to latest ([#640](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/640)) ([0134109](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/0134109fb9cb347234e328de3c6cd5a240cfecdf))

## [1.13.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.13.0...v1.13.1) (2024-11-12)


### Bug Fixes

* bump dependencies to latest ([#634](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/634)) ([11895c7](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/11895c72bb9dbc0275433b8072ba8df0f07bfae0))

## [1.13.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.12.1...v1.13.0) (2024-10-08)


### Features

* add bytes_sent and bytes_received as metrics ([#624](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/624)) ([4aa27a5](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/4aa27a520d1e2fe14410e8078cfc38bc29209629))

## [1.12.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.12.0...v1.12.1) (2024-09-10)


### Bug Fixes

* update dependencies to latest ([#617](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/617)) ([9c3865d](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/9c3865d94ead7579ce566f041fc2fec05d11bf7d))

## [1.12.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.11.1...v1.12.0) (2024-08-14)


### Features

* add support for Go 1.23 ([#609](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/609)) ([d8d6261](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/d8d6261f17e0fdd508c234193cbac11d2c47b436))

## [1.11.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.11.0...v1.11.1) (2024-07-10)


### Bug Fixes

* rely on the PSC DNS name instead of the private IP address ([#590](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/590)) ([f4dd341](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/f4dd34160d1fb0ee04e0edab1d79bdd0dc69e8c7))

## [1.11.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.10.0...v1.11.0) (2024-06-12)


### Features

* generate RSA key lazily for lazy refresh ([#589](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/589)) ([f106169](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/f106169b8eee837d9b37d72de22d3a8b86c02966)), closes [#584](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/584)


### Bug Fixes

* ensure connection count is correctly reported ([#586](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/586)) ([b640ffb](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b640ffbc6304f1f4bf67b8a2792f54c8feffee38))

## [1.10.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.9.0...v1.10.0) (2024-05-14)


### Features

* add context debug logger ([#573](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/573)) ([375cca3](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/375cca3af2d4a074fea7597e0d6be35e72a2976d))
* add support for a lazy refresh ([#565](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/565)) ([75fb63e](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/75fb63e9b9a77427c58be8e57705eb8cdaf91e41)), closes [#549](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/549)
* invalidate cache on IP type errors ([#555](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/555)) ([154ab5f](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/154ab5f7197eca3e57dceb5c337d0beb187e2496)), closes [#554](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/554)
* support static connection info ([#572](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/572)) ([af8b703](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/af8b7039e288a4d10b6865df0bf5e45985f5ed5c))

## [1.9.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.8.0...v1.9.0) (2024-04-16)


### Features

* add support for PSC ([#537](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/537)) ([7b79b32](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/7b79b32b0d0dfb3a8302155c0571a093fe3583bf))


### Bug Fixes

* return a friendly error if the dialer is closed ([#538](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/538)) ([66d7bd0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/66d7bd0ce7f5a3a66eacacb594b9eb743fbbce86)), closes [#522](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/522)

## [1.8.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.7.0...v1.8.0) (2024-03-12)


### Features

* add support for debug logging ([#523](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/523)) ([a9b8557](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/a9b8557ffb4ea046cf19961a8f7fed86548e6ac8)), closes [#506](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/506)
* add support for PSC ([#513](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/513)) ([614dbd3](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/614dbd3382fc9fca137eb234a8133572fa1ad3a7))


### Bug Fixes

* remove duplicate refresh for all connections ([#526](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/526)) ([e9f63a3](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/e9f63a3ffd5d75b7c961913097d64c70e2c23320))

## [1.7.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.6.0...v1.7.0) (2024-02-13)


### Features

* add support for Go 1.22 ([#504](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/504)) ([a944afb](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/a944afb862324b6b987cfe357957621d4065d9f5))

## [1.6.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.5.2...v1.6.0) (2024-01-29)


### Features

* add support for public IP ([#474](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/474)) ([e51ef9b](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/e51ef9b873111f142f1e6d70f006eee35456c4aa))


### Bug Fixes

* avoid scheduling an instance refresh if the context is done ([#491](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/491)) ([42c8ae3](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/42c8ae3456d88957138abfd95d90bab8c6448f72)), closes [#493](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/493)

## [1.5.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.5.1...v1.5.2) (2024-01-17)


### Bug Fixes

* update dependencies to latest versions ([#476](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/476)) ([eee45e8](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/eee45e8b17fc5c8aabcc471c3782d6e5734bb4cb))

## [1.5.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.5.0...v1.5.1) (2023-12-13)


### Bug Fixes

* ensure cert refresh recovers from computer sleep ([#456](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/456)) ([79fcbc8](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/79fcbc8d6b2c926e0e16eb5806b61aae90860780))

## [1.5.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.4.1...v1.5.0) (2023-11-15)


### Features

* add pgx v5 support ([#395](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/395)) ([#413](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/413)) ([c07799c](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/c07799cbd194d003e558be136961511b37481f4f))
* add support for Auto IAM AuthN ([#358](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/358)) ([e50dd25](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/e50dd25503ba29125023b8dc3084f4876079c48b))


### Bug Fixes

* use HandshakeContext by default ([#417](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/417)) ([81bd2d6](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/81bd2d651db7adf91f9e404f278354364ffea73d))

## [1.4.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.4.0...v1.4.1) (2023-10-11)


### Bug Fixes

* bump minimum supported Go version to 1.19 ([#394](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/394)) ([b16f269](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b16f269d39ec3478ac7985d82f8273185f7222ad))
* update dependencies to latest versions ([#380](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/380)) ([0e6a42e](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/0e6a42e5e45ba0c7f7783613b6579714a05017d3))

## [1.4.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.3.4...v1.4.0) (2023-08-28)


### Features

* add support for WithOneOffDialFunc ([#364](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/364)) ([a54b649](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/a54b64931e9bd0527f0162d187a008aea59563f2))


### Bug Fixes

* update ForceRefresh to block if invalid ([#360](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/360)) ([b0c9ffa](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b0c9ffa9c2a781a7b7b10eb7e1a3ff3a1b99f59e))

## [1.3.4](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.3.3...v1.3.4) (2023-08-16)


### Bug Fixes

* re-use current connection info during force refresh ([#356](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/356)) ([6dfadff](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/6dfadfff1de5c96127a7a55c179f406a4117410a))

## [1.3.3](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.3.2...v1.3.3) (2023-08-08)


### Bug Fixes

* avoid holding lock over IO ([#333](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/333)) ([888e735](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/888e735492d25b1c42194213038f4458e4b96aaf))

## [1.3.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.3.1...v1.3.2) (2023-07-11)


### Bug Fixes

* update dependencies to latest ([#330](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/330)) ([da2758c](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/da2758c0a2af998fd8dad9377ad718ef345bb6cd))

## [1.3.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.3.0...v1.3.1) (2023-06-12)


### Bug Fixes

* remove leading slash from metric names ([#313](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/313)) ([3a6b675](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/3a6b675abd8e2520a65e93b172972430290fba23)), closes [#311](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/311)
* stop background refresh for bad instances ([#308](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/308)) ([8965aa5](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/8965aa5aee7c623d6bea171015f4909e292ad716))

## [1.3.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.2.2...v1.3.0) (2023-05-09)


### Features

* use auto-generated AlloyDB client ([#268](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/268)) ([6613965](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/66139656f17dfdf0c34a987f86034135f70974c6))
* use instance IP as SAN ([#289](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/289)) ([30d9740](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/30d9740885d8aa1c31877cb12f8754ccdf418e1c))


### Bug Fixes

* require TLS 1.3 always ([#292](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/292)) ([05f8430](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/05f84302b11c7abeff22c1a6a04c18f9b61cd19b))

## [1.2.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.2.1...v1.2.2) (2023-04-11)


### Bug Fixes

* update dependencies to latest versions ([#277](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/277)) ([e263db1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/e263db1c5d53beda6df5a91faa1ef0cb085cb50b))

## [1.2.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.2.0...v1.2.1) (2023-03-14)


### Bug Fixes

* update dependencies to latest versions ([#247](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/247)) ([5c5b680](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/5c5b68029e2bdb6f41faeedf35e970f3ca316636))

## [1.2.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.1.0...v1.2.0) (2023-02-15)


### Features

* add support for Go 1.20 ([#216](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/216)) ([43e16c0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/43e16c049b7e2d55c73ee2a21ef936f18620923f))


### Bug Fixes

* improve reliability of certificate refresh ([#220](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/220)) ([db686a9](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/db686a9058b7998472a3a32df6598c90390abf84))
* prevent repeated context expired errors ([#228](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/228)) ([33d1369](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/33d1369f4ce7011b15e91004caddc350a64d2127))

## [1.1.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v1.0.0...v1.1.0) (2023-01-10)


### Features

* use handshake context when possible ([#199](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/199)) ([533eb4e](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/533eb4e3cce97ac5f3fbfa3c0c7cd4f2e857ff05))

## [1.0.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.4.0...v1.0.0) (2022-12-13)


### Miscellaneous Chores

* release 1.0.0 ([#188](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/188)) ([34c9c5b](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/34c9c5b70be51ef8dc3a25ce92f730cc002b1571))

## [0.4.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.3.1...v0.4.0) (2022-11-28)


### Features

* limit ephemeral certificates to 1 hour ([#168](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/168)) ([b9bb918](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b9bb918a1a9befb44c4a0cfce5e7a48a80e3ea20))

## [0.3.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.3.0...v0.3.1) (2022-11-01)


### Bug Fixes

* update dependencies to latest versions ([#150](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/150)) ([369121b](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/369121b7421243c2be6f2fa3e6c998a8d01d08e2))

## [0.3.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.2.2...v0.3.0) (2022-10-18)


### Features

* add support for Go 1.19 ([#123](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/123)) ([8e93b9f](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/8e93b9fd5ad508b4f30eb62ccedfcf326d34e03d))

## [0.2.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.2.1...v0.2.2) (2022-09-07)


### Bug Fixes

* support shorter refresh durations ([#103](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/103)) ([6f6a7a0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/6f6a7a05875c3d62a8a71cd54c59db8d793d3c25))

## [0.2.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.2.0...v0.2.1) (2022-08-01)


### Bug Fixes

* include intermediate cert when verifying server ([#83](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/83)) ([072c20d](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/072c20d974ac6705617f10cd8f3889a4adc685ee))

## [0.2.0](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.1.2...v0.2.0) (2022-07-12)


### ⚠ BREAKING CHANGES

* use instance uri instead of conn name (#15)

### Features

* add AlloyDB instance type ([da23ca9](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/da23ca9579f5b90e86287e5b7dc689a549ea9240))
* add AlloyDB refresher ([c3a4372](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/c3a43727a1b1d76ce50c288155fa8c6bb31d09ab))
* add AlloyDB refresher ([#2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/2)) ([d0d6a11](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/d0d6a119fcb3cc5613de065a168f415dbce70789))
* add support for dialer ([#4](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/4)) ([483ffda](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/483ffdae1870835db79aa04c59a6322b9ec8e9bb))
* add WithUserAgent opt ([#10](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/10)) ([6582164](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/658216477813b92aadfd44403b9389dcaea9f081))
* switch to Connect API and verify server name ([#70](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/70)) ([36197b6](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/36197b6c9f6626952d37e30087d986c4226a13dc))
* switch to prod endpoint ([#13](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/13)) ([b477122](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b47712202088e43533820c51633dff65fe552ce4))
* use v1beta endpoint ([#16](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/16)) ([bfe5fe5](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/bfe5fe56294c76bf7be4ad1ba09cc7b982479d24))


### Bug Fixes

* adjust alignment for 32-bit arch ([#33](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/33)) ([b0e76fa](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b0e76fa5384fc66365b5d15b56927942f4031fda))
* admin API client handles non-20x responses ([#14](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/14)) ([c2f5dc9](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/c2f5dc92e1a57262c10cd715fc6082a931d0cf70))
* prevent memory leak in driver ([#22](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/22)) ([861d798](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/861d798e0715f16b88d501950a8d9a0493cc8257))
* specify scope for WithCredentialsFile/JSON ([#29](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/29)) ([9424d57](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/9424d572346f16cee86e80dccc9e01618b97df73))
* update dependencies to latest versions ([#55](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/55)) ([7e3af54](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/7e3af549b4991d77348751b8f1fa9d0074846782))
* use instance uri instead of conn name ([#15](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/15)) ([0da01fd](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/0da01fd311f1e8829be0a9eb0efdeb169ee7c555))


## [0.1.2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.1.1...v0.1.2) (2022-06-07)


### Bug Fixes

* update dependencies to latest versions ([#55](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/55)) ([7e3af54](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/7e3af549b4991d77348751b8f1fa9d0074846782))

### [0.1.1](https://github.com/GoogleCloudPlatform/alloydb-go-connector/compare/v0.1.0...v0.1.1) (2022-05-18)


### Bug Fixes

* adjust alignment for 32-bit arch ([#33](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/33)) ([b0e76fa](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b0e76fa5384fc66365b5d15b56927942f4031fda))
* specify scope for WithCredentialsFile/JSON ([#29](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/29)) ([9424d57](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/9424d572346f16cee86e80dccc9e01618b97df73))

## 0.1.0 (2022-04-26)


### ⚠ BREAKING CHANGES

* use instance uri instead of conn name (#15)

### Features

* add AlloyDB refresher ([#2](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/2)) ([d0d6a11](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/d0d6a119fcb3cc5613de065a168f415dbce70789))
* add support for dialer ([#4](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/4)) ([483ffda](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/483ffdae1870835db79aa04c59a6322b9ec8e9bb))
* add WithUserAgent opt ([#10](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/10)) ([6582164](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/658216477813b92aadfd44403b9389dcaea9f081))
* switch to prod endpoint ([#13](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/13)) ([b477122](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/b47712202088e43533820c51633dff65fe552ce4))
* use v1beta endpoint ([#16](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/16)) ([bfe5fe5](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/bfe5fe56294c76bf7be4ad1ba09cc7b982479d24))


### Bug Fixes

* admin API client handles non-20x responses ([#14](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/14)) ([c2f5dc9](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/c2f5dc92e1a57262c10cd715fc6082a931d0cf70))
* prevent memory leak in driver ([#22](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/22)) ([861d798](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/861d798e0715f16b88d501950a8d9a0493cc8257))
* use instance uri instead of conn name ([#15](https://github.com/GoogleCloudPlatform/alloydb-go-connector/issues/15)) ([0da01fd](https://github.com/GoogleCloudPlatform/alloydb-go-connector/commit/0da01fd311f1e8829be0a9eb0efdeb169ee7c555))
