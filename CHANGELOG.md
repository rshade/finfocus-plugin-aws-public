# Changelog

## [0.0.16](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.15...v0.0.16) (2026-01-08)


### Features

* **actual:** implement runtime-based actual cost from Pulumi metadata ([#217](https://github.com/rshade/pulumicost-plugin-aws-public/issues/217)) ([aa8a951](https://github.com/rshade/pulumicost-plugin-aws-public/commit/aa8a9519018cee51620ba6743280c40337143d36)), closes [#196](https://github.com/rshade/pulumicost-plugin-aws-public/issues/196)
* **carbon:** implement comprehensive carbon estimation for AWS resources ([#220](https://github.com/rshade/pulumicost-plugin-aws-public/issues/220)) ([996aaca](https://github.com/rshade/pulumicost-plugin-aws-public/commit/996aaca531e01180b32a386a16831f03360a834e)), closes [#135](https://github.com/rshade/pulumicost-plugin-aws-public/issues/135) [#136](https://github.com/rshade/pulumicost-plugin-aws-public/issues/136) [#137](https://github.com/rshade/pulumicost-plugin-aws-public/issues/137) [#138](https://github.com/rshade/pulumicost-plugin-aws-public/issues/138) [#139](https://github.com/rshade/pulumicost-plugin-aws-public/issues/139) [#140](https://github.com/rshade/pulumicost-plugin-aws-public/issues/140)
* **cloudwatch:** implement CloudWatch cost estimation ([#210](https://github.com/rshade/pulumicost-plugin-aws-public/issues/210)) ([f95f4ad](https://github.com/rshade/pulumicost-plugin-aws-public/commit/f95f4ad7a221d392515922deb0ec9cd24cbb60a3)), closes [#19](https://github.com/rshade/pulumicost-plugin-aws-public/issues/19)
* **dynamodb:** add validation, error handling, and integration tests ([#218](https://github.com/rshade/pulumicost-plugin-aws-public/issues/218)) ([e87e640](https://github.com/rshade/pulumicost-plugin-aws-public/commit/e87e640077a8ff8fdde9edb796ff66be03898d06)), closes [#147](https://github.com/rshade/pulumicost-plugin-aws-public/issues/147) [#149](https://github.com/rshade/pulumicost-plugin-aws-public/issues/149) [#151](https://github.com/rshade/pulumicost-plugin-aws-public/issues/151) [#152](https://github.com/rshade/pulumicost-plugin-aws-public/issues/152)
* **elasticache:** implement cost estimation for ElastiCache clusters ([#219](https://github.com/rshade/pulumicost-plugin-aws-public/issues/219)) ([f483015](https://github.com/rshade/pulumicost-plugin-aws-public/commit/f483015bc4a00922c252e82659cdc03cb6cb03c3))
* **natgw:** implement VPC NAT Gateway cost estimation ([#195](https://github.com/rshade/pulumicost-plugin-aws-public/issues/195)) ([cce77ab](https://github.com/rshade/pulumicost-plugin-aws-public/commit/cce77abe0e34cab60b230a0d8d3ccc6e2c27ea38)), closes [#56](https://github.com/rshade/pulumicost-plugin-aws-public/issues/56)
* **plugin:** add FOCUS 1.2 cost records, RDS recommendations, and pricing specs ([313c322](https://github.com/rshade/pulumicost-plugin-aws-public/commit/313c322e891b6b299e462e4ac028ef7558ad68ae))


### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-plugin-aws-public to v0.0.15 ([#204](https://github.com/rshade/pulumicost-plugin-aws-public/issues/204)) ([c57b000](https://github.com/rshade/pulumicost-plugin-aws-public/commit/c57b000eee9390517177df9235e954970e27d67a))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.12 ([#200](https://github.com/rshade/pulumicost-plugin-aws-public/issues/200)) ([c99bc97](https://github.com/rshade/pulumicost-plugin-aws-public/commit/c99bc973d6593cad8a04ca010e627e0fd0ed460c))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.13 ([#230](https://github.com/rshade/pulumicost-plugin-aws-public/issues/230)) ([b623e8d](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b623e8df9c9a24045e91b1d056d6db0679debd40))

## [0.0.15](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.14...v0.0.15) (2025-12-26)


### Features

* **recommendations:** pass through ResourceDescriptor.Id to recommendations ([#199](https://github.com/rshade/pulumicost-plugin-aws-public/issues/199)) ([2b9f3fd](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2b9f3fde72a621b40d04f0ba10eae78ea8482d69)), closes [#198](https://github.com/rshade/pulumicost-plugin-aws-public/issues/198)


### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-plugin-aws-public to v0.0.14 ([#189](https://github.com/rshade/pulumicost-plugin-aws-public/issues/189)) ([adf84ee](https://github.com/rshade/pulumicost-plugin-aws-public/commit/adf84ee7caf247abab69e08c4154280d5e0cd38a))
* **deps:** update module google.golang.org/grpc to v1.78.0 ([#194](https://github.com/rshade/pulumicost-plugin-aws-public/issues/194)) ([df698dc](https://github.com/rshade/pulumicost-plugin-aws-public/commit/df698dc396abc6697a754e532c3ae7f4d2f20a69))


### Documentation

* comprehensive documentation improvements and fixes ([#193](https://github.com/rshade/pulumicost-plugin-aws-public/issues/193)) ([ca93f63](https://github.com/rshade/pulumicost-plugin-aws-public/commit/ca93f633c3a2f8a9a4b1e5b63c0e90679fb182e8))

## [0.0.14](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.13...v0.0.14) (2025-12-21)

### Bug Fixes

* **release:** correct verify script to match actual archive naming pattern ([d46ccd3](https://github.com/rshade/pulumicost-plugin-aws-public/commit/d46ccd36c8d31a1868a81bcea6f1cd87b7993a0e))

### Documentation

* updating claude.md for size issues ([#187](https://github.com/rshade/pulumicost-plugin-aws-public/issues/187)) ([2bfd8d9](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2bfd8d9059a6b607d1973ee4622505a77d52e725))

## [0.0.13](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.12...v0.0.13) (2025-12-21)

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-plugin-aws-public to v0.0.12 ([#173](https://github.com/rshade/pulumicost-plugin-aws-public/issues/173)) ([a1b7dc7](https://github.com/rshade/pulumicost-plugin-aws-public/commit/a1b7dc73d5704e031daa1143ffc4073827246303))

### Miscellaneous

* **pricing:** embed raw AWS pricing JSON per service ([#172](https://github.com/rshade/pulumicost-plugin-aws-public/issues/172)) ([b6898f1](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b6898f1b31413bec04eb8eb8ed3e49f24ada0c32))

## [0.0.12](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.11...v0.0.12) (2025-12-21)

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-plugin-aws-public to v0.0.11 ([#167](https://github.com/rshade/pulumicost-plugin-aws-public/issues/167)) ([820b46b](https://github.com/rshade/pulumicost-plugin-aws-public/commit/820b46b3153ddc7e29b7736c2b1843a2720fc1aa))
* **elb:** address CodeRabbit review comments for ELB cost estimation ([#168](https://github.com/rshade/pulumicost-plugin-aws-public/issues/168)) ([f3e7cea](https://github.com/rshade/pulumicost-plugin-aws-public/commit/f3e7cea9d12543e63484cec35c110cd1bd1f8883))

## [0.0.11](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.10...v0.0.11) (2025-12-20)

### Features

* **elb:** implement Elastic Load Balancing (ALB/NLB) cost estimation ([#154](https://github.com/rshade/pulumicost-plugin-aws-public/issues/154)) ([62989a0](https://github.com/rshade/pulumicost-plugin-aws-public/commit/62989a0b0245d05c771c9317db1269776f674dcf)), closes [#017](https://github.com/rshade/pulumicost-plugin-aws-public/issues/017)

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-plugin-aws-public to v0.0.10 ([#148](https://github.com/rshade/pulumicost-plugin-aws-public/issues/148)) ([e8402cd](https://github.com/rshade/pulumicost-plugin-aws-public/commit/e8402cd555f312ce658b54ab308f1b01eb408060))
* prevent panic in recommendations batch processing and improve validation ([#153](https://github.com/rshade/pulumicost-plugin-aws-public/issues/153)) ([84c2b82](https://github.com/rshade/pulumicost-plugin-aws-public/commit/84c2b825eb1167508a433461a7e676e72b1a4ecd))

## [0.0.10](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.9...v0.0.10) (2025-12-20)

### Features

* **carbon:** implement carbon emission estimation for EC2 instances ([#132](https://github.com/rshade/pulumicost-plugin-aws-public/issues/132)) ([6de8039](https://github.com/rshade/pulumicost-plugin-aws-public/commit/6de80391c5b483a907d0d5851609b9a7daacb3fa))
* **dynamodb:** implement DynamoDB On-Demand and Provisioned cost est… ([#141](https://github.com/rshade/pulumicost-plugin-aws-public/issues/141)) ([5d5814d](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5d5814ddc1789fcd7ce8dae3ce809a361a0983bf))
* **lambda:** implement Lambda function cost estimation ([#121](https://github.com/rshade/pulumicost-plugin-aws-public/issues/121)) ([193cc41](https://github.com/rshade/pulumicost-plugin-aws-public/commit/193cc4184687516cd56e07b038b2772047c9cfa1)), closes [#53](https://github.com/rshade/pulumicost-plugin-aws-public/issues/53)
* **recommendations:** support target_resources batch processing ([#122](https://github.com/rshade/pulumicost-plugin-aws-public/issues/122)) ([80165f6](https://github.com/rshade/pulumicost-plugin-aws-public/commit/80165f69b75864b84bde51b6568f323be0ada09d)), closes [#118](https://github.com/rshade/pulumicost-plugin-aws-public/issues/118)

### Bug Fixes

* **deps:** update module github.com/goccy/go-yaml to v1.19.1 ([#108](https://github.com/rshade/pulumicost-plugin-aws-public/issues/108)) ([5e3587a](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5e3587ad913a39d858d3741484676a7cdc1c388c))
* **deps:** update module github.com/rshade/pulumicost-plugin-aws-public to v0.0.9 ([#117](https://github.com/rshade/pulumicost-plugin-aws-public/issues/117)) ([0e8fa71](https://github.com/rshade/pulumicost-plugin-aws-public/commit/0e8fa710c60e1cd7d54fbd3530e0b10aeacd91b7))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.10 ([#131](https://github.com/rshade/pulumicost-plugin-aws-public/issues/131)) ([de00623](https://github.com/rshade/pulumicost-plugin-aws-public/commit/de006236e7c5439404a0904de5ecdc9d582b53a0))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.9 ([#119](https://github.com/rshade/pulumicost-plugin-aws-public/issues/119)) ([861b171](https://github.com/rshade/pulumicost-plugin-aws-public/commit/861b17175b2f89f3aded3747e0b95c73b9437083))

## [0.0.9](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.8...v0.0.9) (2025-12-17)

### Features

* **finops:** implement GetRecommendations RPC for cost optimization ([#106](https://github.com/rshade/pulumicost-plugin-aws-public/issues/106)) ([deb8eff](https://github.com/rshade/pulumicost-plugin-aws-public/commit/deb8effc4cd16cbd1add1c32e6757a5317a7cfbc)), closes [#105](https://github.com/rshade/pulumicost-plugin-aws-public/issues/105)
* **s3:** implement S3 storage cost estimation ([#99](https://github.com/rshade/pulumicost-plugin-aws-public/issues/99)) ([06167dc](https://github.com/rshade/pulumicost-plugin-aws-public/commit/06167dc150c5119ecb18d08b1b546a482e9ecfee))

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.7 ([#100](https://github.com/rshade/pulumicost-plugin-aws-public/issues/100)) ([b623aa3](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b623aa30c5b6715e14a4226b78f668b5f6957a07))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.8 ([#107](https://github.com/rshade/pulumicost-plugin-aws-public/issues/107)) ([a9a3360](https://github.com/rshade/pulumicost-plugin-aws-public/commit/a9a33605c9125b7b1e43a81afe96b57aca22675c))
* **deps:** update module google.golang.org/protobuf to v1.36.11 ([#101](https://github.com/rshade/pulumicost-plugin-aws-public/issues/101)) ([463e44a](https://github.com/rshade/pulumicost-plugin-aws-public/commit/463e44ae649bc89a2e558bd047dd7fcfe5bc8b2c))
* **resource:** Support Pulumi resource type format ([#97](https://github.com/rshade/pulumicost-plugin-aws-public/issues/97)) ([2fb4008](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2fb4008af3908522654171939eabd02e4a796562)), closes [#82](https://github.com/rshade/pulumicost-plugin-aws-public/issues/82)

## [0.0.8](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.7...v0.0.8) (2025-12-07)

### Features

* **eks:** add EKS pricing validation and missing pricing test coverage ([#93](https://github.com/rshade/pulumicost-plugin-aws-public/issues/93)) ([de697c4](https://github.com/rshade/pulumicost-plugin-aws-public/commit/de697c49f36ddf6eb6e16c313e8582e38ed8ed52)), closes [#90](https://github.com/rshade/pulumicost-plugin-aws-public/issues/90)

### Bug Fixes

* **eks:** resolve pricing parser zero-value bug and case-sensitive support_type ([#95](https://github.com/rshade/pulumicost-plugin-aws-public/issues/95)) ([8b24250](https://github.com/rshade/pulumicost-plugin-aws-public/commit/8b24250104126abc9dc6e08d144f73220883072c)), closes [#89](https://github.com/rshade/pulumicost-plugin-aws-public/issues/89)

### Documentation

* **eks:** clarify EKS cost estimation scope in CLAUDE.md ([#92](https://github.com/rshade/pulumicost-plugin-aws-public/issues/92)) ([bac39ac](https://github.com/rshade/pulumicost-plugin-aws-public/commit/bac39ac9bb318d339f1c4f6dd662775bd6a23015))

## [0.0.7](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.6...v0.0.7) (2025-12-07)

### Features

* **eks:** implement EKS cluster cost estimation ([#76](https://github.com/rshade/pulumicost-plugin-aws-public/issues/76)) ([cc5a19a](https://github.com/rshade/pulumicost-plugin-aws-public/commit/cc5a19accba384c05397a77c9c8d65594192a825))

### Bug Fixes

* **resource:** enhance resource type compatibility ([#81](https://github.com/rshade/pulumicost-plugin-aws-public/issues/81)) ([74fd00d](https://github.com/rshade/pulumicost-plugin-aws-public/commit/74fd00dcf7cfc77b08c26574285eb750e58f4379))

## [0.0.6](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.5...v0.0.6) (2025-12-06)

### Bug Fixes

* **release:** fix automated release workflow for GoReleaser v2 ([84fc43e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/84fc43e7ed052237c14e04e286de5af6cc2c2140))
* **release:** remove verify-regions step blocking automated releases ([ebfe7e3](https://github.com/rshade/pulumicost-plugin-aws-public/commit/ebfe7e3436afe964070a0f614f1c3190299d4386))

## [0.0.5](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.4...v0.0.5) (2025-12-06)

### Features

* add support for 4 Asia Pacific AWS regions ([#19](https://github.com/rshade/pulumicost-plugin-aws-public/issues/19)) ([1c19ca5](https://github.com/rshade/pulumicost-plugin-aws-public/commit/1c19ca5cb9f557399068f7daea1405b25b5be984)), closes [#1](https://github.com/rshade/pulumicost-plugin-aws-public/issues/1)
* add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1) ([#46](https://github.com/rshade/pulumicost-plugin-aws-public/issues/46)) ([ce71fd4](https://github.com/rshade/pulumicost-plugin-aws-public/commit/ce71fd45f35379ca9f8db86f12ace007f54950de)), closes [#4](https://github.com/rshade/pulumicost-plugin-aws-public/issues/4)
* automate region build matrix ([#49](https://github.com/rshade/pulumicost-plugin-aws-public/issues/49)) ([8003dcf](https://github.com/rshade/pulumicost-plugin-aws-public/commit/8003dcff87680c42255c5a6ebb0092389a5b0ed5))
* **build:** replace sed/awk YAML parsing with Go-based parser ([#72](https://github.com/rshade/pulumicost-plugin-aws-public/issues/72)) ([df27421](https://github.com/rshade/pulumicost-plugin-aws-public/commit/df27421015168860c140beeedb7b6394d3ac29b6))
* implement AWS public pricing plugin with gRPC interface ([5f1de2e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5f1de2edd0851519cd0998ce077358a65a3eb3d2))
* implement fallback GetActualCost using runtime × list price ([#34](https://github.com/rshade/pulumicost-plugin-aws-public/issues/34)) ([25122b2](https://github.com/rshade/pulumicost-plugin-aws-public/commit/25122b2a599083d4e324c9815283689219fc0b53)), closes [#24](https://github.com/rshade/pulumicost-plugin-aws-public/issues/24)
* implement Zerolog Structured Logging with Trace Propagation ([#39](https://github.com/rshade/pulumicost-plugin-aws-public/issues/39)) ([8ab8037](https://github.com/rshade/pulumicost-plugin-aws-public/commit/8ab803714fa9ab6fe96d09adb2a6dd807eba45f2))
* MVP implementation - AWS public pricing plugin ([b093949](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b093949bb5dab85ad95f3fc415e5d30b948ca887))
* **pricing:** add Canada and South America regions with real AWS pricing ([#43](https://github.com/rshade/pulumicost-plugin-aws-public/issues/43)) ([2406c34](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2406c34e48ed975358d77b52240c901ed1a1f710))

### Bug Fixes

* **deps:** update github.com/rshade/pulumicost-core digest to 4680d9c ([#18](https://github.com/rshade/pulumicost-plugin-aws-public/issues/18)) ([38f0bde](https://github.com/rshade/pulumicost-plugin-aws-public/commit/38f0bdea8ce2b3d119372a097b3872f1b027a769))
* **deps:** update github.com/rshade/pulumicost-core digest to b2ad29f ([#11](https://github.com/rshade/pulumicost-plugin-aws-public/issues/11)) ([859d4d1](https://github.com/rshade/pulumicost-plugin-aws-public/commit/859d4d1fdda7fb36a51cda6b4b0f983f8eb1fad6))
* **deps:** update github.com/rshade/pulumicost-core digest to c93f761 ([#21](https://github.com/rshade/pulumicost-plugin-aws-public/issues/21)) ([060cb63](https://github.com/rshade/pulumicost-plugin-aws-public/commit/060cb6316d28d21fd2ff788e8eff5327e7f17a8c))
* **deps:** update module github.com/goccy/go-yaml to v1.19.0 ([#61](https://github.com/rshade/pulumicost-plugin-aws-public/issues/61)) ([e83e42b](https://github.com/rshade/pulumicost-plugin-aws-public/commit/e83e42b50bc0083a1adb55c887f48a104d666eb3))
* **deps:** update module github.com/rshade/pulumicost-core to v0.1.0 ([#32](https://github.com/rshade/pulumicost-plugin-aws-public/issues/32)) ([3477911](https://github.com/rshade/pulumicost-plugin-aws-public/commit/3477911cb7150a81eeef979b110874f71ba5c695))
* **deps:** update module github.com/rshade/pulumicost-core to v0.1.1 ([#41](https://github.com/rshade/pulumicost-plugin-aws-public/issues/41)) ([251f432](https://github.com/rshade/pulumicost-plugin-aws-public/commit/251f4322e0cb9af7b444cb96e02fe5d9040eafe7))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.3.0 ([#12](https://github.com/rshade/pulumicost-plugin-aws-public/issues/12)) ([e4d435d](https://github.com/rshade/pulumicost-plugin-aws-public/commit/e4d435d5ca86ab9402f272fd44c362a5eef7246f))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.0 ([#37](https://github.com/rshade/pulumicost-plugin-aws-public/issues/37)) ([997ca6c](https://github.com/rshade/pulumicost-plugin-aws-public/commit/997ca6c92d476130703683aea7d417df5bfb7a27))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.1 ([#40](https://github.com/rshade/pulumicost-plugin-aws-public/issues/40)) ([5de522e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5de522ec3acaddae79dbc3eb5b28c614e326c02a))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.2 ([#47](https://github.com/rshade/pulumicost-plugin-aws-public/issues/47)) ([21dbb97](https://github.com/rshade/pulumicost-plugin-aws-public/commit/21dbb97f4ccc7a0ea23f535128cc97d87eaa74e2))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.3 ([#69](https://github.com/rshade/pulumicost-plugin-aws-public/issues/69)) ([b56c439](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b56c439873dec68432d6ff621308e90663692d1d))

### Documentation

* clarify zerolog logging requirements in constitution v2.1.1 ([#74](https://github.com/rshade/pulumicost-plugin-aws-public/issues/74)) ([88e8d2f](https://github.com/rshade/pulumicost-plugin-aws-public/commit/88e8d2f585a73f26fe22bf28903c931ebd43f7db))
* updating the constitution ([0c15505](https://github.com/rshade/pulumicost-plugin-aws-public/commit/0c1550548c65d2646f7d30243c19f0246220297a))

## [0.0.4](https://github.com/rshade/pulumicost-plugin-aws-public/compare/pulumicost-plugin-aws-public-v0.0.3...pulumicost-plugin-aws-public-v0.0.4) (2025-11-30)

### Features

* **pricing:** add Canada and South America regions with real AWS pricing ([#43](https://github.com/rshade/pulumicost-plugin-aws-public/issues/43)) ([2406c34](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2406c34e48ed975358d77b52240c901ed1a1f710))

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-core to v0.1.1 ([#41](https://github.com/rshade/pulumicost-plugin-aws-public/issues/41)) ([251f432](https://github.com/rshade/pulumicost-plugin-aws-public/commit/251f4322e0cb9af7b444cb96e02fe5d9040eafe7))

## [0.0.3](https://github.com/rshade/pulumicost-plugin-aws-public/compare/pulumicost-plugin-aws-public-v0.0.2...pulumicost-plugin-aws-public-v0.0.3) (2025-11-29)

### Features

* implement Zerolog Structured Logging with Trace Propagation ([#39](https://github.com/rshade/pulumicost-plugin-aws-public/issues/39)) ([8ab8037](https://github.com/rshade/pulumicost-plugin-aws-public/commit/8ab803714fa9ab6fe96d09adb2a6dd807eba45f2))

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.0 ([#37](https://github.com/rshade/pulumicost-plugin-aws-public/issues/37)) ([997ca6c](https://github.com/rshade/pulumicost-plugin-aws-public/commit/997ca6c92d476130703683aea7d417df5bfb7a27))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.4.1 ([#40](https://github.com/rshade/pulumicost-plugin-aws-public/issues/40)) ([5de522e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5de522ec3acaddae79dbc3eb5b28c614e326c02a))

## [0.0.2](https://github.com/rshade/pulumicost-plugin-aws-public/compare/pulumicost-plugin-aws-public-v0.0.1...pulumicost-plugin-aws-public-v0.0.2) (2025-11-26)

### Features

* implement fallback GetActualCost using runtime × list price ([#34](https://github.com/rshade/pulumicost-plugin-aws-public/issues/34)) ([25122b2](https://github.com/rshade/pulumicost-plugin-aws-public/commit/25122b2a599083d4e324c9815283689219fc0b53)), closes [#24](https://github.com/rshade/pulumicost-plugin-aws-public/issues/24)

### Bug Fixes

* **deps:** update module github.com/rshade/pulumicost-core to v0.1.0 ([#32](https://github.com/rshade/pulumicost-plugin-aws-public/issues/32)) ([3477911](https://github.com/rshade/pulumicost-plugin-aws-public/commit/3477911cb7150a81eeef979b110874f71ba5c695))

## [0.0.1](https://github.com/rshade/pulumicost-plugin-aws-public/compare/pulumicost-plugin-aws-public-v0.0.1...pulumicost-plugin-aws-public-v0.0.1) (2025-11-26)

### Features

* add support for 4 Asia Pacific AWS regions ([#19](https://github.com/rshade/pulumicost-plugin-aws-public/issues/19)) ([1c19ca5](https://github.com/rshade/pulumicost-plugin-aws-public/commit/1c19ca5cb9f557399068f7daea1405b25b5be984)), closes [#1](https://github.com/rshade/pulumicost-plugin-aws-public/issues/1)
* implement AWS public pricing plugin with gRPC interface ([5f1de2e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5f1de2edd0851519cd0998ce077358a65a3eb3d2))
* MVP implementation - AWS public pricing plugin ([b093949](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b093949bb5dab85ad95f3fc415e5d30b948ca887))

### Bug Fixes

* **deps:** update github.com/rshade/pulumicost-core digest to 4680d9c ([#18](https://github.com/rshade/pulumicost-plugin-aws-public/issues/18)) ([38f0bde](https://github.com/rshade/pulumicost-plugin-aws-public/commit/38f0bdea8ce2b3d119372a097b3872f1b027a769))
* **deps:** update github.com/rshade/pulumicost-core digest to b2ad29f ([#11](https://github.com/rshade/pulumicost-plugin-aws-public/issues/11)) ([859d4d1](https://github.com/rshade/pulumicost-plugin-aws-public/commit/859d4d1fdda7fb36a51cda6b4b0f983f8eb1fad6))
* **deps:** update github.com/rshade/pulumicost-core digest to c93f761 ([#21](https://github.com/rshade/pulumicost-plugin-aws-public/issues/21)) ([060cb63](https://github.com/rshade/pulumicost-plugin-aws-public/commit/060cb6316d28d21fd2ff788e8eff5327e7f17a8c))
* **deps:** update module github.com/rshade/pulumicost-spec to v0.3.0 ([#12](https://github.com/rshade/pulumicost-plugin-aws-public/issues/12)) ([e4d435d](https://github.com/rshade/pulumicost-plugin-aws-public/commit/e4d435d5ca86ab9402f272fd44c362a5eef7246f))

### Documentation

* updating the constitution ([0c15505](https://github.com/rshade/pulumicost-plugin-aws-public/commit/0c1550548c65d2646f7d30243c19f0246220297a))
