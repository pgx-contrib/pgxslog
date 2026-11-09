# Changelog

## 1.0.0 (2026-11-09)


### Features

* add comprehensive test suite, ci/cd, and dev environment ([5c7f0c4](https://github.com/pgx-contrib/pgxslog/commit/5c7f0c44083a855d8facb5c8c8f701c7d4457c9a))
* **ci:** read the ralch Nix cache ([#68](https://github.com/pgx-contrib/pgxslog/issues/68)) ([16129e7](https://github.com/pgx-contrib/pgxslog/commit/16129e79a85f176699c8a923a383e86b6a2e3e4f))
* **logger:** add ContextKey and LoggerKey for context handling ([b7ec761](https://github.com/pgx-contrib/pgxslog/commit/b7ec761369e4f003a927cc9328c320951d1478d3))


### Bug Fixes

* check for non-nil ContextKey before retrieving logger from context ([b7c2a0b](https://github.com/pgx-contrib/pgxslog/commit/b7c2a0b3e13d47af48da37f76a0343fcbfbb5e22))
* **ci:** merge with the App token so main gets its CI run ([#67](https://github.com/pgx-contrib/pgxslog/issues/67)) ([c1720ef](https://github.com/pgx-contrib/pgxslog/commit/c1720efbeb6ceedb40901f846fee8724e256f745))
* **github:** correct action versions in update.yml ([ef7fd26](https://github.com/pgx-contrib/pgxslog/commit/ef7fd2687a6e42d4aa24402bf8804b8dae6961fd))
* handle acronyms correctly in ConvertAttr (SQL → sql, SQLQuery → sql_query) ([522b836](https://github.com/pgx-contrib/pgxslog/commit/522b8369f5d49d27592b5cefac3981c5d4b80a35))
