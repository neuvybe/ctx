# [0.4.0](https://github.com/neuvybe/ctx/compare/v0.3.0...v0.4.0) (2026-10-07)


* feat!: include behavior context by default ([#3](https://github.com/neuvybe/ctx/issues/3)) ([5c3b88e](https://github.com/neuvybe/ctx/commit/5c3b88ea08d079239ad4798f72c601d8afa6c163))


### BREAKING CHANGES

* New layout-v2 scaffolds select behavior and glossary by default.
Use --without behavior,glossary, or an explicit empty InitOptions.Addons slice,
for the fixed core only. Existing scaffolds retain their selected add-ons;
after upgrading ctx, run ctx update followed by ctx add behavior to adopt
the new document.

# [0.3.0](https://github.com/neuvybe/ctx/compare/v0.2.0...v0.3.0) (2026-09-01)


* feat!: introduce lean context layout v2 ([999234a](https://github.com/neuvybe/ctx/commit/999234abf3995b5415c204faa4f64076932d30fd))


### Bug Fixes

* reject symlinked status evidence ([074b007](https://github.com/neuvybe/ctx/commit/074b007f6654984923551fe7c8562d1b4654835f))


### BREAKING CHANGES

* New scaffolds use layout v2; InitOptions and Config changed.

# [0.2.0](https://github.com/neuvybe/ctx/compare/v0.1.2...v0.2.0) (2026-08-31)


* feat!: make team mode the default ([448f7b2](https://github.com/neuvybe/ctx/commit/448f7b22660a759428698fdc03cc253fb416260f))


### BREAKING CHANGES

* Team mode is now the default for new scaffolds.

## [0.1.2](https://github.com/neuvybe/ctx/compare/v0.1.1...v0.1.2) (2026-08-29)


### Bug Fixes

* ship npm/bin/ctx.js — gitignore 'bin/' was excluding it ([5c3a053](https://github.com/neuvybe/ctx/commit/5c3a053c8c6dc858cc97438da6714fa776c31f13))

## [0.1.1](https://github.com/neuvybe/ctx/compare/v0.1.0...v0.1.1) (2026-08-29)


### Bug Fixes

* launcher postinstall tar ESM import + drop macOS AppleDouble from archives ([e54b8c7](https://github.com/neuvybe/ctx/commit/e54b8c7902832204aebcafa2f16d2609d5dec1c5))
