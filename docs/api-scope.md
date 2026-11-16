# API scope

## Core package

`github.com/nexssp/desktop` is independent of Flow and Kernel at the source-import level. Its APIs are deliberately parameterized:

- `ListenLoopback(context.Context, ListenerOptions)` binds only a literal loopback IP and persists a selected port to the supplied file.
- `OpenLog(LogOptions)` creates a caller-named log in a caller-selected directory and applies the supplied rotation policy.
- `ParseShortcut` validates a key combination on any platform. `EnableWindowShortcut` and `SetWindowShortcut` provide Windows process-window activation; unsupported platforms return `ErrUnsupported` when enabling the capability.
- `NativePicker` accepts typed file/folder requests containing optional title, starting directory, and file filters. Cancellation is represented by empty selections, not by an error.
- Placement types and functions validate, normalize, load, and save geometry using caller-provided paths, work areas, and defaults.

The core does not declare application routes, action names, port numbers, configuration locations, log filenames, browser settings, UI sizes, or product policies.

## Optional `nexssflow` package

`github.com/nexssp/desktop/nexssflow` is the only package in this module that imports `github.com/nexssp/flow` and `github.com/nexssp/kernel`. It provides a registered `desktop` bundle and a `NewBundle` constructor for injected `desktop.Picker` implementations. Its public action names are `nexss.desktop.pick_file`, `nexss.desktop.pick_folder`, and `nexss.desktop.pick_folders`.

Each action accepts a typed request, propagates the action context, fills an endpoint only when one is not already present, and adds a trace event. Expected cancellation, timeout, unsupported-platform, invalid-request, and internal failures map to Kernel error kinds. The adapter contains no app identity, route configuration, authorization policy, host lifecycle, storage, or feature-specific action.

## Compatibility and stability

The first release is intended to be tagged `v0.1.0` after owner review. Keep later API changes additive where practical. An application-specific wrapper may continue to expose its existing API and supply its own filenames, defaults, routes, error wording, and business rules while forwarding mechanics to this core.
