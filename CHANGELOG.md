# [0.18.0](https://github.com/Belphemur/night-routine/compare/v0.17.0...v0.18.0) (2025-11-24)


### Bug Fixes

* **css:** implement automatic cache busting ([802a555](https://github.com/Belphemur/night-routine/commit/802a5553132420d2faed5abe89ebbae314f2d86b))
* **css:** rebuild css ([2a90129](https://github.com/Belphemur/night-routine/commit/2a9012927ea876e7785b0e81dc914291ddaa5642))


### Features

* **statistics:** implement mobile-friendly card view with month grouping ([3e6dada](https://github.com/Belphemur/night-routine/commit/3e6dadac72c6bea271a128c41096633694482bea))

# [0.17.0](https://github.com/Belphemur/night-routine/compare/v0.16.0...v0.17.0) (2025-11-24)


### Features

* **ui:** add modal animations using tailwind ([6c7b35b](https://github.com/Belphemur/night-routine/commit/6c7b35b902ec6567462dbc46d146888c374335f2))

# [0.16.0](https://github.com/Belphemur/night-routine/compare/v0.15.7...v0.16.0) (2025-11-21)


### Features

* **home:** display calendar name with ID as subtitle ([#163](https://github.com/Belphemur/night-routine/issues/163)) ([a4f0e56](https://github.com/Belphemur/night-routine/commit/a4f0e56e7a08eb51354cefa64f5af1e0c2840113))

## [0.15.7](https://github.com/Belphemur/night-routine/compare/v0.15.6...v0.15.7) (2025-11-21)


### Bug Fixes

* **handlers:** set 12h cache with revalidation for static CSS ([#161](https://github.com/Belphemur/night-routine/issues/161)) ([c0aea2d](https://github.com/Belphemur/night-routine/commit/c0aea2d83bcfa90c204b9f06119daae71d42cde6))

## [0.15.6](https://github.com/Belphemur/night-routine/compare/v0.15.5...v0.15.6) (2025-11-21)


### Bug Fixes

* **ux:** Fix text overflow and dialog positioning issues ([#159](https://github.com/Belphemur/night-routine/issues/159)) ([7a9a8dd](https://github.com/Belphemur/night-routine/commit/7a9a8dd719a851e8a321812a42b68ffa6ddd9c6a))

## [0.15.5](https://github.com/Belphemur/night-routine/compare/v0.15.4...v0.15.5) (2025-11-21)


### Bug Fixes

* Add ETag support for static CSS file serving ([#156](https://github.com/Belphemur/night-routine/issues/156)) ([231d632](https://github.com/Belphemur/night-routine/commit/231d632ae3871c8d56db4d6fba6fbf2edfe3658b))
* replace custom unlock modal with native HTML dialog element ([7b98466](https://github.com/Belphemur/night-routine/commit/7b984669ebaf4f159308f4fd1e629bd583225c6a))

## [0.15.4](https://github.com/Belphemur/night-routine/compare/v0.15.3...v0.15.4) (2025-11-20)


### Bug Fixes

* **today:** fix today styling to use proper tailwind css ([#153](https://github.com/Belphemur/night-routine/issues/153)) ([3bcb219](https://github.com/Belphemur/night-routine/commit/3bcb219dc18c48ea9c555172a48adc9191673ac6))

## [0.15.3](https://github.com/Belphemur/night-routine/compare/v0.15.2...v0.15.3) (2025-11-20)


### Bug Fixes

* Replace inline styles with Tailwind ring utilities for highlighting today's date in the calendar. ([#152](https://github.com/Belphemur/night-routine/issues/152)) ([cf39584](https://github.com/Belphemur/night-routine/commit/cf39584c353973b25bcaabc3e20b62883461014a))

## [0.15.2](https://github.com/Belphemur/night-routine/compare/v0.15.1...v0.15.2) (2025-11-20)


### Bug Fixes

* have proper highlight for the current day ([#151](https://github.com/Belphemur/night-routine/issues/151)) ([4d3882b](https://github.com/Belphemur/night-routine/commit/4d3882b745bad3533f8088b36cc94684feb274be))

## [0.15.1](https://github.com/Belphemur/night-routine/compare/v0.15.0...v0.15.1) (2025-11-20)


### Bug Fixes

* add robust validation and tests for unlocking assignments, refactor unlock handler dependencies, and refine home page UI. ([#150](https://github.com/Belphemur/night-routine/issues/150)) ([7c8075a](https://github.com/Belphemur/night-routine/commit/7c8075a175c528016647c607f46ec247262c8abd))

# [0.15.0](https://github.com/Belphemur/night-routine/compare/v0.14.1...v0.15.0) (2025-11-20)


### Features

* **ui:** Responsive UI/UX using Tailwind CSS ([e6db0bc](https://github.com/Belphemur/night-routine/commit/e6db0bc183095a6373e6b329b3ec2ccca9be7364))

## [0.14.1](https://github.com/Belphemur/night-routine/compare/v0.14.0...v0.14.1) (2025-11-20)


### Bug Fixes

* Add mobile-responsive UI with production Tailwind CSS build and accessibility improvements ([#144](https://github.com/Belphemur/night-routine/issues/144)) ([172c10b](https://github.com/Belphemur/night-routine/commit/172c10b9ca38a4fecf0680285f76b0a154a0ecd7))

# [0.14.0](https://github.com/Belphemur/night-routine/compare/v0.13.0...v0.14.0) (2025-11-19)


### Bug Fixes

* **decision:** clean up the decision reason when unlocking ([de4f511](https://github.com/Belphemur/night-routine/commit/de4f5117cc552a544e2c66841d1c5ebb71703a81))
* **home:** clean up the formatting ([23f0467](https://github.com/Belphemur/night-routine/commit/23f0467c95fd3d1744ee14bca015204c43103ae2))
* **home:** Enhance calendar styling with decision reasons, tooltips, and dynamic current day highlight. ([0e35de9](https://github.com/Belphemur/night-routine/commit/0e35de9174eb2bb580c600c9682f1a892dc9eb0b))
* **home:** fix home template and remove isToday ([82d84d6](https://github.com/Belphemur/night-routine/commit/82d84d6fd3a95de5d7a84a5d7591325ef969574c))


### Features

* Add assignment unlock functionality and centralize authentication logic in base handler. ([a06d919](https://github.com/Belphemur/night-routine/commit/a06d919567fa59d6b94656a7ecaa0a65af9bcc70))
* **modal:** have a good modal for confirming unlocking event ([f6d9395](https://github.com/Belphemur/night-routine/commit/f6d9395bac74510899e3f9f3dccda2b182b181f8))

# [0.13.0](https://github.com/Belphemur/night-routine/compare/v0.12.8...v0.13.0) (2025-11-19)


### Features

* settings menu ([#139](https://github.com/Belphemur/night-routine/issues/139)) ([017bbb2](https://github.com/Belphemur/night-routine/commit/017bbb215710d5ea45d50793d7b3fb93fc33da33))

## [0.12.8](https://github.com/Belphemur/night-routine/compare/v0.12.7...v0.12.8) (2025-11-11)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.256.0 ([#132](https://github.com/Belphemur/night-routine/issues/132)) ([b36ea8b](https://github.com/Belphemur/night-routine/commit/b36ea8b5e84e5332d9800982c4b254339cb2b0bd))

## [0.12.7](https://github.com/Belphemur/night-routine/compare/v0.12.6...v0.12.7) (2025-11-09)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.30.1 ([#131](https://github.com/Belphemur/night-routine/issues/131)) ([0199cc2](https://github.com/Belphemur/night-routine/commit/0199cc2f873036db4cf77f201c865d0a40267c1e))

## [0.12.6](https://github.com/Belphemur/night-routine/compare/v0.12.5...v0.12.6) (2025-11-08)


### Bug Fixes

* **deps:** update module golang.org/x/oauth2 to v0.33.0 ([#130](https://github.com/Belphemur/night-routine/issues/130)) ([8b36bb6](https://github.com/Belphemur/night-routine/commit/8b36bb6099f91bdc4394572214bf2b13f5b8e35b))

## [0.12.5](https://github.com/Belphemur/night-routine/compare/v0.12.4...v0.12.5) (2025-11-05)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.30.0 ([#127](https://github.com/Belphemur/night-routine/issues/127)) ([128c103](https://github.com/Belphemur/night-routine/commit/128c103947d3736eb8fc1b3c483dc6bc9910e3cf))

## [0.12.4](https://github.com/Belphemur/night-routine/compare/v0.12.3...v0.12.4) (2025-11-04)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.255.0 ([#126](https://github.com/Belphemur/night-routine/issues/126)) ([2e5c936](https://github.com/Belphemur/night-routine/commit/2e5c936f465f888001fe23f74696ff91db21325f))

## [0.12.3](https://github.com/Belphemur/night-routine/compare/v0.12.2...v0.12.3) (2025-10-29)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.254.0 ([#125](https://github.com/Belphemur/night-routine/issues/125)) ([c49c0ac](https://github.com/Belphemur/night-routine/commit/c49c0ac34a306aa6be477de66a7c3dc185832856))

## [0.12.2](https://github.com/Belphemur/night-routine/compare/v0.12.1...v0.12.2) (2025-10-22)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.253.0 ([#122](https://github.com/Belphemur/night-routine/issues/122)) ([e9b68be](https://github.com/Belphemur/night-routine/commit/e9b68be2a02b00b6b50ec04eddb1252a3be8b9ee))

## [0.12.1](https://github.com/Belphemur/night-routine/compare/v0.12.0...v0.12.1) (2025-10-20)


### Bug Fixes

* Streamline README and point to GitHub Pages for full documentation ([#121](https://github.com/Belphemur/night-routine/issues/121)) ([19e76bb](https://github.com/Belphemur/night-routine/commit/19e76bb7261c8a644ba719e00bfd69662cf53c61))

# [0.12.0](https://github.com/Belphemur/night-routine/compare/v0.11.0...v0.12.0) (2025-10-19)


### Features

* migrate to docker v2 for goreleaser and setup provenance attestation ([#111](https://github.com/Belphemur/night-routine/issues/111)) ([fc8b800](https://github.com/Belphemur/night-routine/commit/fc8b800f6297be22f9afb89427f62cfb61a2c80d))

# [0.11.0](https://github.com/Belphemur/night-routine/compare/v0.10.13...v0.11.0) (2025-10-19)


### Features

* **webhook:** Add configurable threshold for accepting past event changes in webhook handler ([#107](https://github.com/Belphemur/night-routine/issues/107)) ([582bd9a](https://github.com/Belphemur/night-routine/commit/582bd9acc4f875e48c749a0eb5b9d400a42240aa))

## [0.10.13](https://github.com/Belphemur/night-routine/compare/v0.10.12...v0.10.13) (2025-10-08)


### Bug Fixes

* **deps:** update module golang.org/x/oauth2 to v0.32.0 ([#99](https://github.com/Belphemur/night-routine/issues/99)) ([9706030](https://github.com/Belphemur/night-routine/commit/9706030e271a7bf28df2f4bebd70c2a147ef3d33))

## [0.10.12](https://github.com/Belphemur/night-routine/compare/v0.10.11...v0.10.12) (2025-10-07)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.252.0 ([#98](https://github.com/Belphemur/night-routine/issues/98)) ([3ad767e](https://github.com/Belphemur/night-routine/commit/3ad767e768c684e9cb459236f9a8a0a51fa0acf8))

## [0.10.11](https://github.com/Belphemur/night-routine/compare/v0.10.10...v0.10.11) (2025-10-01)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.29.1 ([#97](https://github.com/Belphemur/night-routine/issues/97)) ([c43640c](https://github.com/Belphemur/night-routine/commit/c43640c17512d64cde3cbe4c6118c5981e4edc1f))

## [0.10.10](https://github.com/Belphemur/night-routine/compare/v0.10.9...v0.10.10) (2025-09-30)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.251.0 ([#96](https://github.com/Belphemur/night-routine/issues/96)) ([045fd6c](https://github.com/Belphemur/night-routine/commit/045fd6c2ea370f80c56ee4f6879fee15e805bef7))

## [0.10.9](https://github.com/Belphemur/night-routine/compare/v0.10.8...v0.10.9) (2025-09-28)


### Bug Fixes

* **deps:** update module github.com/maniartech/signals to v1.3.1 ([#95](https://github.com/Belphemur/night-routine/issues/95)) ([c4d3816](https://github.com/Belphemur/night-routine/commit/c4d3816bd88c1c2b3bf04c75e9edc38f927a4417))

## [0.10.8](https://github.com/Belphemur/night-routine/compare/v0.10.7...v0.10.8) (2025-09-25)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.250.0 ([#94](https://github.com/Belphemur/night-routine/issues/94)) ([845d627](https://github.com/Belphemur/night-routine/commit/845d627c795859059ec198b9d4f4e170775d6999))

## [0.10.7](https://github.com/Belphemur/night-routine/compare/v0.10.6...v0.10.7) (2025-09-23)


### Bug Fixes

* **deps:** update module github.com/maniartech/signals to v1.3.0 ([#93](https://github.com/Belphemur/night-routine/issues/93)) ([9827f47](https://github.com/Belphemur/night-routine/commit/9827f479b94be6a490759c29411c6310ec743243))

## [0.10.6](https://github.com/Belphemur/night-routine/compare/v0.10.5...v0.10.6) (2025-09-08)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.249.0 ([#88](https://github.com/Belphemur/night-routine/issues/88)) ([fa80263](https://github.com/Belphemur/night-routine/commit/fa80263fb110a9fcaf7a696362fab46e4f5ec3bd))

## [0.10.5](https://github.com/Belphemur/night-routine/compare/v0.10.4...v0.10.5) (2025-09-08)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.29.0 ([#87](https://github.com/Belphemur/night-routine/issues/87)) ([c32c5fd](https://github.com/Belphemur/night-routine/commit/c32c5fd6fb165fdc76816fb31560624581b9e475))

## [0.10.4](https://github.com/Belphemur/night-routine/compare/v0.10.3...v0.10.4) (2025-09-07)


### Bug Fixes

* **deps:** update module golang.org/x/oauth2 to v0.31.0 ([#86](https://github.com/Belphemur/night-routine/issues/86)) ([f8ef100](https://github.com/Belphemur/night-routine/commit/f8ef100819bb1ca61b2e06a9de23b88360de7d9c))

## [0.10.3](https://github.com/Belphemur/night-routine/compare/v0.10.2...v0.10.3) (2025-09-03)


### Bug Fixes

* go version used ([4e7ff9e](https://github.com/Belphemur/night-routine/commit/4e7ff9e621eb624ce462ffbd7100ac042fdc0bf8))

## [0.10.2](https://github.com/Belphemur/night-routine/compare/v0.10.1...v0.10.2) (2025-09-03)


### Bug Fixes

* **deps:** update module github.com/stretchr/testify to v1.11.1 ([8bf5e5b](https://github.com/Belphemur/night-routine/commit/8bf5e5b6cbdf792e7ddffa8bfb7ff919a1ae65b3))

## [0.10.1](https://github.com/Belphemur/night-routine/compare/v0.10.0...v0.10.1) (2025-09-03)


### Bug Fixes

* **deps:** update module github.com/golang-migrate/migrate/v4 to v4.19.0 ([9aca299](https://github.com/Belphemur/night-routine/commit/9aca299a6e53db478c2f2cb58945171b007f9c54))

# [0.10.0](https://github.com/Belphemur/night-routine/compare/v0.9.20...v0.10.0) (2025-08-27)


### Features

* use transaction for webhook handler ([e3b14cc](https://github.com/Belphemur/night-routine/commit/e3b14cce22d52b1c915869da36eab8bdf858918e))

## [0.9.20](https://github.com/Belphemur/night-routine/compare/v0.9.19...v0.9.20) (2025-08-24)


### Bug Fixes

* **deps:** update module github.com/stretchr/testify to v1.11.0 ([#77](https://github.com/Belphemur/night-routine/issues/77)) ([c301008](https://github.com/Belphemur/night-routine/commit/c301008ff7a67a0489b3a23e14e4244c75222776))

## [0.9.19](https://github.com/Belphemur/night-routine/compare/v0.9.18...v0.9.19) (2025-08-22)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.28.0 ([#76](https://github.com/Belphemur/night-routine/issues/76)) ([86a31e5](https://github.com/Belphemur/night-routine/commit/86a31e5f9ed27651985c2502fbfed7c4e712e20a))

## [0.9.18](https://github.com/Belphemur/night-routine/compare/v0.9.17...v0.9.18) (2025-08-19)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.248.0 ([#75](https://github.com/Belphemur/night-routine/issues/75)) ([afd99e9](https://github.com/Belphemur/night-routine/commit/afd99e994fa181eaacba9d8e508a4ad94be1387a))

## [0.9.17](https://github.com/Belphemur/night-routine/compare/v0.9.16...v0.9.17) (2025-08-12)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.247.0 ([#72](https://github.com/Belphemur/night-routine/issues/72)) ([c757082](https://github.com/Belphemur/night-routine/commit/c75708253852c25d9796c4926b2f7df92fe742cb))

## [0.9.16](https://github.com/Belphemur/night-routine/compare/v0.9.15...v0.9.16) (2025-08-07)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.246.0 ([#70](https://github.com/Belphemur/night-routine/issues/70)) ([3789409](https://github.com/Belphemur/night-routine/commit/3789409e3ea59b08d264082778d603fe4b55ad58))

## [0.9.15](https://github.com/Belphemur/night-routine/compare/v0.9.14...v0.9.15) (2025-07-22)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.243.0 ([#69](https://github.com/Belphemur/night-routine/issues/69)) ([787175f](https://github.com/Belphemur/night-routine/commit/787175f593f2a0004ba9b1be88b84d9dc87ab963))

## [0.9.14](https://github.com/Belphemur/night-routine/compare/v0.9.13...v0.9.14) (2025-07-18)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.27.1 ([#67](https://github.com/Belphemur/night-routine/issues/67)) ([99b59f2](https://github.com/Belphemur/night-routine/commit/99b59f2177f09ea6b3cdcf46a7b19dee87572c3e))

## [0.9.13](https://github.com/Belphemur/night-routine/compare/v0.9.12...v0.9.13) (2025-07-17)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.27.0 ([#65](https://github.com/Belphemur/night-routine/issues/65)) ([dc854cd](https://github.com/Belphemur/night-routine/commit/dc854cde6fd1447fabd2bfe5adac24e7f3a236bb))

## [0.9.12](https://github.com/Belphemur/night-routine/compare/v0.9.11...v0.9.12) (2025-07-16)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.242.0 ([#64](https://github.com/Belphemur/night-routine/issues/64)) ([5bbc07a](https://github.com/Belphemur/night-routine/commit/5bbc07a47d9f57455b043358801dcaedf6c1cebc))

## [0.9.11](https://github.com/Belphemur/night-routine/compare/v0.9.10...v0.9.11) (2025-07-09)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.241.0 ([#62](https://github.com/Belphemur/night-routine/issues/62)) ([79f264e](https://github.com/Belphemur/night-routine/commit/79f264e6bdf3f587d0f06907c620c8dc1f371e89))

## [0.9.10](https://github.com/Belphemur/night-routine/compare/v0.9.9...v0.9.10) (2025-07-02)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.240.0 ([#61](https://github.com/Belphemur/night-routine/issues/61)) ([7e9107d](https://github.com/Belphemur/night-routine/commit/7e9107dd443c6761bb2046a8ca12dd91c0909e85))

## [0.9.9](https://github.com/Belphemur/night-routine/compare/v0.9.8...v0.9.9) (2025-06-30)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.26.3 ([#59](https://github.com/Belphemur/night-routine/issues/59)) ([e17b187](https://github.com/Belphemur/night-routine/commit/e17b18795b451b5a00e67ff8fa57baf788e8afa0))

## [0.9.8](https://github.com/Belphemur/night-routine/compare/v0.9.7...v0.9.8) (2025-06-25)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.239.0 ([#57](https://github.com/Belphemur/night-routine/issues/57)) ([7c88870](https://github.com/Belphemur/night-routine/commit/7c88870aa4770890773a274905b416b43b02bf9e))

## [0.9.7](https://github.com/Belphemur/night-routine/compare/v0.9.6...v0.9.7) (2025-06-24)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.26.2 ([#56](https://github.com/Belphemur/night-routine/issues/56)) ([eb22f30](https://github.com/Belphemur/night-routine/commit/eb22f30430da1e96d4942a1b214d4cdd4682c73b))

## [0.9.6](https://github.com/Belphemur/night-routine/compare/v0.9.5...v0.9.6) (2025-06-18)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.238.0 ([#54](https://github.com/Belphemur/night-routine/issues/54)) ([52f618b](https://github.com/Belphemur/night-routine/commit/52f618b5dacce484728569f4cba13f39cf281884))

## [0.9.5](https://github.com/Belphemur/night-routine/compare/v0.9.4...v0.9.5) (2025-06-13)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.237.0 ([#51](https://github.com/Belphemur/night-routine/issues/51)) ([3f29a75](https://github.com/Belphemur/night-routine/commit/3f29a751eda55168f4613825e59798f359dfae4e))

## [0.9.4](https://github.com/Belphemur/night-routine/compare/v0.9.3...v0.9.4) (2025-06-08)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.26.1 ([#50](https://github.com/Belphemur/night-routine/issues/50)) ([b06da57](https://github.com/Belphemur/night-routine/commit/b06da5738f6066ae59d747786999a416ace69556))

## [0.9.3](https://github.com/Belphemur/night-routine/compare/v0.9.2...v0.9.3) (2025-06-06)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.236.0 ([#49](https://github.com/Belphemur/night-routine/issues/49)) ([bde7904](https://github.com/Belphemur/night-routine/commit/bde7904203e388237d6f92b7f56aa56fc699e585))

## [0.9.2](https://github.com/Belphemur/night-routine/compare/v0.9.1...v0.9.2) (2025-05-31)


### Bug Fixes

* **deps:** update module github.com/ncruces/go-sqlite3 to v0.26.0 ([#48](https://github.com/Belphemur/night-routine/issues/48)) ([14dfc38](https://github.com/Belphemur/night-routine/commit/14dfc38488aa3a1b181be5bf9d2a283a347c7592))

## [0.9.1](https://github.com/Belphemur/night-routine/compare/v0.9.0...v0.9.1) (2025-05-28)


### Bug Fixes

* **deps:** update module google.golang.org/api to v0.235.0 ([#47](https://github.com/Belphemur/night-routine/issues/47)) ([3615ecd](https://github.com/Belphemur/night-routine/commit/3615ecd44a14eeb076b75d25f83df1d6c2748e85))

# [0.9.0](https://github.com/Belphemur/night-routine/compare/v0.8.4...v0.9.0) (2025-05-24)


### Bug Fixes

* explicitly disable all calendar event reminders ([1da2343](https://github.com/Belphemur/night-routine/commit/1da23438596db24842671911bbc46b5b8df88ec1))


### Features

* add assignment reason to calendar event description and remove reminders ([6c44afe](https://github.com/Belphemur/night-routine/commit/6c44afefc67927cef420bb569917161b2f74837b))

## [0.8.4](https://github.com/Belphemur/night-routine/compare/v0.8.3...v0.8.4) (2025-05-24)


### Bug Fixes

* set GITHUB_TOKEN for attest build provenance step ([519d6ef](https://github.com/Belphemur/night-routine/commit/519d6ef676d3a9250f41501458224ec6c2c25005))

## [0.8.3](https://github.com/Belphemur/night-routine/compare/v0.8.2...v0.8.3) (2025-05-24)


### Bug Fixes

* remove GITHUB_TOKEN environment variable from attest build provenance step ([03c9d9e](https://github.com/Belphemur/night-routine/commit/03c9d9eb8d53b4fe366504eb796681f3da60b58d))

## [0.8.2](https://github.com/Belphemur/night-routine/compare/v0.8.1...v0.8.2) (2025-05-24)


### Bug Fixes

* update permissions and use GITHUB_TOKEN for authentication in release workflow ([5b5d445](https://github.com/Belphemur/night-routine/commit/5b5d445411c31096feb35fb664ca1deaf6319686))

## [0.8.1](https://github.com/Belphemur/night-routine/compare/v0.8.0...v0.8.1) (2025-05-24)


### Bug Fixes

* add SSH key for checkout in release workflow and update .gitignore to include node_modules ([fa12dc1](https://github.com/Belphemur/night-routine/commit/fa12dc177f4405f61d94a9b1c573257e4ebb206f))
* **deps:** update module google.golang.org/api to v0.233.0 ([#41](https://github.com/Belphemur/night-routine/issues/41)) ([5959018](https://github.com/Belphemur/night-routine/commit/59590184f4e3ebbdc0f5cea55a8c161fd492b825))
* **deps:** update module google.golang.org/api to v0.234.0 ([#43](https://github.com/Belphemur/night-routine/issues/43)) ([62337ef](https://github.com/Belphemur/night-routine/commit/62337ef950aec44a5f5b0029c94ade7c30321aee))
* enable gpg signing for releases ([0610ede](https://github.com/Belphemur/night-routine/commit/0610edefe603740d3c535b3e1daf61dfe82b161d))
* implement semantic-release with goreleaser ([32ef50a](https://github.com/Belphemur/night-routine/commit/32ef50a58fa67de7d7b31104b21ab7a92e3fa21b))
* update GoReleaser CLI installation to install only without running ([1360d83](https://github.com/Belphemur/night-routine/commit/1360d83efc8b8b427f3172c37a04acf8f47204cd))
* use npm ci and npx for semantic-release ([5fb3ab9](https://github.com/Belphemur/night-routine/commit/5fb3ab9594446079655fd6a3dfd7a5345e5049eb))
