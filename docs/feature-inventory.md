# Desktop feature inventory — v0.1.0

This inventory describes the public repository at tag `v0.1.0`. The main distinction is important: **exported Go APIs are not automatically NFlow actions**. The core package is app-neutral; the optional `nexssflow` adapter explicitly registers only the three picker actions listed below.

## Core Go features

### 1. Loopback TCP listener with saved-port reuse

`ListenLoopback` binds a TCP listener only to a literal loopback IP (IPv4 or IPv6). `ListenerOptions` supplies the address and caller-owned port-file path. On first use, the chosen port is saved with owner-only permissions; later calls try to reuse that port. If the saved port is occupied, the call returns `PortInUseError` / `ErrPortInUse` rather than silently choosing another port. The function checks context cancellation and validates the address, port, and file path. It does not create missing parent directories.

Public API: `ListenerOptions`, `PortInUseError`, `ErrPortInUse`, `ListenLoopback` — [implementation](../loopback.go), [tests](../loopback_test.go).

### 2. Caller-owned log files and rotation

`OpenLog` creates a log directory and opens a caller-named file for append. `LogOptions` lets the caller choose its directory, filename, rotation threshold, backup count, and file/directory modes. Zero permission modes default to `0700` for the directory and `0600` for the file. When the active log reaches the configured size, numbered backups are rotated before the new file is opened; with zero backups, the old file is removed. The returned `*os.File` is owned and closed by the caller.

Public API: `LogOptions`, `OpenLog` — [implementation](../logging.go), [tests](../logging_test.go).

### 3. Native file and folder pickers

The `Picker` interface and `NativePicker` provide three operations: choose one file, one folder, or multiple folders. Requests can include a dialog title and initial directory; file requests can also include named filters. On Windows, the implementation uses native dialogs and returns filesystem paths. On platforms without an implementation, the calls return `ErrUnsupported`.

A user closing a dialog is a normal selection result, not an error: a canceled single selection has an empty `Path`, and a canceled multi-folder selection has an empty `Paths` slice. By contrast, context cancellation, timeouts, unsupported platforms, and dialog failures are errors. The context is checked before native dialog work; the implementation does not show cancellation polling while a dialog remains open.

Public API: `Picker`, `NativePicker`, `FileFilter`, `FilePickerRequest`, `FolderPickerRequest`, `FileSelection`, `FolderSelection`, `FoldersSelection`, `ErrUnsupported`, `IsDialogCancellation`, `ValidateFilePath` — [types and helpers](../picker.go), [Windows implementation](../picker_windows.go), [other-platform implementation](../picker_other.go), [tests](../picker_test.go) and [unsupported-platform tests](../picker_other_test.go).

### 4. Shortcut parsing and Windows window activation

`ParseShortcut` validates and canonicalizes key combinations using Ctrl, Alt, and Shift plus supported letters, digits, F1–F24, and common navigation keys. On Windows, `EnableWindowShortcut` registers a global hotkey for a target process and activates one of that process's visible windows when pressed. `SetWindowShortcut` can replace the active shortcut. On non-Windows platforms, enabling returns `ErrUnsupported`; setting is a no-op.

Public API: `Shortcut`, `ModifierAlt`, `ModifierControl`, `ModifierShift`, `ParseShortcut`, `EnableWindowShortcut`, `SetWindowShortcut` — [portable parser](../shortcut.go), [Windows implementation](../shortcut_windows.go), [other-platform implementation](../shortcut_other.go), [tests](../shortcut_test.go).

### 5. Window placement validation and persistence

The placement API models a window rectangle, monitor work area, default size, and size/visibility limits. It can validate dimensions and coordinates, determine whether a saved window is sufficiently visible, restore and clamp a usable placement, or center a fallback placement. `LoadPlacement` reads versioned JSON from a caller-selected path; `SavePlacement` writes version-1 JSON through a temporary file and rename, creating directories as needed.

Public API: `Placement`, `ScreenRect`, `WindowSize`, `PlacementLimits`, `DefaultPlacementLimits`, `ValidPlacementSize`, `ValidPlacementPosition`, `PlacementVisibleInWorkArea`, `NormalizePlacement`, `LoadPlacement`, `SavePlacement` — [implementation](../window_placement.go), [tests](../window_placement_test.go).

## The actual NFlow surface

The optional package [registers bundle `desktop`](../nexssflow/library.go#L20-L71). `Bundle` uses the current platform's `NativePicker`; `NewBundle` lets a Go host inject a `Picker` implementation. The bundle registers exactly these actions:

| NFlow action | Typed input | Typed output |
|---|---|---|
| `nexss.desktop.pick_file` | `PickFileRequest`: `title`, `initial_directory`, and optional `filters` (`name` and `pattern`) | `desktop.FileSelection`: `path` |
| `nexss.desktop.pick_folder` | `PickFolderRequest`: `title`, `initial_directory` | `desktop.FolderSelection`: `path` |
| `nexss.desktop.pick_folders` | `PickFolderRequest`: `title`, `initial_directory` | `desktop.FoldersSelection`: `paths` |

The file-picker adapter rejects filters whose name or pattern is blank. All three actions preserve a supplied action endpoint, add their trace entry, and classify returned errors: context cancellation as `Canceled`, deadline as `Timeout`, unsupported native picker as `Unavailable`, invalid file filters as `BadRequest`, and other picker failures as `Internal`. The multi-folder action normalizes a nil `paths` result to an empty list. A user dismissing the Windows dialog is represented by the empty selection described above, so examples should not treat that as an action error.

No loopback, logging, shortcut, or window-placement action is registered by this adapter. Those remain Go APIs unless a future adapter explicitly registers them. See [adapter implementation](../nexssflow/library.go) and [adapter tests](../nexssflow/library_test.go).

## Examples and verification boundaries

The runnable interactive fixtures are in [`nexssflow/nflows/`](../nexssflow/nflows/README.md). They exercise every registered action, show empty-selection cancellation handling, and include one two-step example that returns a file-to-folder plan without performing a copy. They open native dialogs, so they are manual desktop examples rather than unattended CI tests.

The repository also has focused Go tests for each core area and for the adapter; their exact coverage is in the linked test files. The Windows native-dialog and hotkey operations are platform-specific and are not exercised by the portable picker/shortcut parser tests.

The module declares Go `1.27.4` and uses Flow `0.80.1` (`go.mod`). Repository quality checks are defined in [Taskfile.yml](../Taskfile.yml) and CI invokes the repository gate in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).
