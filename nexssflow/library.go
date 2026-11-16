// Package nexssflow exposes optional generic desktop-picker actions to NFlow.
// Importing this package is not required by applications that use the desktop
// core directly.
package nexssflow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nexssp/flow/compiler"
	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xctx"
	"github.com/nexssp/kernel/xerr"

	"github.com/nexssp/desktop"
)

// ID is the optional Flow bundle identifier.
const ID = "desktop"

// Picker is the injected core service used by the adapter.
type Picker = desktop.Picker

// PickFileRequest is the typed input accepted by nexss.desktop.pick_file.
type PickFileRequest struct {
	Title            string               `json:"title,omitempty"`
	InitialDirectory string               `json:"initial_directory,omitempty"`
	Filters          []desktop.FileFilter `json:"filters,omitempty"`
}

// PickFolderRequest is the typed input accepted by nexss.desktop.pick_folder
// and nexss.desktop.pick_folders.
type PickFolderRequest struct {
	Title            string `json:"title,omitempty"`
	InitialDirectory string `json:"initial_directory,omitempty"`
}

// Bundle constructs a Flow bundle backed by the current platform's native
// desktop dialogs. The bundle accepts no @require options.
func Bundle(_ map[string]string) compiler.Bundle {
	return bundle(desktop.NativePicker{})
}

// NewBundle constructs a Flow bundle backed by the supplied desktop service.
// Hosts can inject mocks or policy-aware native implementations for tests and
// controlled deployments.
func NewBundle(picker Picker) (compiler.Bundle, error) {
	if picker == nil {
		return compiler.Bundle{}, xerr.Validation("desktop: picker service is required")
	}
	return bundle(picker), nil
}

func init() { compiler.Register(ID, Bundle) }

func bundle(picker Picker) compiler.Bundle {
	return compiler.Bundle{
		ID: ID,
		Libraries: []action.Library{{
			Name:        "desktop",
			Description: "Native desktop selection actions",
			Actions: []action.AnyAction{
				newPickFileAction(picker),
				newPickFolderAction(picker),
				newPickFoldersAction(picker),
			},
		}},
	}
}

func newPickFileAction(picker Picker) action.AnyAction {
	return action.New("nexss.desktop.pick_file", func(ctx context.Context, request PickFileRequest) (desktop.FileSelection, error) {
		if err := validateFileRequest(request); err != nil {
			return desktop.FileSelection{}, xerr.BadRequest("desktop: invalid file picker request", err)
		}
		ctx = actionContext(ctx, "nexss.desktop.pick_file")
		result, err := picker.PickFile(ctx, desktop.FilePickerRequest{
			Title: request.Title, InitialDirectory: request.InitialDirectory, Filters: request.Filters,
		})
		if err != nil {
			return desktop.FileSelection{}, mapPickerError("file", err)
		}
		return result, nil
	}).Description("Open a native file picker and return the selected path.").Tag("desktop", "picker").Build()
}

func newPickFolderAction(picker Picker) action.AnyAction {
	return action.New("nexss.desktop.pick_folder", func(ctx context.Context, request PickFolderRequest) (desktop.FolderSelection, error) {
		ctx = actionContext(ctx, "nexss.desktop.pick_folder")
		result, err := picker.PickFolder(ctx, desktop.FolderPickerRequest{
			Title: request.Title, InitialDirectory: request.InitialDirectory,
		})
		if err != nil {
			return desktop.FolderSelection{}, mapPickerError("folder", err)
		}
		return result, nil
	}).Description("Open a native folder picker and return the selected path.").Tag("desktop", "picker").Build()
}

func newPickFoldersAction(picker Picker) action.AnyAction {
	return action.New("nexss.desktop.pick_folders", func(ctx context.Context, request PickFolderRequest) (desktop.FoldersSelection, error) {
		ctx = actionContext(ctx, "nexss.desktop.pick_folders")
		result, err := picker.PickFolders(ctx, desktop.FolderPickerRequest{
			Title: request.Title, InitialDirectory: request.InitialDirectory,
		})
		if err != nil {
			return desktop.FoldersSelection{}, mapPickerError("multi-folder", err)
		}
		if result.Paths == nil {
			result.Paths = []string{}
		}
		return result, nil
	}).Description("Open a native multi-folder picker and return the selected paths.").Tag("desktop", "picker").Build()
}

func validateFileRequest(request PickFileRequest) error {
	for index, filter := range request.Filters {
		if strings.TrimSpace(filter.Name) == "" || strings.TrimSpace(filter.Pattern) == "" {
			return fmt.Errorf("filter %d requires a name and pattern", index)
		}
	}
	return nil
}

func actionContext(ctx context.Context, endpoint string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if xctx.EndpointFrom(ctx) == "" {
		ctx = xctx.WithEndpoint(ctx, endpoint)
	}
	xctx.AddTrace(ctx, endpoint)
	return ctx
}

func mapPickerError(kind string, err error) error {
	if errors.Is(err, context.Canceled) {
		return xerr.Canceled("desktop picker request canceled", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return xerr.Timeout("desktop picker request timed out", err)
	}
	if errors.Is(err, desktop.ErrUnsupported) {
		return xerr.Unavailable("native desktop picker is unavailable", err)
	}
	if _, ok := errors.AsType[*xerr.AppError](err); ok {
		return err
	}
	return xerr.Internal("native desktop "+kind+" picker failed", err)
}
