package vdf

import (
	"bytes"
	"testing"
)

func TestParsingCompleteVM(t *testing.T) {
	var content = []byte(`[BEGINVDF]
Comment=This is a comment for the VDF that will be generated
BaseDir=.\
VDFName=.\Demo.vdf
[FILES]
_Work\* -r
* -r
[EXCLUDE]
DESKTOP.INI -r
*.vdf -r
*.vm
*.exe
[INCLUDE]
Demo_Original.vdf -r
[ENDVDF]
`)

	vm, err := parseVM(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse VM. %v", err)
	}

	assertEqual(t, vm.BaseDir, `.\`)
	assertEqual(t, vm.VDFName, `.\Demo.vdf`)

	assertCount(t, vm.Files, 2)
	assertEqual(t, vm.Files[0], `_Work\* -r`)
	assertEqual(t, vm.Files[1], `* -r`)

	assertCount(t, vm.Exclude, 4)
	assertEqual(t, vm.Exclude[0], `DESKTOP.INI -r`)
	assertEqual(t, vm.Exclude[1], `*.vdf -r`)
	assertEqual(t, vm.Exclude[2], `*.vm`)
	assertEqual(t, vm.Exclude[3], `*.exe`)

	assertCount(t, vm.Include, 1)
	assertEqual(t, vm.Include[0], `Demo_Original.vdf -r`)
}

func TestParsing(t *testing.T) {
	var content = []byte(`[BEGINVDF]
BaseDir=BaseDir
VDFName=VDFName.vdf
[ENDVDF]`)

	vm, err := parseVM(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse VM. %v", err)
	}

	assertEqual(t, vm.BaseDir, "BaseDir")
	assertEqual(t, vm.VDFName, "VDFName.vdf")
}

func TestCommentSupportsNewLineEscapes(t *testing.T) {
	var content = []byte(`[BEGINVDF]
Comment=Comment=%%NWith%%NNewLines
[ENDVDF]`)

	vm, err := parseVM(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse VM. %v", err)
	}

	assertEqualf(t, vm.Comment, "Comment=\r\nWith\r\nNewLines", "comment should transform newlines")
}

func TestParsingIgnoresWhiteSpaceForSections(t *testing.T) {
	var content = []byte("\t[BEGINVDF]    \n    [ENDVDF]\t")

	_, err := parseVM(bytes.NewReader(content))
	assertEqual(t, nil, err)
}

func assertEqual[T comparable](t *testing.T, left, right T) {
	t.Helper()
	if left != right {
		t.Errorf("%v != %v", left, right)
		t.Fail()
	}
}

func assertEqualf[T comparable](t *testing.T, left, right T, format string, args ...any) {
	t.Helper()
	if left != right {
		t.Errorf(format, args...)
		t.Fail()
	}
}

func assertCount[T any](t *testing.T, slice []T, count int) {
	t.Helper()
	if len(slice) != count {
		t.Errorf("Expected %d items, but got %d", count, len(slice))
		t.Fail()
	}
}
