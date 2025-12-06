# Changelog

## [0.0.6](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.5...v0.0.6) (2025-12-06)


### Bug Fixes

* **release:** fix automated release workflow for GoReleaser v2 ([84fc43e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/84fc43e7ed052237c14e04e286de5af6cc2c2140))
* **release:** remove verify-regions step blocking automated releases ([ebfe7e3](https://github.com/rshade/pulumicost-plugin-aws-public/commit/ebfe7e3436afe964070a0f614f1c3190299d4386))

## [0.0.5](https://github.com/rshade/pulumicost-plugin-aws-public/compare/v0.0.4...v0.0.5) (2025-12-06)


### Features

* add support for 4 Asia Pacific AWS regions ([#19](https://github.com/rshade/pulumicost-plugin-aws-public/issues/19)) ([1c19ca5](https://github.com/rshade/pulumicost-plugin-aws-public/commit/1c19ca5cb9f557399068f7daea1405b25b5be984)), closes [#1](https://github.com/rshade/pulumicost-plugin-aws-public/issues/1)
* add support for additional US regions (us-west-1, us-gov-west-1, us-gov-east-1) ([#46](https://github.com/rshade/pulumicost-plugin-aws-public/issues/46)) ([ce71fd4](https://github.com/rshade/pulumicost-plugin-aws-public/commit/ce71fd45f35379ca9f8db86f12ace007f54950de)), closes [#4](https://github.com/rshade/pulumicost-plugin-aws-public/issues/4)
* automate region build matrix ([#49](https://github.com/rshade/pulumicost-plugin-aws-public/issues/49)) ([8003dcf](https://github.com/rshade/pulumicost-plugin-aws-public/commit/8003dcff87680c42255c5a6ebb0092389a5b0ed5))
* **build:** replace sed/awk YAML parsing with Go-based parser  ([#72](https://github.com/rshade/pulumicost-plugin-aws-public/issues/72)) ([df27421](https://github.com/rshade/pulumicost-plugin-aws-public/commit/df27421015168860c140beeedb7b6394d3ac29b6))
* implement AWS public pricing plugin with gRPC interface ([5f1de2e](https://github.com/rshade/pulumicost-plugin-aws-public/commit/5f1de2edd0851519cd0998ce077358a65a3eb3d2))
* implement fallback GetActualCost using runtime × list price ([#34](https://github.com/rshade/pulumicost-plugin-aws-public/issues/34)) ([25122b2](https://github.com/rshade/pulumicost-plugin-aws-public/commit/25122b2a599083d4e324c9815283689219fc0b53)), closes [#24](https://github.com/rshade/pulumicost-plugin-aws-public/issues/24)
* implement Zerolog Structured Logging with Trace Propagation ([#39](https://github.com/rshade/pulumicost-plugin-aws-public/issues/39)) ([8ab8037](https://github.com/rshade/pulumicost-plugin-aws-public/commit/8ab803714fa9ab6fe96d09adb2a6dd807eba45f2))
* MVP implementation - AWS public pricing plugin ([b093949](https://github.com/rshade/pulumicost-plugin-aws-public/commit/b093949bb5dab85ad95f3fc415e5d30b948ca887))
* **pricing:** add Canada and South America regions with real AWS pri… ([#43](https://github.com/rshade/pulumicost-plugin-aws-public/issues/43)) ([2406c34](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2406c34e48ed975358d77b52240c901ed1a1f710))


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
* updating the coonstitution ([0c15505](https://github.com/rshade/pulumicost-plugin-aws-public/commit/0c1550548c65d2646f7d30243c19f0246220297a))

## [0.0.4](https://github.com/rshade/pulumicost-plugin-aws-public/compare/pulumicost-plugin-aws-public-v0.0.3...pulumicost-plugin-aws-public-v0.0.4) (2025-11-30)


### Features

* **pricing:** add Canada and South America regions with real AWS pri… ([#43](https://github.com/rshade/pulumicost-plugin-aws-public/issues/43)) ([2406c34](https://github.com/rshade/pulumicost-plugin-aws-public/commit/2406c34e48ed975358d77b52240c901ed1a1f710))


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

* updating the coonstitution ([0c15505](https://github.com/rshade/pulumicost-plugin-aws-public/commit/0c1550548c65d2646f7d30243c19f0246220297a))
