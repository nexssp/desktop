# Desktop NFlow examples

These examples demonstrate the **three actions actually registered** by the optional desktop Flow adapter: `nexss.desktop.pick_file`, `nexss.desktop.pick_folder`, and `nexss.desktop.pick_folders`. The complete Go/API versus NFlow feature list is in [`docs/feature-inventory.md`](../../docs/feature-inventory.md).

## Prerequisites

- Nexss Flow CLI **v0.80.1**, the Flow version declared by desktop `v0.1.0`.
- A Go toolchain satisfying the module's `go 1.27.4` requirement. Flow's `@require` builds a Go harness for the adapter when needed.
- Network access on the first run so Go can resolve `github.com/nexssp/desktop/nexssflow` at tag `v0.1.0` and any uncached dependencies.
- Windows with an interactive desktop session for the default native dialogs. The adapter's default `Bundle` uses `desktop.NativePicker`; on other platforms the picker actions return an unavailable error. A custom Go Flow host can instead register `nexssflow.NewBundle` with its own supported `Picker` implementation.

Each example pins the adapter dependency to the released `v0.1.0` tag. Run them from the repository root:

```sh
nflow run nexssflow/nflows/picker.nflow
nflow run nexssflow/nflows/pick-folder.nflow
nflow run nexssflow/nflows/pick-folders.nflow
nflow run nexssflow/nflows/prepare-file-copy.nflow
```

The first three examples each open one dialog and return a small projection of the typed picker result. `prepare-file-copy.nflow` opens a file dialog and, only if a file was selected, a folder dialog; it returns a plan and **does not copy or write any files**.

## Cancellation and errors

Closing a native dialog is a normal result, not an error. The file and folder actions return an empty `path`; the multiple-folder action returns an empty `paths` list. The simple examples expose this as `cancelled: true`, and the composed example reports which selection step was canceled. They deliberately do not use assertions that would turn a normal cancellation into a failed run.

Other failures still propagate: for example, an unsupported platform is reported as unavailable, and canceled/timed-out action contexts are typed errors. File filters must have nonblank `name` and `pattern` values. These examples are interactive and should not be run in unattended CI or `nflow test`.

`nflow lint` can validate a fixture without opening its dialog; `nflow run` is the command that opens the picker. For API details and platform boundaries, see [`docs/feature-inventory.md`](../../docs/feature-inventory.md).
