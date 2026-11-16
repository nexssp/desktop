# Contributing

Keep changes app-neutral and parameterized. Do not add consuming-application routes, action prefixes, settings, filenames, storage paths, ports, UI defaults, or product workflows to this module. Flow integration belongs in `nexssflow/`; the core `desktop` package must remain free of Flow and Kernel imports.

Before submitting changes, run:

```sh
task tools
task ci
```

Native dialog and window activation tests should use injected services or platform-neutral helpers rather than opening interactive UI. Preserve low-permission CI and the Windows cross-compile gate.
