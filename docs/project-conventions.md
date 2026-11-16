# Project conventions

- Go module: `github.com/nexssp/desktop`, Go `1.27.4`.
- Taskfile v3 is the local interface for build, test, lint, vulnerability, tidy, race and cross-platform checks.
- golangci-lint v2 is configured in `.golangci.yml`; Go source is formatted with gofmt and the configured import/format rules.
- CI uses read-only repository permissions, verifies module tidy state, builds and tests all packages, scans with govulncheck, and compiles Windows/amd64 test binaries.
- Keep platform implementations behind build tags and preserve portable validation APIs and explicit unsupported-platform errors.
- Keep the core free of Flow/Kernel imports. Put Flow-specific registration and typed actions only in `nexssflow/`.
- Keep all app configuration, route, UI, log filename, port, and workflow policy in consuming applications; accept those choices as function arguments when the shared package needs them.
- Tests should exercise boundary behavior and typed requests without opening native dialogs or registering operating-system shortcuts.
