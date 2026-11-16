package desktop

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPickerResultJSONIsGeneric(t *testing.T) {
	data, err := json.Marshal(FileSelection{Path: `C:\Users\Example\report.txt`})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"path":"C:\\Users\\Example\\report.txt"}`; got != want {
		t.Fatalf("serialized selection = %s, want %s", got, want)
	}
	folders, err := json.Marshal(FoldersSelection{Paths: []string{}})
	if err != nil || string(folders) != `{"paths":[]}` {
		t.Fatalf("serialized empty multi-selection = %s, %v", folders, err)
	}
}

func TestIsDialogCancellation(t *testing.T) {
	failure := errors.New("dialog failed")
	for _, test := range []struct {
		name string
		hr   uintptr
		err  error
		want bool
	}{
		{"cancel", dialogCanceledHRESULT, failure, true},
		{"other failure", 0x80004005, failure, false},
		{"successful show", 0, nil, false},
		{"cancel code without failure", dialogCanceledHRESULT, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := IsDialogCancellation(test.hr, test.err); got != test.want {
				t.Fatalf("IsDialogCancellation() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	if _, err := ValidateFilePath(""); err == nil {
		t.Fatal("empty successful result was accepted")
	}
	if got, err := ValidateFilePath("selected.txt"); err != nil || got != "selected.txt" {
		t.Fatalf("ValidateFilePath() = %q, %v", got, err)
	}
}
