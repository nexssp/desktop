package nexssflow

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/nexssp/flow/compiler"
	"github.com/nexssp/kernel/action"
	"github.com/nexssp/kernel/xctx"
	"github.com/nexssp/kernel/xerr"

	"github.com/nexssp/desktop"
)

type recordingPicker struct {
	fileRequest   desktop.FilePickerRequest
	folderRequest desktop.FolderPickerRequest
	ctxRequestID  string
	ctxEndpoint   string
	fileErr       error
	called        string
}

func (p *recordingPicker) PickFile(ctx context.Context, request desktop.FilePickerRequest) (desktop.FileSelection, error) {
	p.called = "file"
	p.fileRequest = request
	p.ctxRequestID = xctx.RequestIDFrom(ctx)
	p.ctxEndpoint = xctx.EndpointFrom(ctx)
	return desktop.FileSelection{Path: "selected.txt"}, p.fileErr
}

func (p *recordingPicker) PickFolder(ctx context.Context, request desktop.FolderPickerRequest) (desktop.FolderSelection, error) {
	p.called = "folder"
	p.folderRequest = request
	p.ctxRequestID = xctx.RequestIDFrom(ctx)
	p.ctxEndpoint = xctx.EndpointFrom(ctx)
	return desktop.FolderSelection{Path: "folder"}, p.fileErr
}

func (p *recordingPicker) PickFolders(ctx context.Context, request desktop.FolderPickerRequest) (desktop.FoldersSelection, error) {
	p.called = "folders"
	p.folderRequest = request
	p.ctxRequestID = xctx.RequestIDFrom(ctx)
	p.ctxEndpoint = xctx.EndpointFrom(ctx)
	return desktop.FoldersSelection{Paths: []string{"one", "two"}}, p.fileErr
}

func TestNewBundleExposesGenericDesktopActions(t *testing.T) {
	bundle, err := NewBundle(&recordingPicker{})
	if err != nil {
		t.Fatal(err)
	}
	if err := compiler.ValidateBundle(bundle); err != nil {
		t.Fatalf("ValidateBundle() error = %v", err)
	}
	if bundle.ID != "desktop" || len(bundle.Libraries) != 1 || bundle.Libraries[0].Name != "desktop" {
		t.Fatalf("bundle identity = %#v", bundle)
	}
	want := []string{"nexss.desktop.pick_file", "nexss.desktop.pick_folder", "nexss.desktop.pick_folders"}
	got := make([]string, 0, len(bundle.Libraries[0].Actions))
	for _, act := range bundle.Libraries[0].Actions {
		got = append(got, act.Describe().Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("action names = %v, want %v", got, want)
	}
	if _, err := NewBundle(nil); xerr.KindFrom(err) != xerr.KindValidation {
		t.Fatalf("nil picker error = %v, want validation error", err)
	}
}

func TestPickFileActionUsesInjectedServiceAndRequestContext(t *testing.T) {
	picker := &recordingPicker{}
	bundle, err := NewBundle(picker)
	if err != nil {
		t.Fatal(err)
	}
	act := actionByName(t, bundle, "nexss.desktop.pick_file")
	ctx, _, cleanup := xctx.NewScope(context.Background())
	defer cleanup()
	ctx = xctx.WithRequestID(ctx, "request-17")
	request := PickFileRequest{
		Title:            "Choose a report",
		InitialDirectory: "/tmp",
		Filters:          []desktop.FileFilter{{Name: "Text documents", Pattern: "*.txt"}},
	}
	value, err := act.DoAny(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if value != (desktop.FileSelection{Path: "selected.txt"}) {
		t.Fatalf("action result = %#v", value)
	}
	if picker.called != "file" || picker.fileRequest.Title != request.Title || picker.fileRequest.InitialDirectory != request.InitialDirectory || !reflect.DeepEqual(picker.fileRequest.Filters, request.Filters) {
		t.Fatalf("injected service call = %#v", picker)
	}
	if picker.ctxRequestID != "request-17" || picker.ctxEndpoint != "nexss.desktop.pick_file" {
		t.Fatalf("service request context = request %q, endpoint %q", picker.ctxRequestID, picker.ctxEndpoint)
	}
}

func TestPickFileRejectsInvalidFilterBeforeCallingService(t *testing.T) {
	picker := &recordingPicker{}
	bundle, err := NewBundle(picker)
	if err != nil {
		t.Fatal(err)
	}
	_, err = actionByName(t, bundle, "nexss.desktop.pick_file").DoAny(context.Background(), PickFileRequest{
		Filters: []desktop.FileFilter{{Name: "Missing pattern"}},
	})
	if xerr.KindFrom(err) != xerr.KindBadRequest || picker.called != "" {
		t.Fatalf("error = %v, picker called = %q; want bad request without service call", err, picker.called)
	}
}

func TestPickerErrorsUseKernelClassification(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want xerr.Kind
	}{
		{"unsupported", desktop.ErrUnsupported, xerr.KindUnavailable},
		{"canceled", context.Canceled, xerr.KindCanceled},
		{"deadline", context.DeadlineExceeded, xerr.KindTimeout},
		{"unexpected", errors.New("unexpected native failure"), xerr.KindInternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := xerr.KindFrom(mapPickerError("file", test.err)); got != test.want {
				t.Fatalf("error kind = %q, want %q", got, test.want)
			}
		})
	}
}

func actionByName(t *testing.T, bundle compiler.Bundle, name string) action.AnyAction {
	t.Helper()
	for libraryIndex := range bundle.Libraries {
		lib := bundle.Libraries[libraryIndex]
		for _, act := range lib.Actions {
			if act.Describe().Name == name {
				return act
			}
		}
	}
	t.Fatalf("action %q not found", name)
	return nil
}
